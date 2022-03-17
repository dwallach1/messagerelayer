# Message Relayer

## Overview
```
go build . && ./messagerelayer
```
To stop it, ctrl+c and it will close out and exit gracefully. Additionally, I added some mock subscribers in the main function to 
show the functionality.

The main function instatiates:
* a root context to handle closing out subscribers, pollers and relayers
* a message relayer
* a message poller
* mock subscribers

It then adds each subscriber to the declared relayer and starts each subscriber, the relayer and the poller.
* To address the priority of new messages, I used a doubly linked list so we could always pop off the head (newest message). We first check
the "StartNewRound" message queue to ensure it takes priority for each `relayer.BroadcastInterval` check
* We use a doubly linked list to avoid local memory consumuption runaway. If we detect the size of the queues are greater than `relayer.QueueSize`, we then resize the list and drop the tails until we are within the desired size range.

## Improvements
To handle addtional load, we could introduce multiplicity across relayers and pollers. We could achieve this in different ways:
1. give the poller a pool of relayers where each relayer has the same copy of the list of subscribers. The poller then adds the incoming 
message to a round-robin selected relayer. I did an example of this: https://github.com/dwallach1/messagerelayer/pull/2
2. use a "manager" concept that holds the queues that multiple relayers can read from and each relayer pops a message when it becomes available to do so
3. Use a DB to store the pruned messages when we resize the array, then read and remove them from the DB during low load and emit the messages when resources permit (this assuming that the delivery time is not a hard requirement and subscribers still want older messages).

## Objective

Build a system that broadcasts messages from a network endpoint to other systems in the codebase.
There are two types of messages, “StartNewRound” and “ReceivedAnswer”
We have to limit the number of messages that we keep queued locally to avoid runaway memory consumption, but the protocol imposes some rules about the priority of these messages:

    - We must always ensure that we broadcast the 2 most recent StartNewRound messages
    - We only need to ensure that we broadcast only the most recent ReceivedAnswer message
    - Any time that both a “StartNewRound” and a “ReceivedAnswer” are queued, the “StartNewRound” message should be broadcasted first
    - Any time that one of the subscribers of this system is busy and cannot receive a message immediately, we should just skip broadcasting to that subscriber

Don’t worry about writing actual networking code.  Instead, the MessageRelayer should simply read messages from an interface given to its constructor:

```go
func NewMessageRelayer(socket NetworkSocket) {
    // ...
}

type NetworkSocket interface {
    Read() (Message, error)
}

type Message struct {
    Type MessageType
    Data []byte
}
```



The MessageRelayer system should expose the following interface to other systems in the codebase:

```go
type MessageRelayer interface {
    SubscribeToMessages(msgType MessageType, ch chan Message)
}


type MessageType int

const (
    StartNewRound   MessageType = 1 << iota
    ReceivedAnswer
)
```

As you can see, SubscribeToMessages takes a “msgType” parameter, which allows other systems in the codebase to specify if they want to receive only one of the message types or both.

Don’t forget to write some tests to ensure that your solution works correctly.  We recommend the following Go libraries:
https://github.com/stretchr/testify/require
https://github.com/stretchr/testify/mock
However, if you prefer to use other libraries, that’s fine as well.
