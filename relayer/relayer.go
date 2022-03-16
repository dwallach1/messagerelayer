package relayer

import (
	"context"
	"fmt"
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
	log.Printf("message relayer starting with %v RecievedAnswer subscribers and %v StartNewRound subsribers\v", len(mr.subscribers[constants.ReceivedAnswer]), len(mr.subscribers[constants.StartNewRound]))
	for {
		select {
		case msg := <-mr.recievedAnswerQueue:
			subscriberChannels := mr.subscribers[constants.ReceivedAnswer]
			// log.Printf("funneling recieved answer message to %v subsribers\n", len(subscriberChannels))
			for _, subscriberChannel := range subscriberChannels {
				fmt.Println("--> adding new message to subsriber channel for recieved answer message")
				subscriberChannel <- msg
			}
		case msg := <-mr.startRoundQueue:
			subscriberChannels := mr.subscribers[constants.StartNewRound]
			// log.Printf("funneling start new round message to %v subsribers\n", len(subscriberChannels))
			for _, subscriberChannel := range subscriberChannels {
				fmt.Println("--> adding new message to subsriber channel for start new round message")
				subscriberChannel <- msg
			}
		case <-ctx.Done():
			log.Printf("context cancelled detected: closing message relayer")
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
	mr.subscribers[msgType] = append(mr.subscribers[msgType], ch)
}
