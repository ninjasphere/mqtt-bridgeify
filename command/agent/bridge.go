package agent

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/juju/loggo"
)

var AlreadyConfigured = errors.New("Already configured")
var AlreadyUnConfigured = errors.New("Already unconfigured")

//
// Acts as a bridge between local and cloud brokers, this includes reconnecting
// and emitting status changes.
//
// Once configured and started this will attempt to connect to connect to local
// and cloud brokers, if something dies it will reconnect based on the configured
// reconnect backoff.
//
type Bridge struct {
	conf   *Config
	local  *mqtt.MqttClient
	remote *mqtt.MqttClient
	log    loggo.Logger

	localTopics []replaceTopic
	cloudTopics []replaceTopic

	cloudUrl *url.URL
	token    string

	timer       *time.Timer
	reconnectCh chan bool
	shutdownCh  chan bool

	Configured bool
	Connected  bool
	Counter    int64

	LastError error

	bridgeLock sync.Mutex
}

type replaceTopic struct {
	on      string
	replace string
	with    string
}

func (r *replaceTopic) updated(originalTopic string) string {
	return strings.Replace(originalTopic, r.replace, r.with, 1)
}

var localTopics = []replaceTopic{
	{on: "$location/calibration", replace: "$location", with: "$cloud/location"},
	{on: "$location/delete", replace: "$location", with: "$cloud/location"},
	{on: "$device/+/+/rssi", replace: "$device", with: "$cloud/device"},
	{on: "$node/+/module/status", replace: "$node", with: "$cloud/node"},
	{on: "$device/+/channel/+/+", replace: "$device", with: "$cloud/device"},
	{on: "$device/+/channel/+/+/event/+", replace: "$device", with: "$cloud/device"},
	{on: "$ninja/services/rpc/+/+", replace: "$ninja", with: "$cloud/ninja"},
	{on: "$ninja/services/+", replace: "$ninja", with: "$cloud/ninja"},

	// temporary alternate topic to distinguish remote device replies from local-destined ones
	{on: "$device/+/channel/+/+/reply", replace: "$device", with: "$cloud/remote_device"},
}

var cloudTopics = []replaceTopic{
	{on: "$cloud/location/calibration/progress", replace: "$cloud/location", with: "$location"},
	{on: "$cloud/device/+/+/location", replace: "$cloud/device", with: "$device"},
	{on: "$cloud/device/+/announce", replace: "$cloud/device", with: "$device"},
	{on: "$cloud/device/+/channel/+/+/announce", replace: "$cloud/device", with: "$device"},
	{on: "$cloud/device/+/channel/+/+/reply", replace: "$cloud/device", with: "$device"},
	{on: "$cloud/ninja/services/rpc/+/+/reply", replace: "$cloud/ninja", with: "$ninja"},

	// temporary alternate topic to distinguish remote-originating device messages from local ones
	{on: "$cloud/remote_device/+/channel/#", replace: "$cloud/remote_device", with: "$device"},
}

func createBridge(conf *Config) *Bridge {
	return &Bridge{conf: conf, localTopics: localTopics, cloudTopics: cloudTopics, log: loggo.GetLogger("bridge")}
}

func (b *Bridge) start(cloudUrl string, token string) (err error) {

	if b.Configured {
		b.log.Warningf("Already configured.")
		return AlreadyConfigured
	}

	defer b.bridgeLock.Unlock()

	b.bridgeLock.Lock()

	b.log.Infof("Connecting the bridge")

	b.Configured = true

	url, err := url.Parse(cloudUrl)

	if err != nil {
		return err
	}

	b.cloudUrl = url
	b.token = token

	b.reconnectCh = make(chan bool, 1)
	b.shutdownCh = make(chan bool, 1)

	if err = b.connect(); err != nil {
		b.log.Errorf("Connect failed %s", err)
		b.scheduleReconnect(err)
	}

	go b.mainBridgeLoop()

	return err
}

func (b *Bridge) stop() error {

	if !b.Configured {
		b.log.Warningf("Already unconfigured.")
		return AlreadyUnConfigured
	}

	defer b.bridgeLock.Unlock()

	b.bridgeLock.Lock()

	b.log.Infof("Disconnecting bridge")

	if b.Configured {
		// tell the worker to shutdown
		b.shutdownCh <- true

		b.Configured = false
	}

	b.resetTimer()

	b.disconnectAll()

	return nil
}

func (b *Bridge) connect() (err error) {

	if b.local, err = b.buildClient(b.conf.LocalUrl, ""); err != nil {
		b.Connected = false
		return err
	}

	if b.remote, err = b.buildClient(b.cloudUrl.String(), b.token); err != nil {
		b.Connected = false
		return err
	}

	if err = b.subscriptions(); err != nil {
		return err
	}

	// we are now connected
	b.Connected = true

	return nil
}

func (b *Bridge) reconnect() (err error) {

	if b.local, err = b.buildClient(b.conf.LocalUrl, ""); err != nil {
		b.Connected = false
		return err
	}

	if b.remote, err = b.buildClient(b.cloudUrl.String(), b.token); err != nil {
		b.Connected = false
		return err
	}

	if err = b.subscriptions(); err != nil {
		return err
	}

	// we are now connected
	b.Connected = true
	b.LastError = nil

	return nil
}

