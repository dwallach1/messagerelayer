package subscriber

import (
	"fmt"
	"log"
	"messagerelayer/constants"
	"messagerelayer/relayer"
	"time"
)

type Subscriber interface {
	Type() constants.MessageType
	Status() string
	Channel(constants.MessageType) chan relayer.Message
	Start()
}

type MockSubscriber struct {
	name           string
	msgType        constants.MessageType
	processedCount int
	waitTime       func() time.Duration
	msgQueues      QueueMap
}

type QueueMap map[constants.MessageType]chan relayer.Message

func (q QueueMap) Get(msgType constants.MessageType) chan relayer.Message {
	c, ok := q[msgType]
	if !ok {
		return make(chan relayer.Message)
	}
	return c
}

func New(msgType constants.MessageType, waitTime func() time.Duration, queueSize int, name string) Subscriber {
	queues := QueueMap{}
	if msgType == constants.ReceivedAnswer || msgType == constants.All {
		queues[constants.ReceivedAnswer] = make(chan relayer.Message, queueSize)
	}
	if msgType == constants.StartNewRound || msgType == constants.All {
		queues[constants.StartNewRound] = make(chan relayer.Message, queueSize)
	}
	return &MockSubscriber{
		name:           name,
		processedCount: 0,
		waitTime:       waitTime,
		msgQueues:      queues,
	}
}

func (ms MockSubscriber) Type() constants.MessageType {
	return ms.msgType
}

func (ms MockSubscriber) Status() string {
	return "active"
}

func (ms MockSubscriber) Channel(msgType constants.MessageType) chan relayer.Message {
	return ms.msgQueues[msgType]
}

func (ms MockSubscriber) Start() {
	fmt.Printf("subscriber %v starting...\n", ms.name)
	for {
		select {
		case msg := <-ms.msgQueues.Get(constants.StartNewRound):
			log.Printf("%v reading new start round message: %+v", ms.name, string(msg.Data))
		case msg := <-ms.msgQueues.Get(constants.ReceivedAnswer):
			log.Printf("%v reading new recieved answer message: %+v", ms.name, string(msg.Data))
		}
	}
}
