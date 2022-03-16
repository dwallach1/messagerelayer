package subscriber

import (
	"context"
	"log"
	"messagerelayer/constants"
	"time"
)

type Subscriber interface {
	Type() constants.MessageType
	Status() string
	Channel(constants.MessageType) chan constants.Message
	Start(context.Context)
}

type MockSubscriber struct {
	name           string
	msgType        constants.MessageType
	processedCount int
	waitTime       func() time.Duration
	msgQueues      QueueMap
}

type QueueMap map[constants.MessageType]chan constants.Message

func (q QueueMap) Get(msgType constants.MessageType) chan constants.Message {
	c, ok := q[msgType]
	if !ok {
		return make(chan constants.Message)
	}
	return c
}

func New(msgType constants.MessageType, waitTime func() time.Duration, queueSize int, name string) Subscriber {
	queues := QueueMap{}
	if msgType == constants.ReceivedAnswer || msgType == constants.All {
		queues[constants.ReceivedAnswer] = make(chan constants.Message, queueSize)
	}
	if msgType == constants.StartNewRound || msgType == constants.All {
		queues[constants.StartNewRound] = make(chan constants.Message, queueSize)
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

func (ms MockSubscriber) Channel(msgType constants.MessageType) chan constants.Message {
	return ms.msgQueues.Get(msgType)
}

func (ms MockSubscriber) Start(ctx context.Context) {
	log.Printf("subscriber %v starting: %+v\n", ms.name, ms.msgQueues)
	for {
		select {
		case msg := <-ms.msgQueues.Get(constants.StartNewRound):
			log.Printf("%v reading new start round message: %+v", ms.name, string(msg.Data))
		case msg := <-ms.msgQueues.Get(constants.ReceivedAnswer):
			log.Printf("%v reading new recieved answer message: %+v", ms.name, string(msg.Data))
		case <-ctx.Done():
			log.Printf("context cancelled detected: closing subscriber %v", ms.name)
			return
			// default:
			// 	log.Printf("no msgs for %v", ms.name)
			// 	time.Sleep(ms.waitTime())
		}
	}
}
