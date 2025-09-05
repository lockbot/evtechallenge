package api

import (
	"fmt"
)

// ResponseChannel represents a response channel with metadata
type ResponseChannel struct {
	ch    chan ResponseMessage
	key   string
	inUse bool
}

// ResponsePool manages a pool of response channels
type ResponsePool struct {
	responseChannels map[string]*ResponseChannel
	responsePool     chan *ResponseChannel
}

// NewResponsePool creates a new response channel pool
func NewResponsePool(size int) *ResponsePool {
	responsePool := make(chan *ResponseChannel, size)
	responseChannels := make(map[string]*ResponseChannel)

	// Pre-create response channels
	for i := 0; i < size; i++ {
		key := fmt.Sprintf("resp_%d", i)
		respCh := &ResponseChannel{
			ch:    make(chan ResponseMessage, 1),
			key:   key,
			inUse: false,
		}
		responseChannels[key] = respCh
		responsePool <- respCh
	}

	return &ResponsePool{
		responseChannels: responseChannels,
		responsePool:     responsePool,
	}
}

// GetChannel gets a response channel from the pool
func (rp *ResponsePool) GetChannel() *ResponseChannel {
	select {
	case respCh := <-rp.responsePool:
		respCh.inUse = true
		return respCh
	default:
		// Pool exhausted, create a temporary one
		return &ResponseChannel{
			ch:    make(chan ResponseMessage, 1),
			key:   "temp",
			inUse: true,
		}
	}
}

// ReturnChannel returns a response channel to the pool
func (rp *ResponsePool) ReturnChannel(respCh *ResponseChannel) {
	if respCh.key == "temp" {
		// Don't return temporary channels to pool
		return
	}
	respCh.inUse = false
	select {
	case rp.responsePool <- respCh:
	default:
		// Pool is full, discard
	}
}

// GetChannelByKey gets a specific response channel by key
func (rp *ResponsePool) GetChannelByKey(key string) (*ResponseChannel, bool) {
	respCh, exists := rp.responseChannels[key]
	return respCh, exists
}
