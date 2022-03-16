package utils

import "messagerelayer/constants"

// ChannelIsFull indicates if the provided channel is full
func ChannelIsFull(ch chan constants.Message) bool {
	return len(ch) == cap(ch)
}
