package config

import (
	"fmt"
	"github.com/spf13/viper"
	"sync"
)

//CREATE TABLE `cc_conf_flashsms` (
//`id` int(10) unsigned NOT NULL ,
//`vcc_id` int(10) unsigned NOT NULL COMMENT '企业ID',
//`enable` tinyint(3) unsigned NOT NULL DEFAULT '1' COMMENT '是否启用',
//`msgflag` bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT '推送消息标识',
//`smsconf` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '推送配置',
//`tempid` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '模板ID',
//`vendor` tinyint(3) unsigned NOT NULL DEFAULT '1' COMMENT '运营商0为全部',
//`param` varchar(1024) NOT NULL COMMENT '模板参数，逗号分隔',
//PRIMARY KEY (`id`)
//) ENGINE=InnoDB DEFAULT CHARSET=utf8;

type FlashSMS struct {
	ID      int
	VccID   int `json:"vcc_id"`
	Enable  bool
	Msgflag int
	Smsconf int
	Tempid  int
	Vendor  int
	Param   string
}

//Config for application use
type Config struct {
	Interfacename string //network interface, ie: eth0
	PprofPort     string //for pprof use
	RabbitmqAddrs string //rabbitmq address
	Exchange      string
	QueueName     string
	RoutingKey    []string

	//RedisAddr    string //redis
	//RedisDbIndex int
	//RedisMaxConn int

	EtcdURL   []string
	PrefixDir string

	Key       string
	Operid    string
	Caller    string
	Tempid    string
	Enterid   string
	Enterpass string
	Args      string
	URL       string

	//base on upper config
	PprofAddrs string

	lock         sync.RWMutex
	FlashSMSConf map[int]*FlashSMS
}

//NewConfig return a config struct
func NewConfig() *Config {
	return &Config{
		lock:         sync.RWMutex{},
		FlashSMSConf: make(map[int]*FlashSMS),
	}
}

//Read loads configure file, currently only supports yaml file
func (c *Config) Read(file string) error {
	var (
		err error
		vip *viper.Viper
	)
	vip = viper.New()
	vip.SetConfigFile(file)
	err = vip.ReadInConfig()
	if err != nil {
		return err
	}

	//c.PprofPort = vip.GetString("pprof.port")
	//c.Interfacename = vip.GetString("interfacename")
	c.RabbitmqAddrs = vip.GetString("rabbitmq.addrs")
	c.QueueName = vip.GetString("rabbitmq.queuename")
	c.Exchange = vip.GetString("rabbitmq.exchange")
	c.RoutingKey = vip.GetStringSlice("rabbitmq.routingKey")

	//c.RedisAddr = vip.GetString("redis.addrs")
	//c.RedisDbIndex = vip.GetInt("redis.db")
	//c.RedisMaxConn = vip.GetInt("redis.maxConn")
	c.EtcdURL = vip.GetStringSlice("etcd.addrs")
	c.PrefixDir = vip.GetString("etcd.prefixDir")
	c.URL = vip.GetString("shanxin.url")
	c.Key = vip.GetString("shanxin.key")
	c.Operid = vip.GetString("shanxin.operid")
	c.Caller = vip.GetString("shanxin.caller")
	c.Tempid = vip.GetString("shanxin.tempid")
	c.Enterid = vip.GetString("shanxin.enterid")
	c.Enterpass = vip.GetString("shanxin.enterpass")
	c.Args = vip.GetString("shanxin.args")

	//ip, err = utility.GetLocalIP(c.Interfacename)
	//if err != nil {
	//	return err
	//}
	//c.PprofAddrs = ip + ":" + c.PprofPort
	return nil
}

func (c *Config) GetSmsConf(vccid int) (*FlashSMS, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if v, ok := c.FlashSMSConf[vccid]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("none exist vcc_id %d", vccid)
}

func (c *Config) SetSmsConf(f *FlashSMS) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.FlashSMSConf[f.VccID] = f
	return nil
}

func (c *Config) DelSmsConf(vccID int) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	for k := range c.FlashSMSConf {
		if k == vccID {
			delete(c.FlashSMSConf, vccID)
		}
	}
	return nil
}
