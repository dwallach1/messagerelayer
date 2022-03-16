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
}

type NetworkSocket interface {
	Read() (constants.Message, error)
}

func NewMessageRelayer(socket NetworkSocket) Relayer {
	return &MessageRelayer{
		socket:              socket,
		startRoundQueue:     make(chan constants.Message, queueSize),
		recievedAnswerQueue: make(chan constants.Message, queueSize),
		subscribers:         make(map[constants.MessageType][]chan constants.Message),
	}
}

type MessageRelayer struct {
	socket              NetworkSocket
	startRoundQueue     chan constants.Message
	recievedAnswerQueue chan constants.Message
	subscribers         map[constants.MessageType][]chan constants.Message // message type -> array of message channels
}

func (mr *MessageRelayer) Start(ctx context.Context) {
	log.Printf("message relayer starting with %v RecievedAnswer subscribers and %v StartNewRound subsribers", len(mr.subscribers[constants.ReceivedAnswer]), len(mr.subscribers[constants.StartNewRound]))
	for {
		select {
		case msg := <-mr.recievedAnswerQueue:
			subscriberChannels := mr.subscribers[constants.ReceivedAnswer]
			for _, subscriberChannel := range subscriberChannels {
				// log.Println("--> adding new message to subscriber channel for recieved answer message")
				if channelIsFull(subscriberChannel) {
					log.Printf("discarding message of full ReceivedAnswer channel")
					discardChannelMsg(subscriberChannel)
				}
				subscriberChannel <- msg
			}
		case msg := <-mr.startRoundQueue:
			subscriberChannels := mr.subscribers[constants.StartNewRound]
			for _, subscriberChannel := range subscriberChannels {
				// log.Println("--> adding new message to subscriber channel for start new round message")
				if channelIsFull(subscriberChannel) {
					log.Printf("discarding message of full StartNewRound channel")
					discardChannelMsg(subscriberChannel)
				}
				subscriberChannel <- msg
			}
		case <-ctx.Done():
			log.Printf("closed message relayer")
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
		log.Println("added new message to recieved answer queue")
	}
	if msg.Type == constants.StartNewRound || msg.Type == constants.All {
		mr.startRoundQueue <- msg
		log.Println("added new message to start round queue")
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

func channelIsFull(ch chan constants.Message) bool {
	// log.Printf("len=%v, cap=%v", len(ch), cap(ch))
	return len(ch) == cap(ch)
}

func discardChannelMsg(ch chan constants.Message) {
	<-ch
}
