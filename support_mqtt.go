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

const (
	MqttDefaultTimeoutSec = 20
	MqttDefaultRetainMsg  = true
	MqttDefaultQos        = Lv2OnlyOnce
	applicationJson       = "application/json"
)

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

func newHttpClient(timeoutSec int64) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DisableKeepAlives: true,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   time.Second * time.Duration(timeoutSec),
	}
}

type MqttPubCli struct {
	httpCli        *http.Client
	MqttPubApiAddr string
	Timeout        int64
	Debug          bool
	UserName       string
	Password       string
	ClientIdPrefix string
}

//noinspection ALL
func NewMqttPubCli(mqttPubApiUrl, username, pass string, timeoutSec int64, debug bool) *MqttPubCli {
	if timeoutSec <= 0 {
		timeoutSec = MqttDefaultTimeoutSec
	}
	cli := &MqttPubCli{
		httpCli:        newHttpClient(timeoutSec),
		MqttPubApiAddr: mqttPubApiUrl,
		Timeout:        timeoutSec,
		Debug:          debug,
	}
	cli.httpCli.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		req.SetBasicAuth(username, pass)
		return nil
	}
	return cli
}

func NewMqttPubCli1(mqttPubApiUrl, username, pass string) *MqttPubCli {
	return NewMqttPubCli(mqttPubApiUrl, username, pass, MqttDefaultTimeoutSec, false)
}

func (this *MqttPubCli) postJson(body string) (string, error) {
	if this.Debug {
		logger.Printf("mqtt pub by http post %s :\n %s", this.MqttPubApiAddr, body)
	}
	req, err := http.NewRequest("POST", this.MqttPubApiAddr, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(this.UserName, this.Password)
	req.Header.Set("Content-Type", applicationJson)
	resp, err := this.httpCli.Do(req)
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

func (this *MqttPubCli) Publish(topic string, msg interface{}, qos Qos, retainMsg bool) error {
	msgBs, err := json.Marshal(msg)
	if err != nil {
		return errors.New(fmt.Sprintf("marshal msg to json failed : %s", err))
	}
	mqttPubReq := mqttPubReq{
		Topic:    topic,
		Payload:  string(msgBs),
		Qos:      qos,
		Retain:   retainMsg,
		ClientId: fmt.Sprintf("%s%s", this.ClientIdPrefix, std.GenRandomUUID()),
	}
	bs, err := json.Marshal(mqttPubReq)
	if err != nil {
		return errors.New(fmt.Sprintf("marshal mqtt publish req failed : %s", err))
	}
	ack, err := this.postJson(string(bs))
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

func (this *MqttPubCli) Publish1(topic string, msg interface{}) error {
	return this.Publish(topic, msg, MqttDefaultQos, MqttDefaultRetainMsg)
}
