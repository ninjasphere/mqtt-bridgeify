package agent

import (
	"crypto/tls"
	"log"
	"strings"
	"sync"
	"time"

	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
)

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

	localTopics []replaceTopic
	cloudTopics []replaceTopic

	cloudUrl string
	token    string

	timer       *time.Timer
	reconnectCh chan bool
	shutdownCh  chan bool

	Configured bool
	Connected  bool
	Counter    int64

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
	{on: "$device/+/+/rssi", replace: "$device", with: "$cloud/device"},
}

var cloudTopics = []replaceTopic{
	{on: "$cloud/location/calibration/complete", replace: "$cloud/location", with: "$location"},
	{on: "$cloud/device/+/+/location", replace: "$cloud/device", with: "$device"},
}

func createBridge(conf *Config) *Bridge {
	return &Bridge{conf: conf, localTopics: localTopics, cloudTopics: cloudTopics, reconnectCh: make(chan bool, 1), shutdownCh: make(chan bool, 1)}
}

func (b *Bridge) start(cloudUrl string, token string) error {

	defer b.bridgeLock.Unlock()

	b.bridgeLock.Lock()

	log.Printf("[INFO] Connecting the bridge")

	b.Configured = true

	b.cloudUrl = cloudUrl
	b.token = token

	if err := b.connect(); err != nil {
		log.Printf("[WARN] reconnect failed trying again in 5s")
		b.disconnectAll()
		b.timer = time.AfterFunc(5*time.Second, func() {
			b.reconnectCh <- true
		})
	}

	go b.mainBridgeLoop()

	return nil
}

func (b *Bridge) stop() error {

	defer b.bridgeLock.Unlock()

	b.bridgeLock.Lock()

	log.Printf("[INFO] Disconnecting bridge")

	if b.Configured {
		// tell the worker to shutdown
		b.shutdownCh <- true

		b.Configured = false
	}

	if b.timer != nil {
		log.Printf("[INFO] Stopping timer")
		b.timer.Stop()
	}

	b.disconnectAll()

	return nil
}

func (b *Bridge) connect() (err error) {

	if b.local, err = b.buildClient(b.conf.LocalUrl, ""); err != nil {
		return err
	}

	if b.remote, err = b.buildClient(b.cloudUrl, b.token); err != nil {
		return err
	}

	if err = b.subscribe(b.local, b.remote, b.localTopics, "local"); err != nil {
		return err
	}

	if err = b.subscribe(b.remote, b.local, b.cloudTopics, "cloud"); err != nil {
		return err
	}

	// we are now connected
	b.Connected = true

	return nil
}

func (b *Bridge) disconnectAll() {
	log.Printf("[INFO] disconnectAll")
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

	// setup a timer

	// setup a shutdown channel

	for {
		select {
		case <-b.reconnectCh:
			log.Printf("[INFO] reconnecting")
			if err := b.connect(); err != nil {
				b.disconnectAll()
				log.Printf("[WARN] reconnect failed trying again in 5s")
				b.timer = time.AfterFunc(5*time.Second, func() {
					b.reconnectCh <- true
				})
			}
		case <-b.shutdownCh:
			log.Printf("[INFO] shutting down bridge")
			return
		}

	}

}

func (b *Bridge) buildClient(server string, token string) (*mqtt.MqttClient, error) {

	opts := mqtt.NewClientOptions().SetClientId("123").SetBroker(server).SetTlsConfig(&tls.Config{InsecureSkipVerify: true})

	if token != "" {
		opts.SetUsername(token)
	}

	// shutup
	opts.SetTraceLevel(mqtt.Off)

	// pretty much log the reason and quit
	opts.SetOnConnectionLost(b.onConnectionLoss)

	client := mqtt.NewClient(opts)
	_, err := client.Start()
	return client, err
}

func (b *Bridge) subscribe(src *mqtt.MqttClient, dst *mqtt.MqttClient, topics []replaceTopic, tag string) (err error) {
	for _, topic := range topics {

		topicFilter, _ := mqtt.NewTopicFilter(topic.on, 0)
		log.Printf("[INFO] (%s) subscribed to %s", tag, topic.on)

		if _, err := src.StartSubscription(b.buildHandler(topic, tag, dst), topicFilter); err != nil {
			return err
		}
	}

	return nil
}

func (b *Bridge) unsubscribe(client *mqtt.MqttClient, topics []replaceTopic, tag string) {
	topicNames := []string{}
	for _, topic := range topics {
		topicNames = append(topicNames, topic.on)
	}
	log.Printf("[INFO] (%s) unsubscribed to %s", tag, topicNames)
	client.EndSubscription(topicNames...)
}

func (b *Bridge) buildHandler(topic replaceTopic, tag string, dst *mqtt.MqttClient) mqtt.MessageHandler {
	return func(src *mqtt.MqttClient, msg mqtt.Message) {
		if b.conf.IsDebug() {
			log.Printf("[INFO] (%s) topic: %s updated: %s len: %d", tag, msg.Topic(), topic.updated(msg.Topic()), len(msg.Payload()))
		}
		b.Counter++
		dst.PublishMessage(topic.updated(msg.Topic()), mqtt.NewMessage(msg.Payload()))
	}
}

func (b *Bridge) onConnectionLoss(client *mqtt.MqttClient, reason error) {
	log.Printf("[WARN] Connection failed %s", reason)

	// we are now disconnected
	b.Connected = false

	b.disconnectAll()

	b.reconnectCh <- true

}
