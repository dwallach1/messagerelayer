# Message Relayer

## To run
```
go build .
./messagerelayer
```

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
