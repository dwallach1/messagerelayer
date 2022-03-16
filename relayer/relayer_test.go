package relayer_test

import (
	"context"
	"fmt"
	"messagerelayer/constants"
	"messagerelayer/relayer"
	"messagerelayer/subscriber"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockNetworkSocket struct {
	DefaultMessage constants.Message
	Messages       []constants.Message
	ReadCallCount  int
}

func (mns *MockNetworkSocket) Read() (constants.Message, error) {
	if mns.ReadCallCount >= len(mns.Messages) {
		return mns.DefaultMessage, nil
	}
	m := mns.Messages[mns.ReadCallCount]
	mns.ReadCallCount++
	return m, nil
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

var mockMsgList = []constants.Message{
	{
		Type: constants.ReceivedAnswer,
		Data: []byte("a"),
	},
	{
		Type: constants.StartNewRound,
		Data: []byte("b"),
	},
	{
		Type: constants.ReceivedAnswer,
		Data: []byte("c"),
	},
	{
		Type: constants.StartNewRound,
		Data: []byte("d"),
	},
	{
		Type: constants.ReceivedAnswer,
		Data: []byte("e"),
	},
	{
		Type: constants.StartNewRound,
		Data: []byte("f"),
	},
	{
		Type: constants.ReceivedAnswer,
		Data: []byte("g"),
	},
	{
		Type: constants.ReceivedAnswer,
		Data: []byte("h"),
	},
}

func TestStartMessageRelayer(t *testing.T) {
	s := subscriber.New(
		constants.All,
		func() time.Duration { return 0 * time.Second },
		10,
		"mock subscriber",
	)
	msgrelayer := relayer.NewMessageRelayer(&MockNetworkSocket{
		ReadCallCount: 0,
		Messages:      mockMsgList,
	})
	msgrelayer.SubscribeToMessages(constants.StartNewRound, s.Channel(constants.StartNewRound))
	msgrelayer.SubscribeToMessages(constants.ReceivedAnswer, s.Channel(constants.ReceivedAnswer))
	relayer.BroadcastInterval = 0 * time.Second
	ctx, cancel := context.WithCancel(context.Background())
	go s.Start(ctx)
	go msgrelayer.Start(ctx)
	for i := 0; i < len(mockMsgList); i++ {
		m, err := msgrelayer.Read()
		assert.Nil(t, err, "read err is nil")
		msgrelayer.Enqueue(m)
	}
	time.Sleep(2 * time.Second) // artificial wait time to allow messages to get processed
	cancel()
	<-s.DoneChannel()
	<-msgrelayer.DoneChannel()
	summary := msgrelayer.Summary()
	assert.Equal(t, len(mockMsgList), summary.QueuedMsgs, "queued message count")
	assert.Equal(t, len(mockMsgList), summary.BroadcastedMsgs, "broadcasted message count")
}

func TestStartMessageRelayerWithBusySubscriber(t *testing.T) {
	s := subscriber.NewNoop(2)
	msgrelayer := relayer.NewMessageRelayer(&MockNetworkSocket{
		ReadCallCount: 0,
		DefaultMessage: constants.Message{
			Type: constants.StartNewRound,
			Data: []byte("mock"),
		},
	})
	relayer.BroadcastInterval = 0 * time.Second
	msgrelayer.SubscribeToMessages(constants.StartNewRound, s.Channel(constants.StartNewRound))
	ctx, cancel := context.WithCancel(context.Background())
	go s.Start(ctx)
	go msgrelayer.Start(ctx)
	for i := 0; i < 8; i++ {
		m, err := msgrelayer.Read()
		assert.Nil(t, err, "read err is nil")
		msgrelayer.Enqueue(m)
	}
	time.Sleep(2 * time.Second) // artificial wait time to allow messages to get processed
	cancel()
	<-s.DoneChannel()
	<-msgrelayer.DoneChannel()
	summary := msgrelayer.Summary()
	assert.Equal(t, len(mockMsgList), summary.QueuedMsgs, "queued message count")
	assert.Equal(t, 2, summary.BroadcastedMsgs, "broadcasted message count should be less since subscriber fills up after 2")
	assert.Equal(t, len(mockMsgList)-2, summary.SkippedMsgs, "skipped message count")
}

func TestMessagePriroty(t *testing.T) {
	s := subscriber.NewNoop(2)
	msgrelayer := relayer.NewMessageRelayer(&MockNetworkSocket{
		ReadCallCount: 0,
		DefaultMessage: constants.Message{
			Type: constants.StartNewRound,
			Data: []byte("mock"),
		},
	})
	relayer.BroadcastInterval = 5 * time.Second // allow both messages to get queued
	msgrelayer.SubscribeToMessages(constants.StartNewRound, s.Channel(constants.StartNewRound))
	msgrelayer.SubscribeToMessages(constants.ReceivedAnswer, s.Channel(constants.ReceivedAnswer))
	ctx, cancel := context.WithCancel(context.Background())
	go s.Start(ctx)
	/*
	 * enqueue both message types
	 */
	msgrelayer.Enqueue(constants.Message{Type: constants.ReceivedAnswer, Data: []byte("recievedAns")})
	msgrelayer.Enqueue(constants.Message{Type: constants.StartNewRound, Data: []byte("startNewRound")})
	go msgrelayer.Start(ctx)
	resultChan := make(chan constants.Message)
	// read only a single message off the queues
	go func(result chan constants.Message, startNewRoundChan chan constants.Message, recievedAnsChan chan constants.Message) {
		select {
		case msg := <-recievedAnsChan:
			result <- msg
			return
		case msg := <-startNewRoundChan:
			result <- msg
			return
		}
	}(resultChan, s.Channel(constants.StartNewRound), s.Channel(constants.ReceivedAnswer))
	msg := <-resultChan
	assert.Equal(t, "startNewRound", string(msg.Data)) // ensure the message we read is a startNewRound message as this has higher priority
	cancel()
	<-s.DoneChannel()
	<-msgrelayer.DoneChannel()
}

func TestMostRecentMessagesBroadcastedFirst(t *testing.T) {
	s := subscriber.NewNoop(10)
	msgrelayer := relayer.NewMessageRelayer(&MockNetworkSocket{
		ReadCallCount: 0,
		DefaultMessage: constants.Message{
			Type: constants.ReceivedAnswer,
			Data: []byte("mock"),
		},
	})
	relayer.BroadcastInterval = 0 * time.Second
	msgrelayer.SubscribeToMessages(constants.ReceivedAnswer, s.Channel(constants.ReceivedAnswer))
	ctx, cancel := context.WithCancel(context.Background())
	go s.Start(ctx)
	msgrelayer.Enqueue(constants.Message{Type: constants.ReceivedAnswer, Data: []byte("recievedAns1")})
	msgrelayer.Enqueue(constants.Message{Type: constants.ReceivedAnswer, Data: []byte("recievedAns2")})
	msgrelayer.Enqueue(constants.Message{Type: constants.ReceivedAnswer, Data: []byte("recievedAns3")})
	msgrelayer.Enqueue(constants.Message{Type: constants.ReceivedAnswer, Data: []byte("recievedAns4")})
	msgrelayer.Enqueue(constants.Message{Type: constants.ReceivedAnswer, Data: []byte("recievedAns5")})

	go msgrelayer.Start(ctx)

	resultChan := make(chan constants.Message)
	// read only a single message off the queues
	go func(result chan constants.Message, recievedAnsChan chan constants.Message) {
		msg := <-recievedAnsChan
		result <- msg
	}(resultChan, s.Channel(constants.ReceivedAnswer))
	msg := <-resultChan
	assert.Equal(t, "recievedAns5", string(msg.Data), "we should read most recent")
	cancel()
	<-s.DoneChannel()
	<-msgrelayer.DoneChannel()
}

func TestLinkedListResizing(t *testing.T) {
	list := relayer.NewLinkedMsgList(5)

	for i := 0; i < 10; i++ {
		list.Push(constants.Message{Data: []byte(fmt.Sprintf("msg_%v", i))})
	}
	assert.Equal(t, 10, list.Size(), "list should have all messages")
	list.Resize()
	assert.Equal(t, 4, list.Size(), "list should have all messages")
	// now ensure that we have the latest ones
	assert.Equal(t, string(list.Pop().Data), "msg_9")
	assert.Equal(t, string(list.Pop().Data), "msg_8")
	assert.Equal(t, string(list.Pop().Data), "msg_7")
	assert.Equal(t, string(list.Pop().Data), "msg_6")
	// once we pop all of them, verify we get null
	assert.Nil(t, list.Pop())
	assert.Equal(t, 0, list.Size(), "list should now be empty")
}
