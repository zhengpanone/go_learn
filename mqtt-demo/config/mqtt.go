package config

import (
	"crypto/tls"
	"errors"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"time"
)

// MqttConnectConfig 连接的相关配置
type MqttConnectConfig struct {
	Broker           string
	Port             int32
	User             string
	Password         string
	Certificate      string // 证书文件
	PrivateKey       string // 密钥
	ClientId         string
	WillEnabled      bool   // 遗愿
	WillTopic        string // 遗愿主题
	WillPayload      string // 遗愿消息
	WillQos          byte   //遗愿服务质量
	Qos              byte   // 服务质量
	Retained         bool   // 保留消息
	OnConnect        mqtt.OnConnectHandler
	OnConnectionLost mqtt.ConnectionLostHandler
}

type MqttClient struct {
	qos      byte
	retained bool
	Client   mqtt.Client
	topics   map[string]mqtt.MessageHandler
}

// 新建证书,也可以不用
func newTLSConfig(certFile string, privateKey string) (*tls.Config, error) {
	cret, err := tls.LoadX509KeyPair(certFile, privateKey)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		ClientAuth:         tls.NoClientCert, //不需要证书
		ClientCAs:          nil,              //不验证证书
		InsecureSkipVerify: true,             //接受服务器提供的任何证书和该证书中的任何主机名
		Certificates:       []tls.Certificate{cret},
	}, nil
}

func NewMqttClient(config MqttConnectConfig) *MqttClient {
	var c MqttClient
	opts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%s:%d", config.Broker, config.Port)).SetClientID(config.ClientId).SetMaxReconnectInterval(time.Second * 5)
	if config.WillEnabled {
		opts.SetWill(config.WillTopic, config.WillPayload, config.WillQos, config.Retained)
	}
	// 判断是否设置证书
	if config.Certificate != "" {
		tlsConfig, err := newTLSConfig(config.Certificate, config.PrivateKey)
		if err != nil {
			log.Panic(err)
			return nil
		}
		opts.SetTLSConfig(tlsConfig)
	} else {
		opts.SetUsername(config.User).SetPassword(config.Password)
	}
	// 初始化
	if config.OnConnect == nil {
		config.OnConnect = func(c mqtt.Client) {}
	}
	if config.OnConnectionLost == nil {
		config.OnConnectionLost = func(client mqtt.Client, err error) {

		}
	}
	opts.SetOnConnectHandler(c.connectHandler(config.OnConnect)).SetConnectionLostHandler(config.OnConnectionLost)
	c.Client = mqtt.NewClient(opts)
	c.qos = config.Qos                              // qos的级别
	c.retained = config.Retained                    // 保留消息
	c.topics = make(map[string]mqtt.MessageHandler) //topic
	if tc := c.Client.Connect(); tc.Wait() && tc.Error() != nil {
		log.Panic(tc.Error())
		return nil
	}
	return &c
}

func (mc *MqttClient) connectHandler(handler mqtt.OnConnectHandler) mqtt.OnConnectHandler {
	return func(c mqtt.Client) {

		for topic, onMessage := range mc.topics {
			fmt.Printf("Connected:%s", topic)
			mc.Client.Subscribe(topic, mc.qos, onMessage)
		}
		handler(c)
	}
}

func (mc *MqttClient) onConnectionLostHandler(handler mqtt.ConnectionLostHandler) mqtt.ConnectionLostHandler {
	return func(c mqtt.Client, e error) {
		fmt.Printf("Connected lost:%v", e)
		handler(c, e)
	}
}

// Publish  Mqtt message.
func (mc *MqttClient) Publish(topic string, payload []byte) error {
	if mc != nil && mc.Client.IsConnected() {
		if tc := mc.Client.Publish(topic, mc.qos, mc.retained, payload); tc.Wait() && tc.Error() != nil {
			return tc.Error()
		}
		return nil
	}
	return errors.New("mqttClient is nil or disconnected")
}

// Subscribes subscribe a Mqtt topic
func (mc *MqttClient) Subscribes(topics []string, onMessage mqtt.MessageHandler) error {
	for _, topic := range topics {
		if tc := mc.Client.Subscribe(topic, mc.qos, onMessage); tc.Wait() && tc.Error() != nil {
			return tc.Error()
		}
		mc.topics[topic] = onMessage
		log.Println(fmt.Sprintf("订阅主题[%s]成功", topic))
	}
	return nil
}

// Subscribe subscribe a Mqtt topic
func (mc *MqttClient) Subscribe(topic string, onMessage mqtt.MessageHandler) error {

	if tc := mc.Client.Subscribe(topic, mc.qos, onMessage); tc.Wait() && tc.Error() != nil {
		return tc.Error()
	}
	mc.topics[topic] = onMessage
	log.Println(fmt.Sprintf("订阅主题[%s]成功", topic))
	return nil
}

// Unsubscribe unsubscribe a mqtt topic
func (mc *MqttClient) Unsubscribe(topics ...string) error {
	if tc := mc.Client.Unsubscribe(topics...); tc.Wait() && tc.Error() != nil {
		return tc.Error()
	}
	for _, topic := range topics {
		delete(mc.topics, topic)
	}
	return nil
}

func (mc *MqttClient) Close() {
	mc.Client.Disconnect(250) // millisecond
}
