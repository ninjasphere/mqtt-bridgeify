package agent

import (
	"crypto/tls"
	"fmt"
	"log"
	"strings"

	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
)

type Bridge struct {
	conf   *Config
	local  *mqtt.MqttClient
	remote *mqtt.MqttClient

	localTopics []replaceTopic
	cloudTopics []replaceTopic
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
	return &Bridge{conf: conf, localTopics: localTopics, cloudTopics: cloudTopics}
}

func (a *Bridge) start() error {

	a.local = a.buildClient(a.conf.LocalUrl, "")
	a.remote = a.buildClient(a.conf.CloudUrl, a.conf.Token)

	// this is a very basic pipe between the two
	// need to do some work on reconnects and receipt handling
	a.subscribe(a.local, a.remote, a.localTopics, "local")
	a.subscribe(a.remote, a.local, a.cloudTopics, "cloud")

	return nil
}

func (a *Bridge) stop() error {
	a.unsubscribe(a.local, a.localTopics, "local")
	a.unsubscribe(a.remote, a.cloudTopics, "local")
	return nil
}

func (a *Bridge) buildClient(server string, token string) *mqtt.MqttClient {

	opts := mqtt.NewClientOptions().SetBroker(server).SetTraceLevel(mqtt.Off).SetTlsConfig(&tls.Config{InsecureSkipVerify: true})

	if token != "" {
		opts.SetUsername(token)
	}

	// pretty much log the reason and quit
	opts.SetOnConnectionLost(a.onConnectionLoss)

	client := mqtt.NewClient(opts)
	_, err := client.Start()
	if err != nil {
		log.Fatalf("error starting connection: %s", err)
	} else {
		fmt.Printf("Connected as %s\n", server)
	}
	return client
}

func (a *Bridge) subscribe(src *mqtt.MqttClient, dst *mqtt.MqttClient, topics []replaceTopic, tag string) {
	for _, topic := range topics {

		topicFilter, _ := mqtt.NewTopicFilter(topic.on, 0)
		log.Printf("[%s] subscribed to %s", tag, topic.on)

		if _, err := src.StartSubscription(a.buildHandler(topic, tag, dst), topicFilter); err != nil {
			log.Fatalf("error starting subscription: %s", err)
		}
	}
}

func (a *Bridge) unsubscribe(client *mqtt.MqttClient, topics []replaceTopic, tag string) {
	topicNames := []string{}
	for _, topic := range topics {
		topicNames = append(topicNames, topic.on)
	}
	log.Printf("[%s] unsubscribed to %s", tag, topicNames)
	client.EndSubscription(topicNames...)
}

func (a *Bridge) buildHandler(topic replaceTopic, tag string, dst *mqtt.MqttClient) mqtt.MessageHandler {
	return func(src *mqtt.MqttClient, msg mqtt.Message) {
		if a.conf.IsDebug() {
			log.Printf("[%s] topic: %s updated: %s len: %d", tag, msg.Topic(), topic.updated(msg.Topic()), len(msg.Payload()))
		}
		dst.PublishMessage(topic.updated(msg.Topic()), mqtt.NewMessage(msg.Payload()))
	}
}

func (a *Bridge) onConnectionLoss(client *mqtt.MqttClient, reason error) {
	log.Fatalf("Connection failed %s", reason)
}
