package poller

import (
	"context"
	"log"
	"messagerelayer/relayer"
	"time"
)

// Poller describes something that polls for new messages and sends them to a message relayer
type Poller interface {
	Start(context.Context, relayer.Relayer)
	DoneChannel() chan bool
}

// MessagePoller is a poller that enqueues messages to a message relayer
type MessagePoller struct {
	done         chan bool
	readInterval time.Duration
}

// New returns an instance of a MessagePoller
func New(readInterval time.Duration) Poller {
	return MessagePoller{
		readInterval: readInterval,
		done:         make(chan bool),
	}
}

// Start invokes a message poller to start polling
func (mp MessagePoller) Start(ctx context.Context, msgRelayer relayer.Relayer) {
	ticker := time.NewTicker(mp.readInterval)
	for {
		select {
		case <-ticker.C:
			log.Println("reading new message...")
			msg, err := msgRelayer.Read()
			if err != nil {
				log.Printf("unable to process message: %v", err)
				break
			}
			log.Printf("got new message of type %v: %v", msg.Type, string(msg.Data))
			msgRelayer.Enqueue(msg)
		case <-ctx.Done():
			log.Println("closing poller")
			ticker.Stop()
			mp.done <- true
			return
		}
	}
}

// DoneChannel returns the subscribers done channel so the parent process can wait until it completes to exit
func (mp MessagePoller) DoneChannel() chan bool {
	return mp.done
}
