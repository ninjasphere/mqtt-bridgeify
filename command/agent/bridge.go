package agent

import (
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/juju/loggo"
	mqtt "github.com/wolfeidau/org.eclipse.paho.mqtt.golang"
)

const (
	BridgeConnecting = iota
	BridgeConnected
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

	cloudUrl string
	token    string

	// timer       *time.Timer
	reconnectCh chan bool
	shutdownCh  chan bool
	bridgeState int

	Configured bool
	Connected  bool
	Counter    int64

	LastError error
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
}

var cloudTopics = []replaceTopic{
	{on: "$cloud/location/calibration/progress", replace: "$cloud/location", with: "$location"},
	{on: "$cloud/device/+/+/location", replace: "$cloud/device", with: "$device"},
	{on: "$cloud/device/+/announce", replace: "$cloud/device", with: "$device"},
	{on: "$cloud/device/+/channel/+/+/announce", replace: "$cloud/device", with: "$device"},
	{on: "$cloud/device/+/channel/+/+/reply", replace: "$cloud/device", with: "$device"},
	{on: "$cloud/ninja/services/rpc/+/+/reply", replace: "$cloud/ninja", with: "$ninja"},
}

func createBridge(conf *Config) *Bridge {
	return &Bridge{conf: conf, localTopics: localTopics, cloudTopics: cloudTopics, log: loggo.GetLogger("bridge")}
}

func (b *Bridge) start(cloudUrl string, token string) (err error) {

	if b.Configured {
		b.log.Warningf("Already configured.")
		return AlreadyConfigured
	}

	b.log.Infof("Connecting the bridge")

	b.Configured = true

	b.cloudUrl = cloudUrl
	b.token = token

	b.reconnectCh = make(chan bool, 1)
	b.shutdownCh = make(chan bool, 1)

	if err = b.connect(); err != nil {
		b.log.Errorf("Connect failed %s", err)
	}

	go b.mainBridgeLoop()

	return err
}

func (b *Bridge) stop() error {

	if !b.Configured {
		b.log.Warningf("Already unconfigured.")
		return AlreadyUnConfigured
	}

	b.log.Infof("Disconnecting bridge")

	b.Configured = false

	return nil
}

func (b *Bridge) connect() (err error) {

	if b.local, err = b.buildClient(b.conf.LocalUrl, ""); err != nil {
		b.Connected = false
		return err
	}

	if b.remote, err = b.buildClient(b.cloudUrl, b.token); err != nil {
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
			return
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

	opts.SetOnConnectHandler(b.onConnection)
	opts.SetConnectionLostHandler(b.onConnectionLoss)

	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return client, nil
}

func (b *Bridge) subscribe(src *mqtt.MqttClient, dst *mqtt.MqttClient, topics []replaceTopic, tag string) (err error) {
	for _, topic := range topics {
		if token := src.Subscribe(topic.on, 0, b.buildHandler(topic, tag, dst)); token.Wait() && token.Error() != nil {
			return token.Error()
		}
		b.log.Infof("(%s) subscribed to %s", tag, topic.on)
	}

	return nil
}

func (b *Bridge) unsubscribe(client *mqtt.MqttClient, topics []replaceTopic, tag string) {

	topicNames := []string{}

	for _, topic := range topics {
		topicNames = append(topicNames, topic.on)
	}

	if token := client.Unsubscribe(topicNames...); token.Wait() && token.Error() != nil {
		b.log.Errorf("Error occured when unsubscribing: %s", token.Error())
	}

	b.log.Infof("(%s) unsubscribed to %s", tag, topicNames)
}

func (b *Bridge) buildHandler(topic replaceTopic, tag string, dst *mqtt.MqttClient) mqtt.MessageHandler {
	return func(src *mqtt.MqttClient, msg mqtt.Message) {
		if b.conf.IsDebug() {
			b.log.Infof("(%s) topic: %s updated: %s len: %d", tag, msg.Topic(), topic.updated(msg.Topic()), len(msg.Payload()))
		}
		b.Counter++
		token := dst.Publish(topic.updated(msg.Topic()), 0, false, msg.Payload())
		token.Wait()
	}
}

func (b *Bridge) onConnection(client *mqtt.MqttClient) {
	b.log.Errorf("Connected to %v", client)

	// we are now connected
	b.Connected = true

}

func (b *Bridge) onConnectionLoss(client *mqtt.MqttClient, reason error) {
	b.log.Errorf("Connection failed %s", reason)

	// we are now disconnected
	b.Connected = false

	//b.scheduleReconnect(reason)

}

func (b *Bridge) IsConnected() bool {
	if b.remote == nil || b.local == nil {
		return false
	}
	return (b.remote.IsConnected() && b.local.IsConnected())
}
