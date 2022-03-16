package poller

import (
	"context"
	"log"
	"messagerelayer/relayer"
	"time"
)

type Poller interface {
	Start(context.Context, relayer.Relayer)
	DoneChannel() chan bool
}

type MessagePoller struct {
	done         chan bool
	readInterval time.Duration
}

func New(readInterval time.Duration) Poller {
	return MessagePoller{
		readInterval: readInterval,
		done:         make(chan bool),
	}
}

func (mp MessagePoller) Start(ctx context.Context, msgRelayer relayer.Relayer) {
	ticker := time.NewTicker(mp.readInterval)
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
		case <-ctx.Done():
			log.Println("closing poller")
			ticker.Stop()
			mp.done <- true
			return
		}
	}
}

func (mp MessagePoller) DoneChannel() chan bool {
	return mp.done
}
