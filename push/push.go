package push

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/alecthomas/log4go"
	"github.com/streadway/amqp"
	"io/ioutil"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sx/config"
	"sx/encrypt"
	"time"
)

var (
	errPhoneNoneExist       = errors.New("get phone failed")
	errNotSupportRoutingKey = errors.New("not support this routingkey")
	errNoneVCCID            = errors.New("none exist vcc_id")
	errWrongVCCID           = errors.New("vcc_id wrong")
)

type SxMessage struct {
	Mobile     string `json:"mobile"`
	Operid     string `json:"operid"`
	Caller     string `json:"caller"`
	Sequenceid string `json:"sequenceid"`
	Tempid     string `json:"tempid"`
	Enterid    string `json:"enterid"`
	Enterpass  string `json:"enterpass"`
	Args       string `json:"args"`
	MsgType    string `json:"msgType"`
}

type SxResponse struct {
	ResultCode string `json:"resultCode"`
	ResultDesc string `json:"resultDesc"`
}

type Message struct {
	MainType int                    `json:"MainType"`
	ExtType  int                    `json:"ExtType"`
	Mode     int                    `json:"Mode"`
	ModeParm string                 `json:"ModeParm"`
	MSGID    string                 `json:"MSGID"`
	TELID    string                 `json:"TELID"`
	MSG      map[string]interface{} `json:"MSG"`
}

type Push struct {
	url string
	key string
	SxMessage
	*http.Client
	*config.Config
}

func NewPusher(conf *config.Config) (*Push, error) {
	if conf == nil {
		panic("conf nil")
	}
	p := &Push{
		url: conf.URL,
		key: conf.Key,
		SxMessage: SxMessage{
			Operid:    conf.Operid,
			Caller:    conf.Caller,
			Tempid:    conf.Tempid,
			Enterid:   conf.Enterid,
			Enterpass: conf.Enterpass,
			Args:      conf.Args,
		},
		Client: &http.Client{Timeout: time.Second * 3},
		Config: conf,
	}
	return p, nil
}

//ReadMsg handler for rmq
func (p *Push) ReadMsg(msg *amqp.Delivery) error {
	log.Debug("rx routingKey: %s, %s", msg.RoutingKey, string(msg.Body))
	phone, err := p.parseMessage(msg)
	if err == nil {
		valid, target := p.valid(phone)
		if !valid {
			log.Warn("invalid phone: %s", phone)
			return nil
		}
		//skip phones of Telecom
		if !p.support(target) {
			log.Warn("China Telecom not supported")
			return nil
		}
		err = p.publish(&SxMessage{Mobile: target})
		if err == nil {
			log.Debug("send %s ok", target)
		}
		return nil
	}
	log.Warn(err)
	return nil
}

//China Telecom
//133,1349,153,189,180,181,177,173,149,1700,1701,1702,199,1410
//China Unicom
//130,131,132,156,155,186,185,145,176,175,1707,1708,1709,166,146
//China Mobile Communication
//139,138,137,136,135,1340,1341,1342,1343,1344,1345,1346,1347,1348,159,158,157,150,151,152,147,188,187,182,183,184,178,1703,1705,1706,198,148,1440
//todo...enchance efficiency of algorithm
func (p *Push) support(phone string) bool {
	l := "133,1349,153,189,180,181,177,173,149,1700,1701,1702,199,1410"
	s := strings.Split(l, ",")
	if len(phone) == 12 {
		phone = phone[1:]
	}
	for _, v := range s {
		if strings.HasPrefix(phone, v) {
			return false
		}
	}
	return true
}

func (p *Push) checkVccid(m *Message) (bool, error) {
	vccid, ok := m.MSG["vcc_id"]
	if !ok {
		return false, errNoneVCCID
	}
	strVccid := vccid.(string)
	if len(strVccid) == 0 {
		return false, errWrongVCCID
	}
	id, _ := strconv.Atoi(strVccid)
	_, err := p.GetSmsConf(id)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (p *Push) parseMessage(msg *amqp.Delivery) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()

	var (
		m      Message
		target string
	)
	if msg.RoutingKey == "msgproxy.1.21" {
		err := json.Unmarshal(msg.Body, &m)
		if err != nil {
			return "", err
		}
		if exist, err := p.checkVccid(&m); !exist {
			return "", err
		}

		status, ok := m.MSG["status"]
		if ok {
			s := status.(string)
			if s == "1" {
				t, ok := m.MSG["called"]
				if ok {
					target = t.(string)
				}
			} else if s == "5" {
				t, ok := m.MSG["trans_called"]
				if ok {
					target = t.(string)
				}
			}
		}
	} else if msg.RoutingKey == "msgproxy.2.10" {
		err := json.Unmarshal(msg.Body, &m)
		if err != nil {
			return "", err
		}
		if exist, err := p.checkVccid(&m); !exist {
			return "", err
		}
		status, ok := m.MSG["call_sta"]
		if ok {
			s := status.(string)
			if s == "2" {
				t, ok := m.MSG["user_num"]
				if ok {
					target = t.(string)
				}
			}
		}
	} else {
		return "", errNotSupportRoutingKey
	}
	if len(target) == 0 {
		return "", errPhoneNoneExist
	}
	return target, nil
}

