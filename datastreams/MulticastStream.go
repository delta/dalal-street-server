package datastreams

import (
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/delta/dalal-street-server/utils"
)

// MulticastStream represents an object that provides methods for handling multiple groups in
// a stream. It allows you to broadcast updates to individual groups and manage the groups.
type MulticastStream interface {
	AddListener(groupId uint32, sessionId string, lis *listener)
	RemoveListener(groupId uint32, sessionId string)
	BroadcastUpdateToGroup(groupId uint32, update interface{})
	MakeGlobalBroadcast(update interface{})
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

// AddListener adds a listener to a group in a stream. Creates group if required
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

// RemoveListener removes a listener from a group of a stream. Removes group if empty.
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

// BroadcastUpdateToGroup broadcasts a given update to all listeners in the stream belonging to the given group
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

// MakeGlobalBroadcast broadcasts a given update to *all* listeners in the stream
func (ms *multicastStream) MakeGlobalBroadcast(update interface{}) {
	l := ms.logger.WithFields(logrus.Fields{
		"method": "MakeGlobalBroadcast",
	})

	// TODO: Remove group if no one's alive in it
	l.Debugf("Locking groups map")
	ms.groups.RLock()
	for _, group := range ms.groups.m {
		group.BroadcastUpdate(update)
	}
	ms.groups.RUnlock()
	l.Debugf("Unlocked groups map")
}
