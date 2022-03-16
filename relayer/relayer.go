package relayer

import (
	"context"
	"log"
	"messagerelayer/constants"
	"messagerelayer/utils"
	"time"
)

const queueSize = 10

var ReadInterval = 5 * time.Second

type WorkSummary struct {
	QueuedMsgs      int // successfully added messages to queue to be broadcasted
	BroadcastedMsgs int // successfully broadcasted to a subscriber
	DiscardedMsgs   int // queus full so we discarded older messages
	SkippedMsgs     int // subscriber busy so we dropped the message
}

type Relayer interface {
	Start(context.Context)
	Read() (constants.Message, error)
	Enqueue(constants.Message)
	SubscribeToMessages(msgType constants.MessageType, ch chan constants.Message)
	DoneChannel() chan bool
	// helpers for test validation
	Summary() WorkSummary
}

type NetworkSocket interface {
	Read() (constants.Message, error)
}

func NewMessageRelayer(socket NetworkSocket) Relayer {
	return &MessageRelayer{
		socket:               socket,
		startRoundQueue:      make(chan constants.Message, queueSize),
		recievedAnswerQueue:  make(chan constants.Message, queueSize),
		subscribers:          make(map[constants.MessageType][]chan constants.Message),
		queuesMsgsCount:      0,
		broadcastedMsgsCount: 0,
		discardedMsgsCount:   0,
		skippedMsgCount:      0,
		done:                 make(chan bool),
	}
}

type MessageRelayer struct {
	socket               NetworkSocket
	startRoundQueue      chan constants.Message
	recievedAnswerQueue  chan constants.Message
	subscribers          map[constants.MessageType][]chan constants.Message // message type -> array of message channels
	queuesMsgsCount      int
	broadcastedMsgsCount int
	discardedMsgsCount   int
	skippedMsgCount      int
	done                 chan bool
}

func (mr *MessageRelayer) Start(ctx context.Context) {
	log.Printf("message relayer starting with %v RecievedAnswer subscribers and %v StartNewRound subsribers", len(mr.subscribers[constants.ReceivedAnswer]), len(mr.subscribers[constants.StartNewRound]))
	for {
		select {
		// start round queue takes precedent over the recieved answer queue
		case msg := <-mr.startRoundQueue:
			subscriberChannels := mr.subscribers[constants.StartNewRound]
			log.Printf("🔊 broadcasting start new round message")
			for _, subscriberChannel := range subscriberChannels {
				if utils.ChannelIsFull(subscriberChannel) {
					mr.skippedMsgCount++
					log.Printf("subscriber busy: detected full StartNewRound subscriber channel: skipping broadcast")
					continue
				}
				mr.broadcastedMsgsCount++
				subscriberChannel <- msg
			}
		case msg := <-mr.recievedAnswerQueue:
			subscriberChannels := mr.subscribers[constants.ReceivedAnswer]
			log.Printf("🔊  broadcasting recieved answer message")
			for _, subscriberChannel := range subscriberChannels {
				if utils.ChannelIsFull(subscriberChannel) {
					mr.skippedMsgCount++
					log.Printf("subscriber busy: detected full ReceivedAnswer subscriber channel: skipping broadcast")
					continue
				}
				mr.broadcastedMsgsCount++
				subscriberChannel <- msg
			}
		case <-ctx.Done():
			log.Printf("closing message relayer:: queued: %v messages, broadcasted: %v messages", mr.queuesMsgsCount, mr.broadcastedMsgsCount)
			mr.done <- true
			return
		default:
			time.Sleep(ReadInterval)
		}
	}
}

func (mr MessageRelayer) Read() (constants.Message, error) {
	return mr.socket.Read()
}

func (mr *MessageRelayer) Enqueue(msg constants.Message) {
	if msg.Type == constants.ReceivedAnswer || msg.Type == constants.All {
		// if our queues are full, we need to discard a message so we can keep most recent messages in queue
		if utils.ChannelIsFull(mr.recievedAnswerQueue) {
			utils.DiscardChannelMsg(mr.recievedAnswerQueue)
			mr.discardedMsgsCount++
		}
		mr.recievedAnswerQueue <- msg
		mr.queuesMsgsCount++
		log.Println("⤴️  added new message to recieved answer queue")
	}
	if msg.Type == constants.StartNewRound || msg.Type == constants.All {
		// if our queues are full, we need to discard a message so we can keep most recent messages in queue
		if utils.ChannelIsFull(mr.startRoundQueue) {
			utils.DiscardChannelMsg(mr.startRoundQueue)
			mr.discardedMsgsCount++
		}
		mr.startRoundQueue <- msg
		mr.queuesMsgsCount++
		log.Println("⤴️  added new message to start round queue")
	}
}

func (mr *MessageRelayer) SubscribeToMessages(msgType constants.MessageType, ch chan constants.Message) {
	if msgType == constants.All {
		mr.subscribers[constants.ReceivedAnswer] = append(mr.subscribers[constants.ReceivedAnswer], ch)
		mr.subscribers[constants.StartNewRound] = append(mr.subscribers[constants.StartNewRound], ch)
		return
	}
	mr.subscribers[msgType] = append(mr.subscribers[msgType], ch)
}

func (mr MessageRelayer) DoneChannel() chan bool {
	return mr.done
}

func (mr MessageRelayer) Summary() WorkSummary {
	return WorkSummary{
		QueuedMsgs:      mr.queuesMsgsCount,
		BroadcastedMsgs: mr.broadcastedMsgsCount,
		DiscardedMsgs:   mr.discardedMsgsCount,
		SkippedMsgs:     mr.skippedMsgCount,
	}
}
