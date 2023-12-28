package mqtt

import (
	"encoding/json"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MqttClient struct {
	broker string
	port   int
	client mqtt.Client
}

/*
Usage

	m := mqttlib.NewMqtt("mqtt.mridulganga.dev", 1883)
	go m.Connect()
	m.WaitUntilConnected()

	m.Sub("topic", func(client mqtt.Client, message mqtt.Message) {
		fmt.Println("Received " + string(message.Payload()))
	})

	m.Publish("topic", "Hello World")

	mqtt guide https://www.emqx.com/en/blog/how-to-use-mqtt-in-golang
*/
func NewMqtt(broker string, port int) MqttClient {
	mqttClient := MqttClient{
		broker: broker,
		port:   port,
		client: nil,
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
	opts.OnConnect = mqttClient.ConnectHandler
	opts.OnConnectionLost = mqttClient.ConnectLostHandler
	client := mqtt.NewClient(opts)
	mqttClient.client = client
	return mqttClient
}

func (m MqttClient) Connect() error {
	if token := m.client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Println("error while connect " + token.Error().Error())
		return token.Error()
	}
	select {}
}
func (m MqttClient) IsConnected() bool {
	return m.client.IsConnected()
}

func (m MqttClient) WaitUntilConnected() {
	for !m.client.IsConnected() {
		time.Sleep(time.Second)
	}
}

func (m MqttClient) ConnectHandler(client mqtt.Client) {
	fmt.Println("Connected")
}

func (m MqttClient) ConnectLostHandler(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v", err)
}

func (m MqttClient) Sub(topic string, messageHanler mqtt.MessageHandler) error {
	token := m.client.Subscribe(topic, 1, messageHanler)
	if token.Wait() && token.Error() != nil {
		fmt.Println("error while sub " + token.Error().Error())
		return token.Error()
	}
	fmt.Printf("Subscribed to topic: %s\n", topic)
	return nil
}

func (m MqttClient) Publish(topic string, data map[string]any) error {
	jsonData, _ := json.Marshal(data)
	token := m.client.Publish(topic, 0, false, jsonData)
	if token.Wait() && token.Error() != nil {
		fmt.Println("error while pub " + token.Error().Error())
		return token.Error()
	}
	// fmt.Printf("Published to topic: %s\n", topic)
	return nil
}