//1开头11位
//01开头12位并且不是010开头
func (p *Push) valid(phone string) (bool, string) {
	target := phone
	pattern := "^[0-9]*$" //反斜杠要转义
	if result, err := regexp.MatchString(pattern, phone); err == nil {
		if !result {
			return false, ""
		}
	}

	l := len(phone)
	if l == 12 {
		if phone[0] == '0' {
			target = phone[1:]
		} else {
			return false, ""
		}
	}

	begin := string(target[0])
	begin2 := string(target[1])
	if begin == "1" && begin2 != "0" {
		return true, target
	}

	return false, ""
}

//Deprecated
func (p *Push) parseMessage2(msg *amqp.Delivery)(string, error){
	defer func() {
		if err := recover(); err != nil{
			log.Error(err)
		}
	}()

	var (
		m	Message
		target string
	)
	switch msg.RoutingKey {
	case "msgproxy.1.21":
		err := json.Unmarshal(msg.Body, &m)
		if err != nil{
			return "", err
		}
		value := reflect.ValueOf(m.MSG)
		if status := value.FieldByName("status");status != (reflect.Value{}){
			s := status.String()
			if s == "1"{
				target = value.FieldByName("called").String()
			}else if s == "5"{
				target = value.FieldByName("trans_called").String()
			}
		}
	case "msgproxy.2.10":
		err := json.Unmarshal(msg.Body, &m)
		if err != nil{
			return "", err
		}
		value := reflect.ValueOf(m.MSG).Elem()
		if status := value.FieldByName("call_sta");status != (reflect.Value{}){
			s := status.String()
			if s == "2"{
				target = value.FieldByName("user_num").String()
			}
		}
	default:
		log.Error("not support this routingkey")
	}
	if len(target) == 0{
		return "", fmt.Errorf("get phone failed")
	}
	return target, nil



	//if msg.RoutingKey == "msgproxy.1.21"{
	//	err := json.Unmarshal(msg.Body, &m)
	//	if err != nil{
	//		return "", err
	//	}
	//	value := reflect.ValueOf(m.MSG)
	//	if status := value.FieldByName("status");status != (reflect.Value{}){
	//		s := status.String()
	//		if s == "1"{
	//			target = value.FieldByName("called").String()
	//		}else if s == "5"{
	//			target = value.FieldByName("trans_called").String()
	//		}
	//	}
	//}else if msg.RoutingKey == "msgproxy.2.10"{
	//	err := json.Unmarshal(msg.Body, &m)
	//	if err != nil{
	//		return "", err
	//	}
	//	value := reflect.ValueOf(m.MSG).Elem()
	//	if status := value.FieldByName("call_sta");status != (reflect.Value{}){
	//		s := status.String()
	//		if s == "2"{
	//			target = value.FieldByName("user_num").String()
	//		}
	//	}
	//}else{
	//	log.Error("not support this routingkey")
	//}
	//if len(target) == 0{
	//	return "", fmt.Errorf("get phone failed")
	//}
	//return target, nil
}

func (p *Push) publish(m *SxMessage) error {
	if len(m.Mobile) == 0 {
		return fmt.Errorf("mobile number empty, %+v", m)
	}
	m.Caller = p.SxMessage.Caller
	m.Operid = p.SxMessage.Operid
	m.Sequenceid = time.Now().Format("20060102150405.999")
	m.Sequenceid = m.Sequenceid + "_" + p.SxMessage.Operid
	if len(p.SxMessage.Args) > 0 {
		m.Args = p.SxMessage.Args
	}
	m.Tempid = p.SxMessage.Tempid
	m.Enterid = p.SxMessage.Enterid
	m.Enterpass = p.SxMessage.Enterpass
	if len(m.MsgType) == 0 {
		m.MsgType = "4"
	}
	fmt.Printf("加密前：%+v\n", m)
	value := reflect.ValueOf(m).Elem()
	l := value.NumField()
	for i := 0; i < l; i++ {
		v := value.Field(i).String()
		if len(v) > 0 {
			enc, err := encrypt.AESBase64Encrypt(v, p.key)
			if err != nil {
				return fmt.Errorf("%s, %+v", err.Error(), m)
			}
			value.Field(i).SetString(enc)
		}
	}
	fmt.Printf("加密后：%+v\n", m)
	buf, _ := json.Marshal(m)
	return p.post(buf)
}

func (p *Push) post(buf []byte) error {
	resp, err := p.Post(p.url, "Content-Type:application/json", bytes.NewReader(buf))
	if err != nil {
		log.Error(err)
		return err
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)

	var rep SxResponse
	err = json.Unmarshal(data, &rep)
	if err != nil {
		log.Error("%s, %s", err.Error(), string(data))
		return err
	}
	if rep.ResultCode != "200" {
		log.Error(string(data))
		return fmt.Errorf("%+v", rep)
	}
	return nil
}
