package datastreams

import (
	"sync"

	"github.com/Sirupsen/logrus"

	datastreams_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/datastreams"
)

var orderListenersLock sync.Mutex

type orderListenersSingleUser struct {
	sync.Mutex
	l map[string]*listener
}

var orderListeners = make(map[uint32]*orderListenersSingleUser)

func SendOrderUpdate(userId uint32, orderId uint32, isAsk bool, tradeQuantity uint32, isClosed bool) {
	var l = logger.WithFields(logrus.Fields{
		"method":              "SendOrder",
		"param_orderId":       orderId,
		"param_isAsk":         isAsk,
		"param_tradeQuantity": tradeQuantity,
		"param_isClosed":      isClosed,
	})

	l.Debugf("Attempting")

	orderListenersLock.Lock()
	listeners, ok := orderListeners[userId]
	if !ok {
		l.Debugf("No listener found. Done.")
		orderListenersLock.Unlock()
		return
	}
	orderListenersLock.Unlock()

	orderUpdateProto := &datastreams_proto.MyOrderUpdate{
		Id:            orderId,
		IsAsk:         isAsk,
		TradeQuantity: tradeQuantity,
		IsClosed:      isClosed,
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
				delete(orderListeners, userId)
			}
		case listener.update <- orderUpdateProto:
			sent++
		}
	}
	listeners.Unlock()

	l.Debugf("Sent to %d listeners", listeners)
}

func RegOrdersListener(done <-chan struct{}, update chan interface{}, userId uint32, sessionId string) {
	var l = logger.WithFields(logrus.Fields{
		"method":       "RegOrderListener",
		"param_userId": userId,
	})

	l.Debugf("Attempting")

	orderListenersLock.Lock()
	olu, ok := orderListeners[userId]
	if !ok {
		orderListeners[userId] = &orderListenersSingleUser{
			l: make(map[string]*listener),
		}
		olu = orderListeners[userId]
	}
	orderListenersLock.Unlock()

	olu.Lock()
	olu.l[sessionId] = &listener{
		update,
		done,
	}
	olu.Unlock()

	l.Debugf("Appended to listeners")

	go func() {
		<-done
		UnregOrdersListener(userId, sessionId)
		l.Debugf("Removed dead listener")
	}()
}

func UnregOrdersListener(userId uint32, sessionId string) {
	orderListenersLock.Lock()
	defer orderListenersLock.Unlock()
	listeners, ok := orderListeners[userId]
	if !ok {
		return
	}
	listeners.Lock()
	delete(listeners.l, sessionId)
	if len(listeners.l) == 0 {
		delete(orderListeners, userId)
	}
	listeners.Unlock()
}
