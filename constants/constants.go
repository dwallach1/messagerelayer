package constants

type MessageType int

const (
	StartNewRound MessageType = 1 << iota
	ReceivedAnswer
	All
)

type Message struct {
	Type MessageType
	Data []byte
}
