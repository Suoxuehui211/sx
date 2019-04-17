package push

import (
	"encoding/json"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"reflect"
	"sx/config"
	"testing"
)

func TestPush_Publish(t *testing.T) {
	conf := config.NewConfig()
	err := conf.Read("../conf.yml")
	assert.NoError(t, err)
	p, err := NewPusher(conf)
	assert.NoError(t, err)

	s := &SxMessage{
		//Mobile:"11111111111",
		Mobile: "13651694599",
	}
	err = p.publish(s)
	assert.NoError(t, err)
}

func TestReflect(t *testing.T) {
	m := &SxMessage{
		Operid:    "7777",
		Caller:    "123456",
		Tempid:    "1222",
		Enterid:   "dsewew23",
		Enterpass: "sfwe3efsfds@~12",
	}
	value := reflect.ValueOf(m).Elem()
	l := value.NumField()
	assert.Equal(t, 9, l)
	value.Field(0).SetString("aaaaaaaa")
	assert.Equal(t, "aaaaaaaa", value.Field(0).String())
}

func TestReflectStruct(t *testing.T) {
	d := `{
    "MainType":1,
    "ExtType":21,
    "Mode":2,
    "ModeParm":"mcall-6521888999731105792",
    "MSGID":"75684002",
    "TELID":"def",
    "MSG":{
        "vcc_id":"2000196",
        "call_id":"6521888999731105792",
        "caller":"59658059",
        "called":"013619289103",
        "trans_caller":"59658059",
        "trans_called":"013553082092",
        "start_time":"1554939520",
        "ring_time":"1554939528",
        "answer_time":"1554939538",
        "status":"7"
    }
}`

	var m Message
	err := json.Unmarshal([]byte(d), &m)
	assert.NoError(t, err)
	value := reflect.TypeOf(m.MSG).Elem()
	s, _ := value.FieldByName("status")
	assert.Equal(t, "7", s)
}

func TestParseMessage(t *testing.T) {
	conf := config.NewConfig()
	err := conf.Read("../conf.yml")
	assert.NoError(t, err)

	p, err := NewPusher(conf)
	assert.NoError(t, err)

	d := `{"MainType":1,"ExtType":21,"Mode":2,"ModeParm":"mcall-6522301381645316096","MSGID":"44252","TELID":"def","MSG":{"vcc_id":"782","call_id":"6522301381645316096","caller":"01057624343","called":"15201164261","trans_caller":"15201164261","trans_called":"7777","start_time":"1555037834","ring_time":"1555037834","answer_time":"1555037836","hangup_time":"1555037838","status":"8","user_data":{"test":123456}}}`
	msg := amqp.Delivery{
		RoutingKey: "msgproxy.1.21",
		Body:       []byte(d),
	}

	str, err := p.parseMessage(&msg)
	assert.NoError(t, err)
	assert.Equal(t, "15201164261", str)

}

func TestValid(t *testing.T) {
	p := Push{}
	r, ta := p.valid("12345678900")
	assert.Equal(t, true, r)
	assert.Equal(t, "12345678900", ta)

	r, _ = p.valid("123456789000")
	assert.Equal(t, false, r)

	r, _ = p.valid("0103456789000")
	assert.Equal(t, false, r)

	r, _ = p.valid("010345678900")
	assert.Equal(t, false, r)

	r, _ = p.valid("123456789a0")
	assert.Equal(t, false, r)

	r, ta = p.valid("012345678900")
	assert.Equal(t, true, r)
	assert.Equal(t, "12345678900", ta)

}
