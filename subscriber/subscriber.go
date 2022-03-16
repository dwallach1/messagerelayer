package subscriber

import (
	"context"
	"log"
	"messagerelayer/constants"
	"time"
)

// Subscriber reads messages sent from a message relayer
type Subscriber interface {
	Start(context.Context)
	Channel(constants.MessageType) chan constants.Message
	DoneChannel() chan bool
	Type() constants.MessageType
	// helper methods for testing
	Name() string
	ProcessedCount() int
	WaitTime() time.Duration
}

// MockSubscriber is a noop subscriber that just reads and prints the incoming messages
type MockSubscriber struct {
	name           string
	msgType        constants.MessageType
	processedCount int
	waitTime       func() time.Duration
	msgQueues      QueueMap
	done           chan bool
}

// QueueMap stores the associated channels for each message type for a subscriber
type QueueMap map[constants.MessageType]chan constants.Message

// Get retrieves the associated queue map for the parent subscriber
func (q QueueMap) Get(msgType constants.MessageType) chan constants.Message {
	c, ok := q[msgType]
	if !ok {
		return make(chan constants.Message)
	}
	return c
}

// New returns a new subscriber
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

// Name returns the subscribers name
func (ms MockSubscriber) Name() string {
	return ms.name
}

// ProcessedCount returns the number of messages a subscriber processed
func (ms MockSubscriber) ProcessedCount() int {
	return ms.processedCount
}

// WaitTime returns the duration for the subscriber to wait inbetween reading messages that have been broadcasted to it
func (ms MockSubscriber) WaitTime() time.Duration {
	return ms.waitTime()
}

// DoneChannel returns the subscribers done channel so the parent process can wait until it completes to exit
func (ms MockSubscriber) DoneChannel() chan bool {
	return ms.done
}

// Type returns the message type the subscriber was registered with
func (ms MockSubscriber) Type() constants.MessageType {
	return ms.msgType
}

// Channel returns the subscribers associated channel
func (ms MockSubscriber) Channel(msgType constants.MessageType) chan constants.Message {
	return ms.msgQueues.Get(msgType)
}

// Start begins the subscriber to listen for new messages from the message relayer
func (ms *MockSubscriber) Start(ctx context.Context) {
	log.Printf("subscriber %v starting", ms.name)
	for {
		select {
		case msg := <-ms.msgQueues.Get(constants.StartNewRound):
			log.Printf("👨 %v reading new start round message: %+v", ms.name, string(msg.Data))
			ms.processedCount++
		case msg := <-ms.msgQueues.Get(constants.ReceivedAnswer):
			log.Printf("👨 %v reading new recieved answer message: %+v", ms.name, string(msg.Data))
			ms.processedCount++
		case <-ctx.Done():
			log.Printf("closing subscriber %v who processed %v messages", ms.name, ms.processedCount)
			ms.done <- true
			return
		default:
			time.Sleep(ms.waitTime())
		}
	}
}

// NoopSubscriber doesn't read any messages from its queues
type NoopSubscriber struct {
	done      chan bool
	msgQueues QueueMap
}

// NewNoop returns a new instance a NoopSubsciber
func NewNoop(queueSize int) Subscriber {
	queues := QueueMap{}
	queues[constants.ReceivedAnswer] = make(chan constants.Message, queueSize)
	queues[constants.StartNewRound] = make(chan constants.Message, queueSize)
	return &NoopSubscriber{
		msgQueues: queues,
		done:      make(chan bool),
	}
}

// Start begins the subscriber to listen for new messages from the message relayer
func (ns *NoopSubscriber) Start(ctx context.Context) {
	<-ctx.Done()
	ns.done <- true
}

// Name returns the subscribers name
func (ns NoopSubscriber) Name() string {
	return "noop subscriber"
}

// ProcessedCount returns the number of messages a subscriber processed
func (ns NoopSubscriber) ProcessedCount() int {
	return 0
}

// WaitTime returns the duration for the subscriber to wait inbetween reading messages that have been broadcasted to it
func (ns NoopSubscriber) WaitTime() time.Duration {
	return 0 * time.Second
}

// DoneChannel returns the subscribers done channel so the parent process can wait until it completes to exit
func (ns NoopSubscriber) DoneChannel() chan bool {
	return ns.done
}

// Type returns the message type the subscriber was registered with
func (ns NoopSubscriber) Type() constants.MessageType {
	return constants.All
}

// Channel returns the subscribers associated channel
func (ns NoopSubscriber) Channel(msgType constants.MessageType) chan constants.Message {
	return ns.msgQueues.Get(msgType)
}
