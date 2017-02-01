package datastreams

import (
	"github.com/thakkarparth007/dalal-street-server/utils"
)

type listener struct {
	update chan interface{}
	done   <-chan struct{}
}

type Stream interface {
	stream
}
