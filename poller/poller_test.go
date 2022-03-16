package poller_test

import (
	"context"
	"messagerelayer/constants"
	"messagerelayer/poller"
	"messagerelayer/relayer"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockNetworkSocket struct{}

func (mns *MockNetworkSocket) Read() (constants.Message, error) {
	return constants.Message{
		Type: constants.ReceivedAnswer,
		Data: []byte("a"),
	}, nil
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func TestStartPoller(t *testing.T) {
	msgrelayer := relayer.NewMessageRelayer(&MockNetworkSocket{})
	msgpoller := poller.New(1 * time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	go msgpoller.Start(ctx, msgrelayer)
	go msgrelayer.Start(ctx)
	time.Sleep(5 * time.Second) // artificial wait time to produce messages
	cancel()
	<-msgrelayer.DoneChannel()
	<-msgpoller.DoneChannel()
	summary := msgrelayer.Summary()
	assert.GreaterOrEqual(t, summary.QueuedMsgs, 4, "queued message count")
	assert.Equal(t, 0, summary.BroadcastedMsgs, "broadcasted message count should be 0 since we have no subscribers")
}
