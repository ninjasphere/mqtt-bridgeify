package agent

import (
	"fmt"
	"log"

	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
)

/*
 Just manages all the data going into out of this service.
*/
type Bus struct {
	conf   *Config
	agent  *Agent
	client *mqtt.MqttClient
}

type connectRequest struct {
	Url   string
	Token string
	Time  int64
}

type disconnectRequest struct {
}

type statsEvent struct {
	msg int64
}

func createBus(conf *Config, agent *Agent) *Bus {
	return &Bus{conf: conf, agent: agent}
}

func (b *Bus) listen() {
	log.Printf("[INFO] connecting to the bus")

	opts := mqtt.NewClientOptions().SetBroker(b.conf.LocalUrl)

	b.client = mqtt.NewClient(opts)

	_, err := b.client.Start()
	if err != nil {
		log.Fatalf("error starting connection: %s", err)
	} else {
		fmt.Printf("Connected as %s\n", b.conf.LocalUrl)
	}

	topicFilter, _ := mqtt.NewTopicFilter("$sphere/bridge/connect", 0)
	if _, err := b.client.StartSubscription(b.handleConnect, topicFilter); err != nil {
		log.Fatalf("error starting subscription: %s", err)
	}

	topicFilter, _ = mqtt.NewTopicFilter("$sphere/bridge/disconnect", 0)
	if _, err := b.client.StartSubscription(b.handleConnect, topicFilter); err != nil {
		log.Fatalf("error starting subscription: %s", err)
	}

}

func (b *Bus) handleConnect(client *mqtt.MqttClient, msg mqtt.Message) {

}

func (b *Bus) handleDisconnect(client *mqtt.MqttClient, msg mqtt.Message) {

}
