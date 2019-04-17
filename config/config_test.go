package config

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	conf := NewConfig()
	err := conf.Read("../conf.yml")
	assert.NoError(t, err)
	assert.Equal(t, "01057624343", conf.Caller)
	fmt.Printf("%+v", conf)
}
func TestWatcher_Watch(t *testing.T) {
	TestWriteJson(t)
	conf := NewConfig()
	err := conf.Read("../conf.yml")
	assert.NoError(t, err)
	w, err := NewWatcher(conf.EtcdURL, conf)
	assert.NoError(t, err)
	go w.Watch("/liupengtest/vccid")
	time.Sleep(3 * time.Second)
	f, err := conf.GetSmsConf(123)
	assert.NoError(t, err)
	assert.Equal(t, 5024, f.Tempid)
	assert.Equal(t, "ClientName", f.Param)

	f, err = conf.GetSmsConf(456)
	assert.NoError(t, err)
	assert.Equal(t, 5025, f.Tempid)
	assert.Equal(t, false, f.Enable)

}

func TestWriteJson(t *testing.T) {
	c, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"192.168.96.6:2379"},
		DialTimeout: 3 * time.Second,
	})
	assert.NoError(t, err)

	f := FlashSMS{
		ID:      1,
		VccID:   782,
		Enable:  true,
		Msgflag: 1,
		Smsconf: 1,
		Tempid:  5024,
		Vendor:  10,
		Param:   "ClientName",
	}
	buf, _ := json.Marshal(f)
	key := "/shanxinConfig/vccid/" + "782"
	resp, err := c.Put(context.Background(), key, string(buf))
	assert.NoError(t, err)
	fmt.Println(resp)

	f = FlashSMS{
		ID:      2,
		VccID:   456,
		Enable:  false,
		Msgflag: 1,
		Smsconf: 1,
		Tempid:  5025,
		Vendor:  10,
		Param:   "ClientName",
	}
	buf, _ = json.Marshal(f)
	key = "/shanxinConfig/vccid/" + "456"
	a := string(buf)
	fmt.Println(a)
	resp, err = c.Put(context.Background(), key, string(buf))
	assert.NoError(t, err)
	fmt.Println(resp)

}

func TestWatcherWatchDelete(t *testing.T) {
	TestWriteJson(t)
	conf := NewConfig()
	err := conf.Read("../conf.yml")
	assert.NoError(t, err)
	w, err := NewWatcher(conf.EtcdURL, conf)
	assert.NoError(t, err)
	go func() {
		time.Sleep(3 * time.Second)
		c, err := clientv3.New(clientv3.Config{
			Endpoints:   []string{"192.168.96.6:2379"},
			DialTimeout: 3 * time.Second,
		})
		assert.NoError(t, err)
		_, err = c.Delete(context.TODO(), "/liupengtest/vccid/123")
		assert.NoError(t, err)
		time.Sleep(1 * time.Second)
		f, err := conf.GetSmsConf(123)
		assert.Equal(t, "none exist vcc_id 123", err.Error())
		assert.Equal(t, nil, f)
		os.Exit(0)
	}()
	w.Watch("/liupengtest/vccid")
}
