package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
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
	rand.Seed(time.Now().UnixNano())
	return constants.Message{
		Type: []constants.MessageType{constants.StartNewRound, constants.ReceivedAnswer}[rand.Intn(1)],
		Data: []byte(fmt.Sprintf("mock message %v", i)),
	}, nil
}

func main() {
	/*
	 * Service configuration and setup
	 */
	rootCtx, cancel := context.WithCancel(context.Background())
	msgRelayer := relayer.NewMessageRelayer(MockNetworkSocket{})
	stopPollingChan := make(chan bool)
	go handleSigInt(stopPollingChan, cancel)
	log.Printf("starting service with read interval set to %v seconds...", READ_INTERVAL_SECS)

	/*
	 * Setup mock subscribers
	 */
	subscribers := []subscriber.Subscriber{
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

	/*
	 * Poller logic to process incoming messages
	 */
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
		case <-stopPollingChan:
			ticker.Stop()
			log.Println("stopped poller...")
		}
	}
}

func handleSigInt(stopPollingChan chan bool, cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Println("detected signal interrupt, cleaning up and exiting..")
	stopPollingChan <- true
	cancel()
	time.Sleep(5 * time.Second) // let things shut down gracefully
	// @todo do we need to do this
	log.Println("exiting gracefully")
	os.Exit(0)
}
