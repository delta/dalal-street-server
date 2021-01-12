package datastreams

import (
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/delta/dalal-street-server/utils"
)

// BroadcastStream represents an object that provides methods for handling a single stream
// All updates are broadcast to *all* listeners.
type BroadcastStream interface {
	AddListener(sessionId string, lis *listener)
	RemoveListener(sessionId string)
	BroadcastUpdate(update interface{})
	GetListenersCount() int
}

// listenersMap combines a RWMutex with a map of listeners
type listenersMap struct {
	sync.RWMutex
	m map[string]*listener
}

// broadcastStream implements BroadcastStream interface
type broadcastStream struct {
	logger    *logrus.Entry
	listeners *listenersMap
}

// NewBroadcastStream creates a BroadcastStream
func NewBroadcastStream() BroadcastStream {
	return &broadcastStream{
		logger: utils.Logger.WithFields(logrus.Fields{
			"module": "datastreams.BroadcastStream",
		}),
		listeners: &listenersMap{
			m: make(map[string]*listener),
		},
	}
}

// AddListener adds a listener to a stream
func (bs *broadcastStream) AddListener(sessionId string, lis *listener) {
	l := bs.logger.WithFields(logrus.Fields{
		"method":          "AddListener",
		"param_sessionId": sessionId,
	})

	bs.listeners.Lock()
	bs.listeners.m[sessionId] = lis
	bs.listeners.Unlock()

	l.Debugf("Added")
}

// RemoveListener removes a listener from a stream
func (bs *broadcastStream) RemoveListener(sessionId string) {
	l := bs.logger.WithFields(logrus.Fields{
		"method":          "RemoveListener",
		"param_sessionId": sessionId,
	})

	bs.listeners.Lock()
	delete(bs.listeners.m, sessionId)
	bs.listeners.Unlock()

	l.Debugf("Removed")
}

// BroadcastUpdate broadcasts a given update to all listeners in the stream
func (bs *broadcastStream) BroadcastUpdate(update interface{}) {
	l := bs.logger.WithFields(logrus.Fields{
		"method": "BroadcastUpdate",
	})

	var deadListenerIds []string

	bs.listeners.RLock()

	l.Debugf("Broadcasting to %d listeners", len(bs.listeners.m))
	for sessionId, listener := range bs.listeners.m {
		select {
		case <-listener.done:
			deadListenerIds = append(deadListenerIds, sessionId)
		case listener.update <- update:
		}
	}
	bs.listeners.RUnlock()

	l.Debugf("Found %d dead listeners", len(deadListenerIds))
	if len(deadListenerIds) == 0 {
		return
	}

	bs.listeners.Lock()
	for _, sessionId := range deadListenerIds {
		delete(bs.listeners.m, sessionId)
	}
	bs.listeners.Unlock()

	l.Debugf("Deleted dead listeners")
}

func (bs *broadcastStream) GetListenersCount() int {
	bs.listeners.RLock()
	count := len(bs.listeners.m)
	bs.listeners.RUnlock()

	return count
}
