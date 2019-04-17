package mq

//IProducer repsents a messge queue broker
type IProducer interface {
	//Publish publishes a message
	Publish(message []byte) error
	//Process loops process message
	Process() error
	//Close release resource
	Close() error
}

//IProducerWithMultiTopic repsents a messge queue broker
type IProducerWithMultiTopic interface {
	//Publish publishes a message
	Publish(topic, key string, value []byte) error
	//Process loops process message
	Process() error
	//Close release resource
	Close() error
}

//IConsumer repsents a queue consumer
type IConsumer interface {
	// //Consume returns a channel to store messages
	// Consume() <-chan []byte
	//Process loops process message
	Process() error
	//Close release resource
	Close() error
	//ID identity this consumer
	ID() string
	//AppendBinding append routingkey/topic to bind
	AppendBinding(key ...string) error
}

//ConsumeMessageHandler handler handles consuming message
type ConsumeMessageHandler func(msg []byte) error
