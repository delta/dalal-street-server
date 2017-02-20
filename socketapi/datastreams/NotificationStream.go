package datastreams

import (
	"fmt"
	"sync"

	"github.com/Sirupsen/logrus"

	datastreams_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/datastreams"
	models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
)

var notifListenersLock sync.Mutex

type notifListenersSingleUser struct {
	sync.Mutex
	l map[string]*listener
}

var notifListeners = make(map[uint32]*notifListenersSingleUser)

func SendNotification(n *models_proto.Notification) {
	var l = logger.WithFields(logrus.Fields{
		"method":  "SendNotification",
		"param_n": fmt.Sprintf("%+v", n),
	})

	l.Debugf("Attempting")

	notifListenersLock.Lock()
	listeners, ok := notifListeners[n.UserId]
	if !ok {
		l.Debugf("No listener found. Done.")
		return
	}
	notifListenersLock.Unlock()

	notifUpdateProto := &datastreams_proto.NotificationUpdate{
		n,
	}

	listeners.Lock()
	l.Debugf("Sending to %d listeners", listeners)
	sent := 0
	for sessId, listener := range listeners.l {
		select {
		case <-listener.done:
			l.Debugf("One has already left. Removing him.")
			delete(listeners.l, sessId)
			if len(listeners.l) == 0 {
				delete(notifListeners, n.UserId)
			}
		case listener.update <- notifUpdateProto:
			sent++
		}
	}
	listeners.Unlock()

	l.Debugf("Sent to %d listeners", listeners)
}

func RegNotificationListener(done <-chan struct{}, update chan interface{}, userId uint32, sessionId string) {
	var l = logger.WithFields(logrus.Fields{
		"method":       "RegNotificationListener",
		"param_userId": userId,
	})

	l.Debugf("Attempting")

	notifListenersLock.Lock()
	defer notifListenersLock.Unlock()
	nlu, ok := notifListeners[userId]
	if !ok {
		notifListeners[userId] = &notifListenersSingleUser{
			l: make(map[string]*listener),
		}
		nlu = notifListeners[userId]
	}

	nlu.l[sessionId] = &listener{
		update,
		done,
	}

	l.Debugf("Appended to listeners")

	go func() {
		<-done
		notifListenersLock.Lock()
		defer notifListenersLock.Unlock()
		listeners, ok := notifListeners[userId]
		if !ok {
			return
		}
		listeners.Lock()
		delete(listeners.l, sessionId)
		if len(listeners.l) == 0 {
			delete(notifListeners, userId)
		}
		listeners.Unlock()
	}()
}
