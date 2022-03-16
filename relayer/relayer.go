package relayer

import (
	"context"
	"log"
	"messagerelayer/constants"
	"messagerelayer/utils"
	"sync"
	"time"
)

// QueueSize is the desired retention of messages to keep locally
var QueueSize = 50

// BroadcastInterval is the amount of wait time to broacast incoming queued messages
var BroadcastInterval = 1 * time.Second

// WorkSummary returns a summary of the work a relayer has completed
type WorkSummary struct {
	QueuedMsgs      int // successfully added messages to queue to be broadcasted
	BroadcastedMsgs int // successfully broadcasted to a subscriber
	DiscardedMsgs   int // queus full so we discarded older messages
	SkippedMsgs     int // subscriber busy so we dropped the message
}

// Relayer relays messages to subscribers
type Relayer interface {
	Start(context.Context)
	Read() (constants.Message, error)
	Enqueue(constants.Message)
	SubscribeToMessages(msgType constants.MessageType, ch chan constants.Message)
	DoneChannel() chan bool
	// helpers for test validation
	Summary() WorkSummary
}

// NetworkSocket reads incoming messages
type NetworkSocket interface {
	Read() (constants.Message, error)
}

type LinkedMsgList struct {
	head        *MsgNode
	tail        *MsgNode
	size        int
	desiredSize int
	mu          sync.Mutex
}

type MsgNode struct {
	msg  *constants.Message
	next *MsgNode
	prev *MsgNode
}

func NewLinkedMsgList(desiredSize int) *LinkedMsgList {
	return &LinkedMsgList{
		head:        nil,
		tail:        nil,
		size:        0,
		desiredSize: desiredSize,
		mu:          sync.Mutex{},
	}
}

func (lml *LinkedMsgList) Push(msg constants.Message) {
	lml.mu.Lock()
	lml.size++
	// no head set, set and return
	if lml.head == nil {
		lml.head = &MsgNode{
			msg:  &msg,
			next: nil,
			prev: nil,
		}
		lml.mu.Unlock()
		return
	}
	// replace head with incoming msg node, have it point its next to the current msg node
	currHead := lml.head
	newHead := &MsgNode{
		msg:  &msg,
		next: currHead,
		prev: nil,
	}
	if lml.tail == nil {
		lml.tail = currHead
	}
	currHead.prev = newHead
	lml.head = newHead
	lml.mu.Unlock()
}

func (lml *LinkedMsgList) Pop() *constants.Message {
	lml.mu.Lock()
	if lml.head == nil {
		lml.mu.Unlock()
		return nil
	}
	curr := lml.head
	msg := curr.msg
	lml.head = curr.next // fast forward pointer
	lml.size--
	lml.mu.Unlock()
	return msg
}

func (lml *LinkedMsgList) Resize() int {
	lml.mu.Lock()
	dropped := 0
	for lml.size >= lml.desiredSize && lml.size > 1 {
		secondToLast := lml.tail.prev
		secondToLast.next = nil
		lml.tail = secondToLast
		lml.size--
		dropped++
	}
	lml.mu.Unlock()
	return dropped
}

func (lml *LinkedMsgList) Size() int {
	lml.mu.Lock()
	size := lml.size
	lml.mu.Unlock()
	return size
}

// NewMessageRelayer returns a new message relayer
func NewMessageRelayer(socket NetworkSocket) Relayer {
	return &MessageRelayer{
		socket:               socket,
		startRoundQueue:      NewLinkedMsgList(QueueSize),
		recievedAnswerQueue:  NewLinkedMsgList(QueueSize),
		subscribers:          make(map[constants.MessageType][]chan constants.Message),
		queuesMsgsCount:      0,
		broadcastedMsgsCount: 0,
		discardedMsgsCount:   0,
		skippedMsgCount:      0,
		done:                 make(chan bool),
	}
}

// MessageRelayer relays messages from a network socket to its subscribers
type MessageRelayer struct {
	socket               NetworkSocket
	startRoundQueue      *LinkedMsgList
	recievedAnswerQueue  *LinkedMsgList
	subscribers          map[constants.MessageType][]chan constants.Message // message type -> array of message channels
	queuesMsgsCount      int
	broadcastedMsgsCount int
	discardedMsgsCount   int
	skippedMsgCount      int
	done                 chan bool
}

