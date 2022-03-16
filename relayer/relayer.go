package relayer

import (
	"log"
	"messagerelayer/constants"
)

const queueSize = 10

type Relayer interface {
	Start() error
	Read() (Message, error)
	Enqueue(Message)
	SubscribeToMessages(msgType constants.MessageType, ch chan Message)
}

type NetworkSocket interface {
	Read() (Message, error)
}

type Message struct {
	Type constants.MessageType
	Data []byte
}

func NewMessageRelayer(socket NetworkSocket) Relayer {
	return &MessageRelayer{
		socket:              socket,
		startRoundQueue:     make(chan Message, queueSize),
		recievedAnswerQueue: make(chan Message, queueSize),
		subscribers:         make(map[constants.MessageType][]chan Message),
	}
}

type MessageRelayer struct {
	socket              NetworkSocket
	startRoundQueue     chan Message
	recievedAnswerQueue chan Message
	subscribers         map[constants.MessageType][]chan Message // message type -> array of message channels
}

func (mr MessageRelayer) Start() error {
	for {
		select {
		case msg := <-mr.recievedAnswerQueue:
			for _, subscriberChannel := range mr.subscribers[constants.ReceivedAnswer] {
				subscriberChannel <- msg
			}
		case msg := <-mr.startRoundQueue:
			for _, subscriberChannel := range mr.subscribers[constants.StartNewRound] {
				subscriberChannel <- msg
			}
		}
	}
}

func (mr MessageRelayer) Read() (Message, error) {
	return mr.socket.Read()
}

func (mr *MessageRelayer) Enqueue(msg Message) {
	if msg.Type == constants.ReceivedAnswer || msg.Type == constants.All {
		mr.recievedAnswerQueue <- msg
		log.Println("added new message to recieved answer queue")
	}
	if msg.Type == constants.StartNewRound || msg.Type == constants.All {
		mr.startRoundQueue <- msg
		log.Println("added new message to start round queue")
	}
}

func (mr *MessageRelayer) SubscribeToMessages(msgType constants.MessageType, ch chan Message) {
	subscribers := mr.subscribers[msgType]
	mr.subscribers[msgType] = append(subscribers, ch)
}
