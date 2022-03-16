package main

import (
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
	Messages                 []relayer.Message
	DelaySecsBetweenMessages func(int) time.Duration // take in the number of messages and return a delay
	ProcessedMsgs            *int
}

var i = 0

func (mns MockNetworkSocket) Read() (relayer.Message, error) {
	i++
	return relayer.Message{
		Type: constants.StartNewRound,
		Data: []byte(fmt.Sprintf("mock message %v", i)),
	}, nil
}

func main() {
	/*
	 * Service configuration and setup
	 */
	msgRelayer := relayer.NewMessageRelayer(MockNetworkSocket{})
	stopPollingChan := make(chan bool)
	go handleSigInt(stopPollingChan)
	log.Printf("starting message relayer with read interval set to %v seconds...", READ_INTERVAL_SECS)

	/*
	 * Setup mock subscribers
	 */
	subscribers := []subscriber.Subscriber{
		subscriber.New(constants.StartNewRound, func() time.Duration { return time.Second * 1 }, 5, "joe"),
		subscriber.New(constants.StartNewRound, func() time.Duration { return time.Second * 1 }, 5, "bob"),
		subscriber.New(constants.All, func() time.Duration { return time.Second * 1 }, 5, "sally"),
	}
	for _, s := range subscribers {
		msgRelayer.SubscribeToMessages(s.Type(), s.Channel(s.Type()))
		go s.Start()
	}

	go msgRelayer.Start()

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

func handleSigInt(stopPollingChan chan bool) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Println("detected signal interrupt, cleaning up and exiting..")
	stopPollingChan <- true
	time.Sleep(1 * time.Second)
	// @todo do we need to do this
	close(stopPollingChan)
	log.Println("exiting gracefully")
	os.Exit(0)
}
