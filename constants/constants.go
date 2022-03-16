package constants

type MessageType int

const (
	StartNewRound MessageType = 1 << iota
	ReceivedAnswer
	All
)

func (mt MessageType) String() string {
	if mt == StartNewRound {
		return "start new round"
	}
	if mt == ReceivedAnswer {
		return "recieved answer"
	}
	return "all"
}

type Message struct {
	Type MessageType
	Data []byte
}
