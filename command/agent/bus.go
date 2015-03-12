package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/juju/loggo"

	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
)

const (
	connectTopic    = "$sphere/bridge/connect"
	disconnectTopic = "$sphere/bridge/disconnect"
	statusTopic     = "$sphere/bridge/status"
	responseTopic   = "$sphere/bridge/response"
)

/*
 Just manages all the data going into out of this service.
*/
type Bus struct {
	conf         *Config
	agent        *Agent
	client       *mqtt.MqttClient
	statusTicker *time.Ticker
	log          loggo.Logger
}

type connectRequest struct {
	Id    string `json:"id"`
	Url   string `json:"url"`
	Token string `json:"token"`
}

type disconnectRequest struct {
	Id string `json:"id"`
}

type statusEvent struct {
	Status string `json:"status"`
}

type resultStatus struct {
	Id         string `json:"id"`
	Connected  bool   `json:"connected"`
	Configured bool   `json:"configured"`
	LastError  string `json:"lastError"`
}

type statsEvent struct {

	// memory related information
	Alloc      uint64 `json:"alloc"`
	HeapAlloc  uint64 `json:"heapAlloc"`
	TotalAlloc uint64 `json:"totalAlloc"`

	LastError string `json:"lastError"`

	Connected  bool  `json:"connected"`
	Configured bool  `json:"configured"`
	Timestamp  int64 `json:"timestamp"`

	IngressCounter int64 `json:"ingressCounter"`
	EgressCounter  int64 `json:"egressCounter"`

	IngressBytes int64 `json:"ingressBytes"`
	EgressBytes  int64 `json:"egressBytes"`
}

func createBus(conf *Config, agent *Agent) *Bus {

	return &Bus{conf: conf, agent: agent, log: loggo.GetLogger("bus")}
}

func (b *Bus) listen() {
	b.log.Infof("connecting to the bus")

	opts := mqtt.NewClientOptions().AddBroker(b.conf.LocalUrl).SetClientId("mqtt-bridgeify-bus")

	b.client = mqtt.NewClient(opts)

	_, err := b.client.Start()
	if err != nil {
		b.log.Errorf("Can't start connection: %s", err)
	} else {
		b.log.Infof("Connected as %s\n", b.conf.LocalUrl)
	}

	topicFilter, _ := mqtt.NewTopicFilter(connectTopic, 0)
	if receipt, err := b.client.StartSubscription(b.handleConnect, topicFilter); err != nil {
		b.log.Errorf("Subscription Failed: %s", err)
		panic(err)
	} else {
		<-receipt
		b.log.Infof("Subscribed to: %+v", topicFilter)
	}

	topicFilter, _ = mqtt.NewTopicFilter(disconnectTopic, 0)
	if receipt, err := b.client.StartSubscription(b.handleDisconnect, topicFilter); err != nil {
		b.log.Errorf("Subscription Failed: %s", err)
		panic(err)
	} else {
		<-receipt
		b.log.Infof("Subscribed to: %+v", topicFilter)
	}

	ev := &statusEvent{Status: "started"}

	b.client.PublishMessage(statusTopic, b.encodeRequest(ev))

	b.setupBackgroundJob()

}

func (b *Bus) handleConnect(client *mqtt.MqttClient, msg mqtt.Message) {
	b.log.Infof("handleConnect")
	req := &connectRequest{}
	err := b.decodeRequest(&msg, req)
	if err != nil {
		b.log.Errorf("Unable to decode connect request %s", err)
	}

	if err := b.agent.startBridge(req); err != nil {
		// send out a bad result
		b.sendResult(req.Id, false, true, err)
	} else {
		// send out a good result
		b.sendResult(req.Id, true, true, err)
	}

}

func (b *Bus) handleDisconnect(client *mqtt.MqttClient, msg mqtt.Message) {
	b.log.Infof("handleDisconnect")
	req := &disconnectRequest{}
	err := b.decodeRequest(&msg, req)
	if err != nil {
		b.log.Errorf("Unable to decode disconnect request %s", err)
	}
	err = b.agent.stopBridge(req)
	// send out a result
	b.sendResult(req.Id, true, true, err)
}

func (b *Bus) sendResult(id string, connected bool, configured bool, result error) {

	var lastError string

	if result != nil {
		lastError = result.Error()
	}

	ev := &resultStatus{Id: id, Connected: connected, Configured: configured, LastError: lastError}
	b.client.PublishMessage(responseTopic, b.encodeRequest(ev))
}

func (b *Bus) setupBackgroundJob() {
	b.statusTicker = time.NewTicker(10 * time.Second)

	metricsTicker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-b.statusTicker.C:
			// emit the status
			status := b.agent.getStatus()
			b.log.Debugf("status %+v", status)
			b.client.PublishMessage(statusTopic, b.encodeRequest(status))
		case <-metricsTicker.C:
			metrics := b.agent.getMetrics()
			b.log.Debugf("metrics %+v", metrics)
			b.client.PublishMessage(fmt.Sprintf("$node/%s/module/status", b.conf.SerialNo), b.encodeRequest(metrics))

		}
	}

}

func (b *Bus) encodeRequest(data interface{}) *mqtt.Message {
	buf := bytes.NewBuffer(nil)
	json.NewEncoder(buf).Encode(data)
	return mqtt.NewMessage(buf.Bytes())
}

func (b *Bus) decodeRequest(msg *mqtt.Message, data interface{}) error {
	return json.NewDecoder(bytes.NewBuffer(msg.Payload())).Decode(data)
}
