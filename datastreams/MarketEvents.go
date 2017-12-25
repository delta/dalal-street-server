package datastreams

import (
	"runtime/debug"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"

	"github.com/thakkarparth007/dalal-street-server/proto_build/datastreams"
	"github.com/thakkarparth007/dalal-street-server/proto_build/models"
)

var (
	marketEventsListenersMutex sync.Mutex
	marketEventsListeners      = make(map[string]listener)
)

func SendMarketEvent(meProto *models_pb.MarketEvent) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "SendMarketEvent",
		"param_meProto": meProto,
	})

	defer func() {
		if r := recover(); r != nil {
			l.Errorf("Error! Stack trace: %s", string(debug.Stack()))
		}
	}()

	updateProto := &datastreams_pb.MarketEventUpdate{
		meProto,
	}

	sent := 0
	marketEventsListenersMutex.Lock()
	l.Debugf("Will be sending %+v to %d listeners", updateProto, len(marketEventsListeners))

	for sessionId, listener := range marketEventsListeners {
		select {
		case <-listener.done:
			delete(marketEventsListeners, sessionId)
			l.Debugf("Found sid %s dead. Removed", sessionId)
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
		"method":          "RegMarketEventsListener",
		"param_sessionId": sessionId,
	})
	l.Debugf("Got a listener.")

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
		l.Debugf("Found a dead listener. Removed")
	}()
}

func UnregMarketEventsListener(sessionId string) {
	marketEventsListenersMutex.Lock()
	delete(marketEventsListeners, sessionId)
	marketEventsListenersMutex.Unlock()
}
