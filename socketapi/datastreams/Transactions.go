package datastreams

import (
	"fmt"
	"sync"

	"github.com/Sirupsen/logrus"

	datastreams_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/datastreams"
	models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
)

var transListenersLock sync.Mutex

type transListenersSingleUser struct {
	sync.Mutex
	l map[string]*listener
}

var transListeners = make(map[uint32]*transListenersSingleUser)

func SendTransaction(n *models_proto.Transaction) {
	var l = logger.WithFields(logrus.Fields{
		"method":  "SendTransaction",
		"param_n": fmt.Sprintf("%+v", n),
	})

	l.Debugf("Attempting")

	transListenersLock.Lock()
	listeners, ok := transListeners[n.UserId]
	if !ok {
		l.Debugf("No listener found. Done.")
		transListenersLock.Unlock()
		return
	}
	transListenersLock.Unlock()

	transUpdateProto := &datastreams_proto.TransactionUpdate{
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
				transListenersLock.Lock()
				delete(transListeners, n.UserId)
				transListenersLock.Unlock()
			}
		case listener.update <- transUpdateProto:
			sent++
		}
	}
	listeners.Unlock()

	l.Debugf("Sent to %d listeners", listeners)
}

func RegTransactionsListener(done <-chan struct{}, update chan interface{}, userId uint32, sessionId string) {
	var l = logger.WithFields(logrus.Fields{
		"method":       "RegTransactionListener",
		"param_userId": userId,
	})

	l.Debugf("Attempting")

	transListenersLock.Lock()
	tlu, ok := transListeners[userId]
	if !ok {
		transListeners[userId] = &transListenersSingleUser{
			l: make(map[string]*listener),
		}
		tlu = transListeners[userId]
	}
	transListenersLock.Unlock()

	tlu.Lock()
	tlu.l[sessionId] = &listener{
		update,
		done,
	}
	tlu.Unlock()

	l.Debugf("Appended to listeners")

	go func() {
		<-done
		UnregTransactionsListener(userId, sessionId)
		l.Debugf("Removed dead listener")
	}()
}

func UnregTransactionsListener(userId uint32, sessionId string) {
	transListenersLock.Lock()
	defer transListenersLock.Unlock()
	listeners, ok := transListeners[userId]
	if !ok {
		return
	}
	listeners.Lock()
	delete(listeners.l, sessionId)
	if len(listeners.l) == 0 {
		delete(transListeners, userId)
	}
	listeners.Unlock()
}
