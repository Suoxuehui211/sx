package rabbitmq

import (
	"fmt"
	"sync/atomic"
	"time"

	"icsoclib/mq"

	log "github.com/alecthomas/log4go"
)

//Msg is a node that rabbitmq producer deal with
type Msg struct {
	Topic   string //queue name
	Message []byte //message body
}

//ReliableProducer repsents a rabbitmq broker
type ReliableProducer struct {
	Producer
	chmsg     chan *Msg //buffer hold Msgs
	buf       []*Msg    //send buffer
	confirmed uint64
	//	lastConfirm uint64
}

//NewRabbitmqProducer get a new Producer object
//url:	rabbitmq url: "amqp://guest:guest@192.168.96.6:5672/im"
//exchangeName:	exchange name
//exchangeType:	exchange type, ie: "topic", "fanout"
//chanBufferSize: send chan size
//return value:
func NewReliableProducer(url, exchangeName, exchangeType string, opts ...Option) mq.IProducerWithMultiTopic {
	p := NewRabbitmqProducer(url, exchangeName, exchangeType, opts...)
	if pp, ok := p.(*Producer); ok {
		for _, option := range opts {
			option(&pp.opts)
		}
		r := ReliableProducer{
			Producer:  *pp,
			chmsg:     make(chan *Msg, pp.opts.chanBufferSize),
			buf:       make([]*Msg, 0, pp.opts.gotSize),
			confirmed: 0,
		}
		return &r
	}

	return nil
}

//Publish produce message
//topic: no use here, for compatible with mq.IProducerWithMultiTopic
//key:	rabbitmq topic mode, topic name
//value:	a message
//return value:
func (r *ReliableProducer) Publish(topic, key string, value []byte) error {
	//if !r.Connected() {
	//	return errors.New("not connected yet")
	//}

	msg := Msg{key, value}
	if time.Duration(r.Producer.opts.sendTimeout) == 0 {
		select {
		case r.chmsg <- &msg:
		case <-r.done:
			return nil
		}
	} else {
		select {
		case r.chmsg <- &msg:
		case <-time.After(r.Producer.opts.sendTimeout):
			err := fmt.Errorf("send buffer chan full, throw value, routingkey: %s, value: %s", key, string(value))
			return err
		case <-r.done:
			return nil
		}
	}
	return nil
}

func (r *ReliableProducer) loopSend() {
	for {
		select {
		case <-r.done:
			log.Debug("exit loop")
			return
		default:
		}

		buffer := r.buf[:0]
	getdata:
		for {
			select {
			case m := <-r.chmsg:
				buffer = append(buffer, m)
				if len(buffer) == r.Producer.opts.gotSize {
					break getdata
				}
			case <-time.After(time.Microsecond):
				break getdata
			}
		}

		r.iterateSend(buffer)
	}
}

func (r *ReliableProducer) iterateSend(buffer []*Msg) {
	l := len(buffer)
	if l == 0 {
		return
	}
	log.Debug("buffer will be send: %d", l)
	ret := r.sendData(buffer)
	if l > ret {
		buf := buffer[ret:]
		r.iterateSend(buf)
	}
}

//sendData returns number of successfully delivery message
func (r *ReliableProducer) sendData(buffer []*Msg) int {
	var (
		sendBuffer  []*Msg
		confirmed   uint64
		lastConfirm uint64
		sent        int
	)

	sendBuffer = buffer[0:]
	sent = len(sendBuffer)
	if sent > 0 {
		//atomic.StoreUint64(&r.lastConfirm, atomic.LoadUint64(&r.confirmed))
		lastConfirm = atomic.LoadUint64(&r.confirmed)
		for i, v := range sendBuffer {
			if err := r.Producer.Publish("", v.Topic, v.Message); err != nil {
				log.Error(err)
				sent = i //send successfully num
				break
			}
		}

		if sent == 0 {
			return 0
		}

		//wait for confirm
		now := time.Now()

		for {
			confirmed = atomic.LoadUint64(&r.confirmed)
			if uint64(sent) <= confirmed-lastConfirm {
				log.Debug("send out, sent:%d, confirmed:%d", sent, confirmed-lastConfirm)
				return sent
			}
			if time.Now().Sub(now) > r.Producer.opts.maxWaitingDur {
				log.Debug("confirm timeout, sent:%d, confirmed:%d", sent, confirmed-lastConfirm)
				return int(confirmed - lastConfirm)
			}
			time.Sleep(time.Millisecond * 10)
		}
	}
	return 0
}

//Process send message to rabbitmq server
func (r *ReliableProducer) Process() error {
	var err error
	log.Debug("Process start")
	defer log.Debug("Process end")
	go r.loopSend()
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
			case confirmed := <-r.chConfirm:
				if confirmed.Ack {
					atomic.AddUint64(&r.confirmed, 1)
				}
			case <-r.done:
				log.Debug("done, exit Process")
				r.shutdown()
				return nil
			}
		}
	}
}
