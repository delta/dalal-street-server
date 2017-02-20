package datastreams

import (
	"runtime/debug"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"

	models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
)

var (
	marketEventsListenersMutex sync.Mutex
	marketEventsListeners      = make(map[string]listener)
)

func SendMarketEvent(updateProto *models_proto.MarketEvent) {
	var l = logger.WithFields(logrus.Fields{
		"method":            "SendMarketEvent",
		"param_updateProto": updateProto,
	})

	defer func() {
		if r := recover(); r != nil {
			l.Errorf("Error! Stack trace: %s", string(debug.Stack()))
		}
	}()

	sent := 0
	marketEventsListenersMutex.Lock()
	l.Debugf("Will be sending %+v to %d listeners", updateProto, len(marketEventsListeners))

	for sessionId, listener := range marketEventsListeners {
		select {
		case <-listener.done:
			delete(marketEventsListeners, sessionId)
			l.Debugf("Found dead listener. Removed")
		case listener.update <- updateProto:
			sent++
		}
	}

	marketEventsListenersMutex.Unlock()

	l.Debugf("Sent to %d listeners!. Sleeping for 15 seconds", sent)

	time.Sleep(time.Minute / 4)
}

func RegMarketEventsListener(done <-chan struct{}, update chan interface{}, sessionId string) {
	var l = logger.WithFields(logrus.Fields{
		"method": "RegMarketEventsListener",
	})
	l.Debugf("Got a listener")

	marketEventsListenersMutex.Lock()
	defer marketEventsListenersMutex.Unlock()

	if oldlistener, ok := marketEventsListeners[sessionId]; ok {
		// remove the old listener.
		close(oldlistener.update)
	}
	marketEventsListeners[sessionId] = listener{
		update,
		done,
	}

	go func() {
		<-done
		UnregMarketEventsListener(sessionId)
		l.Debugf("Found dead listener. Removed")
	}()
}

func UnregMarketEventsListener(sessionId string) {
	marketEventsListenersMutex.Lock()
	delete(marketEventsListeners, sessionId)
	marketEventsListenersMutex.Unlock()
}
