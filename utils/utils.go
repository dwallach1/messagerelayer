package utils

import "messagerelayer/constants"

// ChannelIsFull indicates if the provided channel is full
func ChannelIsFull(ch chan constants.Message) bool {
	return len(ch) == cap(ch)
}

// DiscardChannelMsg takes the oldest message off the channel and does nothing with it
func DiscardChannelMsg(ch chan constants.Message) {
	<-ch
}