func (b *Bridge) subscriptions() (err error) {

	if err = b.subscribe(b.local, b.remote, b.localTopics, "local"); err != nil {
		return err
	}

	if err = b.subscribe(b.remote, b.local, b.cloudTopics, "cloud"); err != nil {
		return err
	}
	return nil

}

func (b *Bridge) disconnectAll() {
	b.log.Infof("disconnectAll")
	// we are now disconnected
	b.Connected = false
	if b.local != nil && b.local.IsConnected() {
		b.local.Disconnect(100)
	}
	if b.remote != nil && b.remote.IsConnected() {
		b.local.Disconnect(100)
	}
}

func (b *Bridge) mainBridgeLoop() {

	for {
		select {
		case <-b.reconnectCh:
			b.log.Infof("reconnecting")
			if err := b.reconnect(); err != nil {
				b.log.Errorf("Reconnect failed %s", err)
				b.scheduleReconnect(err)
			}
		case <-b.shutdownCh:
			b.log.Infof("shutting down bridge")
			return
		}

	}

}

func (b *Bridge) buildClient(server string, token string) (*mqtt.MqttClient, error) {

	b.log.Infof("building client for %s", server)

	opts := mqtt.NewClientOptions().AddBroker(server).SetTlsConfig(&tls.Config{InsecureSkipVerify: true})

	if token != "" {
		opts.SetUsername(token)
	}

	opts.SetClientId(fmt.Sprintf("%d", time.Now().Unix()))

	opts.SetKeepAlive(15) // set a 15 second ping time for ELB

	// pretty much log the reason and quit
	opts.SetOnConnectionLost(b.onConnectionLoss)

	client := mqtt.NewClient(opts)
	_, err := client.Start()
	return client, err
}

func (b *Bridge) subscribe(src *mqtt.MqttClient, dst *mqtt.MqttClient, topics []replaceTopic, tag string) (err error) {
	for _, topic := range topics {

		topicFilter, _ := mqtt.NewTopicFilter(topic.on, 0)
		b.log.Infof("(%s) subscribed to %s", tag, topic.on)

		if receipt, err := src.StartSubscription(b.buildHandler(topic, tag, dst), topicFilter); err != nil {
			return err
		} else {
			<-receipt
			b.log.Infof("(%s) subscribed to %+v", tag, topicFilter)
		}
	}

	return nil
}

func (b *Bridge) unsubscribe(client *mqtt.MqttClient, topics []replaceTopic, tag string) {
	topicNames := []string{}
	for _, topic := range topics {
		topicNames = append(topicNames, topic.on)
	}
	b.log.Infof("(%s) unsubscribed to %s", tag, topicNames)
	client.EndSubscription(topicNames...)
}

func (b *Bridge) buildHandler(topic replaceTopic, tag string, dst *mqtt.MqttClient) mqtt.MessageHandler {
	return func(src *mqtt.MqttClient, msg mqtt.Message) {
		if b.conf.IsDebug() {
			b.log.Infof("(%s) topic: %s updated: %s len: %d", tag, msg.Topic(), topic.updated(msg.Topic()), len(msg.Payload()))
		}
		b.Counter++
		payload := b.updateSource(msg.Payload(), tag)
		dst.PublishMessage(topic.updated(msg.Topic()), mqtt.NewMessage(payload))
	}
}

func (b *Bridge) scheduleReconnect(reason error) {
	b.LastError = reason
	b.disconnectAll()
	b.resetTimer()

	switch reason {
	case mqtt.ErrBadCredentials:
		b.log.Warningf("Reconnect failed trying again in 30s")

		b.timer = time.AfterFunc(30*time.Second, func() {
			b.reconnectCh <- true
		})

	default:
		b.log.Warningf("Reconnect failed trying again in 5s")
		// TODO add exponential backoff
		b.timer = time.AfterFunc(5*time.Second, func() {
			b.reconnectCh <- true
		})
	}

}

func (b *Bridge) resetTimer() {
	if b.timer != nil {
		b.timer.Stop()
	}
}

func (b *Bridge) onConnectionLoss(client *mqtt.MqttClient, reason error) {
	b.log.Errorf("Connection failed %s", reason)

	// we are now disconnected
	b.Connected = false

	b.scheduleReconnect(reason)

}

func (b *Bridge) IsConnected() bool {
	if b.remote == nil || b.local == nil {
		return false
	}
	return (b.remote.IsConnected() && b.local.IsConnected())
}

func (b *Bridge) updateSource(payload []byte, tag string) []byte {
	var msg map[string]interface{}

	err := json.Unmarshal(payload, &msg)

	if err != nil {
		return payload
	}

	if msg["$mesh_source"] == nil {
		switch tag {
		case "local":
			msg["$mesh_source"] = b.conf.SerialNo
		case "cloud":
			msg["$mesh_source"] = "cloud-" + strings.Replace(b.cloudUrl.Host, ".", "_", -1) // encoded to look less wierd
		}
	}

	v, err := json.Marshal(&msg)

	if err != nil {
		return payload
	}

	b.log.Infof("msg %s", string(v))

	return v
}
