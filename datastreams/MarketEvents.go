package datastreams

import (
	"github.com/sirupsen/logrus"

	"github.com/delta/dalal-street-server/proto_build/datastreams"
	"github.com/delta/dalal-street-server/proto_build/models"
	"github.com/delta/dalal-street-server/utils"
)

// MarketEventsStream defines the interface for interacting with the MyOrders datastream
type MarketEventsStream interface {
	SendMarketEvent(me *models_pb.MarketEvent)
	AddListener(done <-chan struct{}, update chan interface{}, sessionId string)
	RemoveListener(sessionId string)
}

// marketEventsStream implements the MarketEventsStream interface
type marketEventsStream struct {
	logger          *logrus.Entry
	broadcastStream BroadcastStream
}

// newMarketEventsStream creates a new MarketEventsStream
func newMarketEventsStream() MarketEventsStream {
	return &marketEventsStream{
		logger: utils.Logger.WithFields(logrus.Fields{
			"module": "datastreams.MarketEventsStream",
		}),
		broadcastStream: NewBroadcastStream(),
	}
}

// SendOrderUpdate sends an order update to a given user
func (mes *marketEventsStream) SendMarketEvent(me *models_pb.MarketEvent) {
	var l = mes.logger.WithFields(logrus.Fields{
		"method":   "SendMarketEvent",
		"param_me": me,
	})

	meUpdate := &datastreams_pb.MarketEventUpdate{
		MarketEvent: me,
	}
	mes.broadcastStream.BroadcastUpdate(meUpdate)

	l.Infof("Sent")
}

// AddListener adds a listener to the MyOrders stream
func (mes *marketEventsStream) AddListener(done <-chan struct{}, update chan interface{}, sessionId string) {
	var l = mes.logger.WithFields(logrus.Fields{
		"method":          "AddListener",
		"param_sessionId": sessionId,
	})

	mes.broadcastStream.AddListener(sessionId, &listener{
		update: update,
		done:   done,
	})

	l.Infof("Added")
}

// RemoveListener removes a listener from the MyOrders stream
func (mes *marketEventsStream) RemoveListener(sessionId string) {
	var l = mes.logger.WithFields(logrus.Fields{
		"method":          "RemoveListener",
		"param_sessionId": sessionId,
	})

	mes.broadcastStream.RemoveListener(sessionId)

	l.Infof("Removed")
}
