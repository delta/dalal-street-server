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

	var userIds []uint32

	notifListenersLock.Lock()
	if n.UserId != 0 {
		if _, ok := notifListeners[n.UserId]; !ok {
			l.Debugf("No listener found. Done.")
			notifListenersLock.Unlock()
			return
		}
		userIds = append(userIds, n.UserId)
	} else {
		for userId := range notifListeners {
			userIds = append(userIds, userId)
		}
	}
	notifListenersLock.Unlock()

	notifUpdateProto := &datastreams_proto.NotificationUpdate{
		n,
	}

	sent := 0
	l.Debugf("Sending to %d listeners", len(userIds))
	for _, userId := range userIds {
		notifListenersLock.Lock()
		listeners := notifListeners[userId]
		notifListenersLock.Unlock()

		listeners.Lock()
		for sessId, listener := range listeners.l {
			select {
			case <-listener.done:
				l.Debugf("One has already left. Removing him.")
				delete(listeners.l, sessId)
				if len(listeners.l) == 0 {
					notifListenersLock.Lock()
					delete(notifListeners, n.UserId)
					notifListenersLock.Unlock()
				}
			case listener.update <- notifUpdateProto:
				sent++
			}
		}
		listeners.Unlock()
	}

	l.Debugf("Sent to %d listeners", sent)
}

func RegNotificationsListener(done <-chan struct{}, update chan interface{}, userId uint32, sessionId string) {
	var l = logger.WithFields(logrus.Fields{
		"method":          "RegNotificationListener",
		"param_userId":    userId,
		"param_sessionId": sessionId,
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

	nlu.Lock()
	nlu.l[sessionId] = &listener{
		update,
		done,
	}
	nlu.Unlock()

	l.Debugf("Appended to listeners")

	go func() {
		<-done
		UnregNotificationsListener(userId, sessionId)
		l.Debugf("Removed dead listener")
	}()
}

func UnregNotificationsListener(userId uint32, sessionId string) {
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
}
