package bootx

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gen-iot/std"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Qos = byte

//noinspection ALL
const (
	/**
	消息是基于TCP/IP网络传输的.
	没有回应,在协议中也没有定义重传的语义.消息可能到达服务器1次,也可能根本不会到达.
	*/
	Lv0OnceMax Qos = iota
	/**
	服务器接收到消息会被确认,通过传输一个PUB ACK信息.
	如果有一个可以辨认的传输失败,无论是通讯连接还是发送设备,还是过了一段时间确认信息没有收到,发送方都会将消息头的DUP位置1,然后再次发送消息.
	消息最少一次到达服务器.SUBSCRIBE和UNSUBSCRIBE都使用level 1 的QoS.
	如果客户端没有接收到PUB ACK信息（无论是应用定义的超时,还是检测到失败然后通讯session重启）,客户端都会再次发送PUBLISH信息,并且将DUP位置1.
	当它从客户端接收到重复的数据,服务器重新发送消息给订阅者,并且发送另一个PUBACK消息.
	*/
	Lv1AtLeastOnce
	/**
	在QoS level 1上附加的协议流保证了重复的消息不会传送到接收的应用.
	这是最高级别的传输,当重复的消息不被允许的情况下使用.
	这样增加了网络流量,但是它通常是可以接受的,因为消息内容很重要.
	QoS level 2在消息头有Message ID.
	*/
	Lv2OnlyOnce
)

const kDefaultConnTimeout = 30

type MqttConfig struct {
	MqttPubApiAddr string `json:"mqttPubApiAddr" validate:"required"`
	Qos            Qos    `json:"qos"`
	RetainMsg      bool   `json:"retainMsg"`
	ConnTimeout    int    `json:"connTimeout"`
	UserName       string `json:"username"`
	Password       string `json:"password"`
	Debug          bool   `json:"debug"`
	ClientIdPrefix string `json:"clientIdPrefix"`
}

var MqttDefaultConfig = &MqttConfig{
	MqttPubApiAddr: "http://127.0.0.1:8080/api/v2/mqtt/publish",
	Qos:            Lv2OnlyOnce,
	RetainMsg:      false,
	ConnTimeout:    kDefaultConnTimeout,
	UserName:       "admin",
	Password:       "pujie123",
	Debug:          false,
	ClientIdPrefix: "",
}

const ApplicationJson = "application/json"

func newHttpClient() *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DisableKeepAlives: true,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   time.Second * time.Duration(mqttConfig.ConnTimeout),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			req.SetBasicAuth(mqttConfig.UserName, mqttConfig.Password)
			return nil
		},
	}
}

type MqttPubCli struct {
	httpCli *http.Client
}

var cli *MqttPubCli = nil
var once = sync.Once{}
var mqttConfig *MqttConfig = nil

func GetCli() *MqttPubCli {
	once.Do(func() {
		err := std.ValidateStruct(mqttConfig)
		std.AssertError(err, "Mqtt配置不正确")
		logger.Println("mqtt init ...")
		cli = &MqttPubCli{
			httpCli: newHttpClient(),
		}
	})
	return cli
}

func sendJsonPost(url string, body string) (string, error) {
	if mqttConfig.Debug {
		logger.Printf("mqtt pub by http post %s :\n %s", url, body)
	}
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(mqttConfig.UserName, mqttConfig.Password)
	req.Header.Set("Content-Type", ApplicationJson)
	resp, err := GetCli().httpCli.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	b, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return "", readErr
	}
	return string(b), nil
}

type mqttPubReq struct {
	Topic    string `json:"topic"`
	Payload  string `json:"payload"`
	Qos      Qos    `json:"qos"`
	Retain   bool   `json:"retain"`
	ClientId string `json:"client_id"`
}

type mqttPubRsp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (cli *MqttPubCli) Publish(topic string, msg interface{}) error {
	msgBs, err := json.Marshal(msg)
	if err != nil {
		return errors.New(fmt.Sprintf("marshal msg to json failed : %s", err))
	}
	mqttPubReq := mqttPubReq{
		Topic:    topic,
		Payload:  string(msgBs),
		Qos:      mqttConfig.Qos,
		Retain:   mqttConfig.RetainMsg,
		ClientId: fmt.Sprintf("%s%s", mqttConfig.ClientIdPrefix, std.GenRandomUUID()),
	}
	bs, err := json.Marshal(mqttPubReq)
	if err != nil {
		return errors.New(fmt.Sprintf("marshal mqtt publish req failed : %s", err))
	}
	ack, err := sendJsonPost(mqttConfig.MqttPubApiAddr, string(bs))
	if err != nil {
		return err
	}
	rsp := new(mqttPubRsp)
	if err = json.Unmarshal([]byte(ack), rsp); err != nil {
		return errors.New(fmt.Sprintf("unmarshal mqtt response failed : %s", err))
	}
	if rsp.Code != 0 {
		return errors.New(fmt.Sprintf("mqtt publish to '%s' failed : %s", topic, rsp.Message))
	}
	return nil
}

func mqttPubCliInit(pubApiAddr string) {
	conf := MqttDefaultConfig
	conf.MqttPubApiAddr = pubApiAddr
	mqttPubCliInitWithConfig(conf)
}

func mqttPubCliInitWithConfig(conf *MqttConfig) {
	MqttDefaultConfig = conf
	GetCli()
}

func mqttPubCliCleanup() {
	logger.Println("mqtt cleanup ...")
}
