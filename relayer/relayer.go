package relayer

import (
	"context"
	"log"
	"messagerelayer/constants"
)

const queueSize = 10

type Relayer interface {
	Start(context.Context)
	Read() (constants.Message, error)
	Enqueue(constants.Message)
	SubscribeToMessages(msgType constants.MessageType, ch chan constants.Message)
	DoneChannel() chan bool
	// helpers for test validation
	QueuedMsgs() int
	BroadcastedMsgs() int
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
	done                 chan bool
}

func (mr *MessageRelayer) Start(ctx context.Context) {
	log.Printf("message relayer starting with %v RecievedAnswer subscribers and %v StartNewRound subsribers", len(mr.subscribers[constants.ReceivedAnswer]), len(mr.subscribers[constants.StartNewRound]))
	for {
		select {
		case msg := <-mr.startRoundQueue:
			subscriberChannels := mr.subscribers[constants.StartNewRound]
			log.Printf("🔊 boradcasting start new round message")
			for _, subscriberChannel := range subscriberChannels {
				if channelIsFull(subscriberChannel) {
					log.Printf("discarding message of full StartNewRound channel")
					discardChannelMsg(subscriberChannel)
				}
				mr.broadcastedMsgsCount++
				subscriberChannel <- msg
			}
		case msg := <-mr.recievedAnswerQueue:
			subscriberChannels := mr.subscribers[constants.ReceivedAnswer]
			log.Printf("🔊 boradcasting recieved answer message")
			for _, subscriberChannel := range subscriberChannels {
				if channelIsFull(subscriberChannel) {
					log.Printf("discarding message of full ReceivedAnswer channel")
					discardChannelMsg(subscriberChannel)
				}
				mr.broadcastedMsgsCount++
				subscriberChannel <- msg
			}
		case <-ctx.Done():
			log.Printf("closing message relayer:: queued: %v messages, broadcasted: %v messages", mr.queuesMsgsCount, mr.broadcastedMsgsCount)
			mr.done <- true
			return
		}
	}
}

func (mr MessageRelayer) Read() (constants.Message, error) {
	return mr.socket.Read()
}

func (mr *MessageRelayer) Enqueue(msg constants.Message) {
	if msg.Type == constants.ReceivedAnswer || msg.Type == constants.All {
		mr.recievedAnswerQueue <- msg
		mr.queuesMsgsCount++
		log.Println("⤴️  added new message to recieved answer queue")
	}
	if msg.Type == constants.StartNewRound || msg.Type == constants.All {
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

func (mr MessageRelayer) QueuedMsgs() int {
	return mr.queuesMsgsCount
}

func (mr MessageRelayer) BroadcastedMsgs() int {
	return mr.broadcastedMsgsCount
}

func channelIsFull(ch chan constants.Message) bool {
	return len(ch) == cap(ch)
}

func discardChannelMsg(ch chan constants.Message) {
	<-ch
}
