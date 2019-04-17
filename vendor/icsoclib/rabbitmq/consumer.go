package rabbitmq

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"icsoclib/mq"

	log "github.com/alecthomas/log4go"
	"github.com/streadway/amqp"
)

//Consumer repsents a rabbitmq consumer
type Consumer struct {
	uri          string           //rabbitmq server url
	exchangeName string           //rabbitmq exchange name
	exchangeType string           //rabbitmq exchange type
	queueName    string           //rabbitmq queue name
	routingKey   []string         //rabbitmq routing key
	conn         *amqp.Connection //amqp connection
	connError    chan *amqp.Error
	ch           *amqp.Channel //amqp receive channel
	done         chan struct{}
	opts         Options
}

//NewRabbitmqConsumer get a new Consumer object
//url:	rabbitmq url: "amqp://guest:guest@192.168.96.6:5672/im"
//exchangeName:	exchange name
//exchangeType:	exchange type, ie: "topic", "fanout"
//queueName: queue name
//routingKey: routing key
//opts: Consumer Option
//return value:
func NewRabbitmqConsumer(uri, exchangeName, exchangeType, queueName string, routingKey []string, opts ...Option) mq.IConsumer {
	r := &Consumer{
		uri:          uri,
		exchangeName: exchangeName,
		exchangeType: exchangeType,
		queueName:    queueName,
		routingKey:   routingKey,
		done:         make(chan struct{}),
		connError:    make(chan *amqp.Error),
		opts:         newDefaultOptions(),
	}

	for _, option := range opts {
		option(&r.opts)
	}

	return r
}

// //Consume returns a channel holds all data received from rabbitmq
// func (r *Consumer) Consume() <-chan []byte {
// 	return (<-chan []byte)(r.msgRecv)
// }

//Process receive messages from rabbitmq
func (r *Consumer) Process() error {
	var (
		err          error
		deliverychan <-chan amqp.Delivery
	)
	log.Debug("Consumer Process begin")
	defer log.Warn("Consumer Process end")

	for {
		//connect
		if err := r.connect(); err != nil {
			log.Error("%s, url:%s", err.Error(), r.uri)
			time.Sleep(time.Millisecond * 500)
			continue
		}
		if deliverychan, err = r.ch.Consume(
			r.queueName,    // queue
			"",             // consumer
			r.opts.autoAck, // auto-ack
			false,          // exclusive
			false,          // no local
			false,          // no wait
			nil,
			//				amqp.Table{}, // args
		); err != nil {
			log.Error(err)
			continue
		}
	consume:
		for {
			select {
			case msg := <-deliverychan:
				for i := 0; i < r.opts.retry; i++ {
					if err = r.opts.handlerV2(&msg); err == nil {
						break
					} else {
						log.Error("%s, err:%s", string(msg.Body), err.Error())
					}
				}
				if false == r.opts.autoAck {
					if err = msg.Ack(false); err != nil {
						log.Error(err)
					}
				}
			case amqpError := <-r.connError:
				log.Error(amqpError)
				r.close()
				break consume
			case <-r.done:
				log.Debug("done, exit Process")
				r.close()
				return nil
			}
		}

	}
}

func (r *Consumer) close() {
	log.Debug("Close start")
	defer log.Debug("Close end")
	if r.conn != nil {
		r.conn.Close()
		r.conn = nil
	}
}

//ID identity this consumer
func (r *Consumer) ID() string {
	return r.uri
}

//Close exit consumer
func (r *Consumer) Close() error {
	r.done <- struct{}{}
	return nil
}

//AppendBinding append routing key to bind
func (r *Consumer) AppendBinding(key ...string) error {
	if r.ch == nil {
		return errors.New("channel closed")
	}
	if r.routingKey == nil {
		r.routingKey = make([]string, 0, 16)
	}
	for _, v := range key {
		r.routingKey = append(r.routingKey, v)
		if err := r.ch.QueueBind(r.queueName, v, r.exchangeName, false, nil); err != nil {
			log.Error(fmt.Sprintf("bingding key %s %s", v, err.Error()))
			return err
		}
	}
	return nil
}

func (r *Consumer) connect() error {
	log.Warn("start")
	var (
		err error
	)
	r.conn, err = amqp.DialConfig(r.uri, r.opts.conf)
	if err != nil {
		return err
	}

	if r.ch, err = r.conn.Channel(); err != nil {
		return err
	}

	if r.connError = r.conn.NotifyClose(make(chan *amqp.Error)); r.connError == nil {
		return errors.New("conn.NotifyClose fail")
	}

	if err = r.ch.ExchangeDeclare(
		r.exchangeName, // name
		r.exchangeType, // type
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // noWait
		nil,            // arguments
	); err != nil {
		return err
	}

	if r.opts.autoDelQue {
		//The value of the x-expires argument or expires policy describes the expiration period in milliseconds.
		_, err = r.ch.QueueDeclare(r.queueName, true, false, false, false, amqp.Table{"x-expires": int32(r.opts.queueExpireTime)})
	} else {
		_, err = r.ch.QueueDeclare(r.queueName, true, false, false, false, nil)
	}
	if err != nil {
		log.Error(err)
		return err
	}

	for _, v := range r.routingKey {
		if err = r.ch.QueueBind(r.queueName, v, r.exchangeName, false, nil); err != nil {
			log.Error(fmt.Sprintf("%s %s", v, err.Error()))
			return err
		}
	}

	if err = r.ch.Qos(r.opts.prefetchCount, 0, false); err != nil {
		log.Error(err)
		return err
	}
	log.Warn("connected rabbitmq, url: %s, exchangeType (%s) exchangeName (%s) queueName (%s)  routingKey (%s)", r.uri, r.exchangeType, r.exchangeName, r.queueName, strings.Join(r.routingKey, " "))
	return nil
}
