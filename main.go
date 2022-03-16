package main

import (
	"context"
	"fmt"
	"log"
	"messagerelayer/constants"
	"messagerelayer/poller"
	"messagerelayer/relayer"
	"messagerelayer/subscriber"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const READ_INTERVAL_SECS = 5

type MockNetworkSocket struct {
	Messages                 []constants.Message
	DelaySecsBetweenMessages func(int) time.Duration // take in the number of messages and return a delay
	ProcessedMsgs            *int
}

var i = 0

func (mns MockNetworkSocket) Read() (constants.Message, error) {
	i++
	return constants.Message{
		Type: []constants.MessageType{constants.StartNewRound, constants.ReceivedAnswer}[i%2],
		Data: []byte(fmt.Sprintf("mock message %v", i)),
	}, nil
}

/*
* Setup mock subscribers
 */
var subscribers = []subscriber.Subscriber{
	subscriber.New(
		constants.ReceivedAnswer,
		func() time.Duration { return time.Second * 3 },
		5,
		"joe",
	),
	subscriber.New(
		constants.StartNewRound,
		func() time.Duration { return time.Second * 3 },
		5,
		"bob",
	),
	subscriber.New(
		constants.All,
		func() time.Duration { return time.Second * 3 },
		5,
		"sally",
	),
}

func main() {
	/*
	 * Service configuration and setup
	 */
	rootCtx, cancel := context.WithCancel(context.Background())
	msgRelayer := relayer.NewMessageRelayer(MockNetworkSocket{})
	msgPoller := poller.New(READ_INTERVAL_SECS * time.Second)
	go handleSigInt(cancel, msgRelayer, msgPoller)
	log.Printf("starting service with read interval set to %v seconds...", READ_INTERVAL_SECS)
	for _, s := range subscribers {
		subscriberType := s.Type()
		if subscriberType == constants.StartNewRound || subscriberType == constants.All {
			subscriberChan := s.Channel(constants.StartNewRound)
			msgRelayer.SubscribeToMessages(constants.StartNewRound, subscriberChan)
		}
		if subscriberType == constants.ReceivedAnswer || subscriberType == constants.All {
			subscriberChan := s.Channel(constants.ReceivedAnswer)
			msgRelayer.SubscribeToMessages(constants.ReceivedAnswer, subscriberChan)
		}
		go s.Start(rootCtx)
	}

	log.Println("starting message relayer...")
	go msgRelayer.Start(rootCtx)
	go msgPoller.Start(rootCtx, msgRelayer)
	select {}
}

func handleSigInt(cancel context.CancelFunc, msgRelayer relayer.Relayer, msgPoller poller.Poller) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Println("detected signal interrupt, cleaning up and exiting..")
	cancel()
	// wait for all subscribers to close gracefully
	for _, s := range subscribers {
		<-s.DoneChannel()
		close(s.DoneChannel())
		log.Printf("subscriber %v is now closed", s.Name())
	}
	// wait for message relayer to close gracefully
	<-msgRelayer.DoneChannel()
	log.Printf("message relayer is now closed")
	close(msgRelayer.DoneChannel())
	// wait for poller to close gracefully
	<-msgPoller.DoneChannel()
	log.Printf("poller is now closed")
	close(msgPoller.DoneChannel())
	log.Println("exiting gracefully")
	os.Exit(0)
}
