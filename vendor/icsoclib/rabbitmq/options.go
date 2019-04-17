package rabbitmq

import (
	"fmt"
	"github.com/streadway/amqp"
	"icsoclib/mq"
	"net"
	"time"
)

//time setting
var (
	QueueExpireTime          = 300000
	DefaultConnectionTimeout = 30 * time.Second
	DefaultRWTimeout         = 3 * time.Second
	DefaultHeartbeat         = 60 * time.Second
)

type Options struct {
	//producer
	confirmationChanSize int   //confirm chan size. The capacity of the chan Confirmation must be at least as large as the number of outstanding publishings.
	deliveryMode         uint8 //delivery mode, transient (0 or 1) or Persistent (2)

	//reliableProducer
	sendTimeout    time.Duration //publish timeout
	chanBufferSize int           //buffer size
	gotSize        int           //how many messages to get once a time
	maxWaitingDur  time.Duration //max waiting time for confirm

	//consumer
	conf amqp.Config // Config is used in DialConfig and Open to specify the desired tuning
	//exchangeName     string           //rabbitmq exchange name
	//exchangeType     string           //rabbitmq exchange type
	//queueName        string           //rabbitmq queue name
	//routingKey		[]string			//routing key
	autoDelQue      bool                     //delete queue when there's no consumer over some time
	queueExpireTime int                      //queue expire time if autoDelQue = ture
	autoAck         bool                     //server auto ack after sent messages out or must continue send after receiving client ack
	prefetchCount   int                      //refer to Qos
	handler         mq.ConsumeMessageHandler //message handler
	handlerV2       ConsumeMessageHandlerV2  //message handler
	retry           int                      //retry count
}

type ConsumeMessageHandlerV2 func(msg *amqp.Delivery) error

type Option func(c *Options)

//ConfirmationChanSize set confirmation channel size
func ConfirmationChanSize(confirmationChanSize int) Option {
	return func(c *Options) {
		if confirmationChanSize > 0 {
			c.confirmationChanSize = confirmationChanSize
		}
	}
}

//PersistentDelivery set whether persistent delivery
//Publishing messages as persistent affects performance
func PersistentDelivery(p bool) Option {
	return func(c *Options) {
		if p {
			c.deliveryMode = amqp.Persistent
		} else {
			c.deliveryMode = amqp.Transient
		}
	}
}

//SendTimeout set whether send blocked
func SendTimeout(timeout time.Duration) Option {
	return func(c *Options) {
		if timeout > 0 {
			c.sendTimeout = timeout
		}
	}
}

//ChanBuffer set buffer size
func ChanBuffer(chanBufferSize int) Option {
	return func(c *Options) {
		if chanBufferSize > 0 {
			c.chanBufferSize = chanBufferSize
		}
	}
}

//SendSize set send buffer that send once a time
func SendSize(s int) Option {
	return func(c *Options) {
		if s > 0 {
			c.gotSize = s
		}
	}
}

//MaxWaitingTime max waiting time for confirm
func MaxWaitingTime(t time.Duration) Option {
	return func(c *Options) {
		if t > 0 {
			c.maxWaitingDur = t
		}
	}
}

// Config set dail config
func Config(conf amqp.Config) Option {
	return func(c *Options) {
		c.conf = conf
		if conf.Dial == nil {
			c.conf.Dial = defaultDial
		}
	}
}

//AutoDelQueue delete the queue after queueExpireTime milliseconds
func AutoDelQueue(autoDelQue bool, queueExpireTime int) Option {
	return func(c *Options) {
		c.autoDelQue = autoDelQue
		if autoDelQue {
			c.queueExpireTime = queueExpireTime
		}
	}
}

//AutoAck set rabbitmq autoack
func AutoAck(autoAck bool) Option {
	return func(c *Options) { c.autoAck = autoAck }
}

//PrefetchCount set prefetchCount for qos use
func PrefetchCount(prefetchCount int) Option {
	return func(c *Options) { c.prefetchCount = prefetchCount }
}

//Handler set a handler that for consume message
func Handler(handler mq.ConsumeMessageHandler) Option {
	return func(c *Options) {
		if handler != nil {
			c.handler = handler
		}
	}
}

//HandlerV2 set a handler that for consume message
func HandlerV2(handler ConsumeMessageHandlerV2) Option {
	return func(c *Options) {
		if handler != nil {
			c.handlerV2 = handler
		}
	}
}

//Retry retry count if handling fail
func Retry(r int) Option {
	return func(c *Options) {
		if r > 0 {
			c.retry = r
		}
	}
}

func defaultDial(network, addr string) (net.Conn, error) {
	conn, err := net.DialTimeout(network, addr, DefaultConnectionTimeout)
	if err != nil {
		return nil, err
	}

	// Heartbeating hasn't started yet, don't stall forever on a dead server.
	// A deadline is set for TLS and AMQP handshaking. After AMQP is established,
	// the deadline is cleared in openComplete.
	if err := conn.SetDeadline(time.Now().Add(DefaultRWTimeout)); err != nil {
		return nil, err
	}

	return conn, nil
}

func newDefaultOptions() Options {
	return Options{
		autoDelQue:           false,
		queueExpireTime:      QueueExpireTime,
		autoAck:              false,
		prefetchCount:        256,
		confirmationChanSize: 256,

		deliveryMode: amqp.Persistent,
		//handler: func(msg []byte) error {
		//	fmt.Println(string(msg))
		//	return nil
		//},
		handlerV2: func(msg *amqp.Delivery) error {
			fmt.Println(string(msg.Body))
			return nil
		},

		sendTimeout:    0,
		chanBufferSize: 1024,
		gotSize:        1024,
		maxWaitingDur:  time.Second * 5,
		retry:          2,
		conf: amqp.Config{
			Heartbeat: DefaultHeartbeat,
			Dial:      defaultDial,
		},
	}
}
