package subscriber

import (
	"context"
	"log"
	"messagerelayer/constants"
	"time"
)

type Subscriber interface {
	Name() string
	ProcessedCount() int
	WaitTime() time.Duration
	DoneChannel() chan bool
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
	done           chan bool
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
		msgType:        msgType,
		done:           make(chan bool),
	}
}

func (ms MockSubscriber) Name() string {
	return ms.name
}

func (ms MockSubscriber) ProcessedCount() int {
	return ms.processedCount
}

func (ms MockSubscriber) WaitTime() time.Duration {
	return ms.waitTime()
}

func (ms MockSubscriber) DoneChannel() chan bool {
	return ms.done
}

func (ms MockSubscriber) Type() constants.MessageType {
	return ms.msgType
}

func (ms MockSubscriber) Status() string {
	return "active"
}

func (ms MockSubscriber) Channel(msgType constants.MessageType) chan constants.Message {
	log.Printf("getChannel for %v has cap of %v", ms.name, len(ms.msgQueues.Get(msgType)))
	return ms.msgQueues.Get(msgType)
}

func (ms *MockSubscriber) Start(ctx context.Context) {
	log.Printf("subscriber %v starting: %+v\n", ms.name, ms.msgQueues)
	for {
		select {
		case msg := <-ms.msgQueues.Get(constants.StartNewRound):
			log.Printf("<<>> %v reading new start round message: %+v", ms.name, string(msg.Data))
			ms.processedCount++
		case msg := <-ms.msgQueues.Get(constants.ReceivedAnswer):
			log.Printf("<<>> %v reading new recieved answer message: %+v", ms.name, string(msg.Data))
			ms.processedCount++
		case <-ctx.Done():
			log.Printf("context cancelled detected: closing subscriber %v who processed %v messages", ms.name, ms.processedCount)
			ms.done <- true
			return
		}
	}
}
