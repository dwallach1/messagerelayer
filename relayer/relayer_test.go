package relayer_test

import (
	"context"
	"messagerelayer/constants"
	"messagerelayer/relayer"
	"messagerelayer/subscriber"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockNetworkSocket struct {
	Messages      []constants.Message
	ReadCallCount int
}

func (mns *MockNetworkSocket) Read() (constants.Message, error) {
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
	assert.Equal(t, 8, msgrelayer.QueuedMsgs(), "queued message count")
	assert.Equal(t, 8, msgrelayer.BroadcastedMsgs(), "broadcasted message count")
}
