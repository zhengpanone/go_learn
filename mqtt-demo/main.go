package main

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"mqtt-demo/config"
	"time"
)

func main() {
	mqttConnectConfig := config.MqttConnectConfig{
		Broker:   "broker.emqx.io",
		Port:     1883,
		ClientId: "go_mqtt_client",
		User:     "emqx",
		Password: "public",
	}

	mc := config.NewMqttClient(mqttConnectConfig)

	mc.Subscribe("a/b/c", func(client mqtt.Client, message mqtt.Message) {
		payload := message.Payload()
		fmt.Printf("subscribe payload:%s", payload)
	})

	for i := 65; i < 75; i++ {
		err := mc.Publish("a/b/c", []byte{byte(i)})
		if err != nil {
			log.Panic(err)
			return
		}
		log.Println([]byte{byte(i)})
		time.Sleep(5 * time.Second)
	}
	mc.Client.Disconnect(2500)
}
