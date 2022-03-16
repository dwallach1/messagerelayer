package main

import (
	"context"
	"fmt"
	"log"
	"messagerelayer/constants"
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
	stopPollingChan := make(chan bool)
	go handleSigInt(stopPollingChan, cancel, msgRelayer)
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
	go pollMsgs(rootCtx, msgRelayer, stopPollingChan)
	select {}
}

func pollMsgs(ctx context.Context, msgRelayer relayer.Relayer, stop chan bool) {
	ticker := time.NewTicker(READ_INTERVAL_SECS * time.Second)
	for {
		select {
		case <-ticker.C:
			log.Println("reading new message...")
			msg, err := msgRelayer.Read()
			if err != nil {
				log.Printf("unable to process message: %v\n", err)
				break
			}
			log.Printf("got new message of type %v: %v\n", msg.Type, string(msg.Data))
			msgRelayer.Enqueue(msg)
		case <-stop:
			ticker.Stop()
			log.Println("closed poller")
			return
		case <-ctx.Done():
			ticker.Stop()
			log.Println("closed poller")
			return
		}
	}
}

func handleSigInt(stopPollingChan chan bool, cancel context.CancelFunc, msgRelayer relayer.Relayer) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Println("detected signal interrupt, cleaning up and exiting..")
	stopPollingChan <- true
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
	log.Println("exiting gracefully")
	os.Exit(0)
}
