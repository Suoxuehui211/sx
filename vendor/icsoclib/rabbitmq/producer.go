package rabbitmq

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"icsoclib/mq"

	log "github.com/alecthomas/log4go"
	"github.com/streadway/amqp"
)

//Producer repsents a rabbitmq broker
type Producer struct {
	uri          string //rabbitmq server url
	exchangeName string //rabbitmq exchange name
	exchangeType string //rabbitmq exchange type
	done         chan struct{}

	conn           *amqp.Connection       //amqp connection
	ch             *amqp.Channel          //amqp sending channel
	chChannelError chan *amqp.Error       //chan notice connection or channel close
	chConnError    chan *amqp.Error       //chan notice connection or channel close
	chReturn       chan amqp.Return       //chan hold returned msg
	chBlocked      chan amqp.Blocking     //chan receive block notice
	chConfirm      chan amqp.Confirmation //chan confirm mode use

	open int32 // Will be 1 if the connection is open, 0 otherwise. Should only be accessed as atomic
	opts Options
}

//NewRabbitmqProducer get a new Producer object
//url:	rabbitmq url: "amqp://guest:guest@192.168.96.6:5672/im"
//exchangeName:	exchange name
//exchangeType:	exchange type, ie: "topic", "fanout"
//chanBufferSize: send chan size
//return value:
func NewRabbitmqProducer(url, exchangeName, exchangeType string, opts ...Option) mq.IProducerWithMultiTopic {
	r := &Producer{
		uri:          url,
		exchangeName: exchangeName,
		exchangeType: exchangeType,
		done:         make(chan struct{}),
		open:         int32(0),
		opts:         newDefaultOptions(),
	}

	for _, option := range opts {
		option(&r.opts)
	}
	return r
}

//Publish produce message
//topic: no use here, for compatible with mq.IProducerWithMultiTopic
//key:	rabbitmq topic mode, topic name
//value:	a message
//return value:
func (r *Producer) Publish(topic, key string, value []byte) error {
	if !r.Connected() {
		return errors.New("not connected yet")
	}
	select {
	case <-r.done:
		return nil
	default:
		return r.publishImpl(key, value)
	}
}

func (r *Producer) connect() error {
	log.Debug("start")
	var (
		err error
	)
	r.conn, err = amqp.DialConfig(r.uri, r.opts.conf)
	if err != nil {
		return err
	}
	return r.connectChannel()
}

func (r *Producer) connectChannel() error {
	var err error
	if r.ch, err = r.conn.Channel(); err != nil {
		return fmt.Errorf("Channel: %s", err.Error())
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
		return fmt.Errorf("ExchangeDeclare %s", err.Error())
	}

	if err = r.ch.Confirm(false); err != nil {
		return fmt.Errorf("Confirm %s", err.Error())
	}
	// Reliable publisher confirms require confirm.select support from the connection.
	if r.chConfirm = r.ch.NotifyPublish(make(chan amqp.Confirmation, r.opts.confirmationChanSize)); r.chConfirm == nil {
		return fmt.Errorf("NotifyPublish return nil")
	}

	if r.chReturn = r.ch.NotifyReturn(make(chan amqp.Return)); r.chReturn == nil {
		return fmt.Errorf("NotifyReturn return nil")
	}

	if r.chConnError = r.conn.NotifyClose(make(chan *amqp.Error)); r.chConnError == nil {
		return fmt.Errorf("conn NotifyClose return nil")
	}

	if r.chChannelError = r.ch.NotifyClose(make(chan *amqp.Error)); r.chChannelError == nil {
		return fmt.Errorf("channel NotifyClose return nil")
	}
	atomic.StoreInt32(&r.open, 1)
	log.Debug("connected rabbitmq, url: %s, exchangeType (%s) exchangeName (%s)", r.uri, r.exchangeType, r.exchangeName)
	return nil
}

//Connected connecte rabbitmq server
func (r *Producer) Connected() bool {
	return (atomic.LoadInt32(&r.open) == 1)
}

func (r *Producer) shutdown() {
	atomic.StoreInt32(&r.open, 0)
	if r.conn != nil {
		r.conn.Close()
		r.conn = nil
	}
}

// Close exit producer
func (r *Producer) Close() error {
	r.done <- struct{}{}
	return nil
}

//implement publish a single message
func (r *Producer) publishImpl(topic string, message []byte) error {
	if r.ch == nil {
		return errors.New("channel nil")
	}

	if err := r.ch.Publish(
		r.exchangeName, // publish to an exchange
		topic,          // routing to 0 or more queues
		true,           // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType:  "text/plain",
			DeliveryMode: r.opts.deliveryMode,
			Body:         message,
		},
	); err != nil {
		return fmt.Errorf("send error:%s routingKey: %s, message: %s", err.Error(), topic, string(message))
	} else {
		log.Debug("tx %s %s", topic, string(message))
	}
	return nil
}

//Process send message to rabbitmq server
func (r *Producer) Process() error {
	var err error
	log.Debug("Process start")
	defer log.Debug("Process end")
	for {
		if err := r.connect(); err != nil {
			log.Error(err)
			time.Sleep(time.Millisecond * 500)
			continue
		}

	publish:
		for {
			select {
			case err := <-r.chConnError: //conn closed received
				log.Error("conn err： %+v", err)
				r.shutdown()
				break publish
			case e := <-r.chChannelError: //channel closed received
				log.Error("channel err： %+v", e)
				r.ch.Close()
				if err = r.connectChannel(); err != nil {
					log.Error(err)
				}
			case cr := <-r.chReturn: //msg returned
				log.Warn("exchange %s, routingKey %s, body %s, replyCode %d, replyText %s", cr.Exchange, cr.RoutingKey, string(cr.Body), cr.ReplyCode, cr.ReplyText)
				// just record if there's no consumer queue
				//if cr.CorrelationId != "" {
				//	id, _ := strconv.Atoi(cr.CorrelationId)
				//	id++
				//	cr.CorrelationId = strconv.Itoa(id)
				//	if id < 3 {
				//		if err := r.Publish("", cr.RoutingKey, cr.Body); err != nil {
				//			log.Error(err)
				//		}
				//	}
				//	//else drop
				//	log.Error("drop %+v", cr)
				//} else {
				//	cr.CorrelationId = "0"
				//}
			case confirmed := <-r.chConfirm:
				if !confirmed.Ack {
					log.Error("failed delivery of delivery %+v", confirmed)
				}
			case <-r.done:
				log.Debug("done, exit Process")
				r.shutdown()
				return nil
			}
		}
	}
}