func (mr *MessageRelayer) Start(ctx context.Context) {
	// log.Printf("-----message relayer starting with %v RecievedAnswer subscribers and %v StartNewRound subsribers----", len(mr.subscribers[constants.ReceivedAnswer]), len(mr.subscribers[constants.StartNewRound]))
	for {
		select {
		case <-ctx.Done():
			log.Printf("closing message relayer:: queued: %v messages, broadcasted: %v messages", mr.queuesMsgsCount, mr.broadcastedMsgsCount)
			mr.done <- true
			return
		default:
			/*
			 * start round queue takes precedent over the recieved answer queue
			 * so we do a nonblocking check for this queue first and then funnel into checking both
			 * if no messages are queued, we will funnel to bottom most default where we sleep for the Broadcast interval
			 */
			startNewRoundMsg := mr.startRoundQueue.Pop()
			if startNewRoundMsg != nil {

				mr.broacast(*startNewRoundMsg)
			}
			recievedAnsMsg := mr.recievedAnswerQueue.Pop()
			if recievedAnsMsg != nil {
				mr.broacast(*recievedAnsMsg)
			}
			mr.skippedMsgCount += mr.recievedAnswerQueue.Resize()
			mr.startRoundQueue.Resize()
			time.Sleep(BroadcastInterval)
		}
	}
}

func (mr *MessageRelayer) broacast(msg constants.Message) {
	var msgType constants.MessageType
	if msg.Type == constants.StartNewRound || msg.Type == constants.All {
		msgType = constants.StartNewRound
	}
	if msg.Type == constants.ReceivedAnswer || msg.Type == constants.All {
		msgType = constants.ReceivedAnswer
	}
	subscriberChannels := mr.subscribers[msgType]
	log.Printf("🔊  broadcasting %v message", msgType.String())
	for _, subscriberChannel := range subscriberChannels {
		if utils.ChannelIsFull(subscriberChannel) {
			mr.skippedMsgCount++
			log.Printf("subscriber busy: detected full StartNewRound subscriber channel: skipping broadcast")
			continue
		}
		mr.broadcastedMsgsCount++
		subscriberChannel <- msg
	}
}

// Read calls the underlying network socket's read method
func (mr MessageRelayer) Read() (constants.Message, error) {
	return mr.socket.Read()
}

// Enqueue takes an incoming message and adds it to the message relayer's broadcasting queues
func (mr *MessageRelayer) Enqueue(msg constants.Message) {
	if msg.Type == constants.ReceivedAnswer || msg.Type == constants.All {
		mr.recievedAnswerQueue.Push(msg)
		mr.queuesMsgsCount++
		log.Println("⤴️  added new message to recieved answer queue")
	}
	if msg.Type == constants.StartNewRound || msg.Type == constants.All {
		mr.startRoundQueue.Push(msg)
		mr.queuesMsgsCount++
		log.Println("⤴️  added new message to start round queue")
	}
}

// SubscribeToMessages registers a new subscriber to a message relayers broadcasting queues
func (mr *MessageRelayer) SubscribeToMessages(msgType constants.MessageType, ch chan constants.Message) {
	if msgType == constants.All {
		mr.subscribers[constants.ReceivedAnswer] = append(mr.subscribers[constants.ReceivedAnswer], ch)
		mr.subscribers[constants.StartNewRound] = append(mr.subscribers[constants.StartNewRound], ch)
		return
	}
	mr.subscribers[msgType] = append(mr.subscribers[msgType], ch)
}

// DoneChannel returns the message relayers done channel for the parent process to wait for it to complete
// before closing
func (mr MessageRelayer) DoneChannel() chan bool {
	return mr.done
}

// Summary returns the WorkSummary of the message relayer
func (mr MessageRelayer) Summary() WorkSummary {
	return WorkSummary{
		QueuedMsgs:      mr.queuesMsgsCount,
		BroadcastedMsgs: mr.broadcastedMsgsCount,
		DiscardedMsgs:   mr.discardedMsgsCount,
		SkippedMsgs:     mr.skippedMsgCount,
	}
}
