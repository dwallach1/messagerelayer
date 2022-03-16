package subscriber_test

import (
	"context"
	"fmt"
	"messagerelayer/constants"
	"messagerelayer/subscriber"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func TestNewSubscriber(t *testing.T) {
	queueSize := 5
	waitTime := 3 * time.Second
	tests := []struct {
		name                              string
		expectedMsgType                   constants.MessageType
		expectedCapForStartRoundQueue     int
		expectedCapForRecievedAnswerQueue int
		expectedWaitTime                  time.Duration
	}{
		{
			name:                              "Message Type Recieved Answer",
			expectedMsgType:                   constants.ReceivedAnswer,
			expectedCapForStartRoundQueue:     0,
			expectedCapForRecievedAnswerQueue: queueSize,
			expectedWaitTime:                  waitTime,
		},
		{
			name:                              "Message Type StartNewRound",
			expectedMsgType:                   constants.StartNewRound,
			expectedCapForStartRoundQueue:     queueSize,
			expectedCapForRecievedAnswerQueue: 0,
			expectedWaitTime:                  waitTime,
		},
		{
			name:                              "Message Type All",
			expectedMsgType:                   constants.All,
			expectedCapForStartRoundQueue:     queueSize,
			expectedCapForRecievedAnswerQueue: queueSize,
			expectedWaitTime:                  waitTime,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := subscriber.New(
				tt.expectedMsgType,
				func() time.Duration { return waitTime },
				queueSize,
				tt.name,
			)
			assert.Equal(t, waitTime, s.WaitTime(), "wait time set correctly")
			assert.Equal(t, tt.name, s.Name(), "name set correctly")
			assert.Equal(t, tt.expectedMsgType, s.Type(), "message type correctly")
			assert.Equal(t, tt.expectedCapForRecievedAnswerQueue, cap(s.Channel(constants.ReceivedAnswer)), "recieved answer queue")
			assert.Equal(t, tt.expectedCapForStartRoundQueue, cap(s.Channel(constants.StartNewRound)), "start new round queue")
		})
	}
}

func TestStartSubscriber(t *testing.T) {
	s := subscriber.New(
		constants.All,
		func() time.Duration { return 0 * time.Second },
		2,
		"mock subscriber",
	)
	ctx, cancel := context.WithCancel(context.Background())
	go s.Start(ctx)
	for i := 0; i < 3; i++ {
		s.Channel(constants.ReceivedAnswer) <- constants.Message{
			Type: constants.ReceivedAnswer,
			Data: []byte(fmt.Sprintf("%v", i)),
		}
		s.Channel(constants.StartNewRound) <- constants.Message{
			Type: constants.StartNewRound,
			Data: []byte(fmt.Sprintf("%v", i)),
		}
	}
	cancel()
	<-s.DoneChannel()
	assert.Equal(t, 6, s.ProcessedCount(), "processed count")
}
