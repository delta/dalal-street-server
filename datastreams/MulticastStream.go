package datastreams

import (
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

// MulticastStream represents an object that provides methods for handling multiple groups in
// a stream. It allows you to broadcast updates to individual groups and manage the groups.
type MulticastStream interface {
	AddListener(groupId uint32, sessionId string, lis *listener)
	RemoveListener(groupId uint32, sessionId string)
	BroadcastUpdateToGroup(groupId uint32, update interface{})
}

// groupsMap combines a RWMutex with a map of MulticastStreams
type groupsMap struct {
	sync.RWMutex
	m map[uint32]BroadcastStream
}

// multicastStream implements MulticastStream interface
type multicastStream struct {
	logger *logrus.Entry
	groups *groupsMap
}

// NewMulticastStream creates a MulticastStream
func NewMulticastStream() MulticastStream {
	return &multicastStream{
		logger: utils.Logger.WithFields(logrus.Fields{
			"module": "datastreams.MulticastStream",
		}),
		groups: &groupsMap{
			m: make(map[uint32]BroadcastStream),
		},
	}
}

// AddListener adds a listener to a stream
func (ms *multicastStream) AddListener(groupId uint32, sessionId string, lis *listener) {
	l := ms.logger.WithFields(logrus.Fields{
		"method":          "AddListener",
		"param_groupId":   groupId,
		"param_sessionId": sessionId,
	})

	l.Debugf("Locking groups map")
	ms.groups.Lock()
	group, exists := ms.groups.m[groupId]
	if !exists {
		l.Debugf("Groups map not found. Creating")
		// create group if it doesn't exist
		ms.groups.m[groupId] = NewBroadcastStream()
		group = ms.groups.m[groupId]
	}
	ms.groups.Unlock()
	l.Debugf("Unlocked groups map")

	group.AddListener(sessionId, lis)

	l.Debugf("Added listener to map")
}

// RemoveListener removes a listener from a stream
func (ms *multicastStream) RemoveListener(groupId uint32, sessionId string) {
	l := ms.logger.WithFields(logrus.Fields{
		"method":          "RemoveListener",
		"param_groupId":   groupId,
		"param_sessionId": sessionId,
	})

	l.Debugf("Locking groups map")
	ms.groups.RLock()
	group, exists := ms.groups.m[groupId]
	ms.groups.RUnlock()
	l.Warnf("Unlocked groups map")

	if !exists {
		l.Warnf("Group not found. Shouldn't happen!")
		return
	}

	group.RemoveListener(sessionId)
	l.Debugf("Removed listener from group")

	if group.GetListenersCount() == 0 {
		l.Debugf("Removing group because of zero listeners")
		ms.groups.Lock()
		delete(ms.groups.m, groupId)
		ms.groups.Unlock()
		l.Debugf("Removed group")
	}
}

// BroadcastUpdate broadcasts a given update to all listeners in the stream
func (ms *multicastStream) BroadcastUpdateToGroup(groupId uint32, update interface{}) {
	l := ms.logger.WithFields(logrus.Fields{
		"method":        "BroadcastUpdateToGroup",
		"param_groupId": groupId,
	})

	l.Debugf("Locking groups map")
	ms.groups.RLock()
	group, exists := ms.groups.m[groupId]
	ms.groups.RUnlock()
	l.Warnf("Unlocked groups map")

	if !exists {
		l.Warnf("Group not found. Shouldn't happen!")
		return
	}

	group.BroadcastUpdate(update)
	l.Debugf("Removed listener from group")

	// this could happen if BroadcastUpdate removes dead listeners after sending the update
	if group.GetListenersCount() == 0 {
		l.Debugf("Removing group because of zero listeners")
		ms.groups.Lock()
		delete(ms.groups.m, groupId)
		ms.groups.Unlock()
		l.Debugf("Removed group")
	}
}
