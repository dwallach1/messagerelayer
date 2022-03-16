package utils_test

import (
	"messagerelayer/constants"
	"messagerelayer/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShrinkChannel(t *testing.T) {
	testChan := make(chan constants.Message, 10)
	for i := 0; i < 5; i++ {
		testChan <- constants.Message{}
	}
	utils.ShrinkChannel(testChan, 2)
	assert.Equal(t, 2, len(testChan), "testChan size after shrinking")
}

func TestShrinkChannelNoChange(t *testing.T) {
	testChan := make(chan constants.Message, 10)
	for i := 0; i < 5; i++ {
		testChan <- constants.Message{}
	}
	utils.ShrinkChannel(testChan, 5)
	assert.Equal(t, 5, len(testChan), "testChan size after shrinking remains the same")
}
