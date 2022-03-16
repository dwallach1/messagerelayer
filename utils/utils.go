package utils

import "messagerelayer/constants"

func ChannelIsFull(ch chan constants.Message) bool {
	return len(ch) == cap(ch)
}

func DiscardChannelMsg(ch chan constants.Message) {
	<-ch
}
