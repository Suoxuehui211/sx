package main

import (
	"flag"
	log "github.com/alecthomas/log4go"
	"icsoclib/rabbitmq"
	"sx/config"
	"sx/push"
)

var (
	confFile string
)

func init() {
	flag.StringVar(&confFile, "conf", "./conf.yml", " set config file path")
}

func main() {

	flag.Parse()

	defer log.Close()
	conf := config.NewConfig()
	if err := conf.Read(confFile); err != nil {
		panic(err)
	}
	log.Debug("%+v", conf)

	w, err := config.NewWatcher(conf.EtcdURL, conf)
	if err != nil {
		panic(err)
	}
	go w.Watch(conf.PrefixDir)

	t, err := push.NewPusher(conf)
	if err != nil {
		panic(err)
	}
	consumer := rabbitmq.NewRabbitmqConsumer(
		conf.RabbitmqAddrs,
		conf.Exchange, "topic",
		conf.QueueName,
		conf.RoutingKey,
		rabbitmq.HandlerV2(t.ReadMsg))
	defer consumer.Close()
	consumer.Process()
}
