package datastreams

import (
	"github.com/Sirupsen/logrus"
	datastreams_pb "github.com/thakkarparth007/dalal-street-server/proto_build/datastreams"
	models_pb "github.com/thakkarparth007/dalal-street-server/proto_build/models"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

// StockHistoryStream interface defines the interface to interact with StockHistory stream
type StockHistoryStream interface {
	SendStockHistoryUpdate(stockId uint32, history *models_pb.StockHistory)
	AddListener(done <-chan struct{}, updates chan interface{}, sessionId string)
	RemoveListener(sessionId string)
}

// stockHistoryStream implements the StockHistory interface
type stockHistoryStream struct {
	logger          *logrus.Entry
	broadcastStream BroadcastStream
	stockId         uint32
}

// newStockHistoryStream returns a new instance of StockHistoryStream
func newStockHistoryStream(stockId uint32) StockHistoryStream {
	shs := &stockHistoryStream{
		stockId: stockId,
		logger: utils.Logger.WithFields(logrus.Fields{
			"module":  "datastreams.StockHistoryStream",
			"stockId": stockId,
		}),
		broadcastStream: NewBroadcastStream(),
	}
	return shs
}

// AddListener adds a listener to the StockHistory stream
func (shs *stockHistoryStream) AddListener(done <-chan struct{}, update chan interface{}, sessionId string) {
	var l = shs.logger.WithFields(logrus.Fields{
		"method":          "addListener",
		"param_sessionId": sessionId,
	})

	l.Infof("Sending History Broadcast")

	shs.broadcastStream.AddListener(sessionId, &listener{
		done:   done,
		update: update,
	})
}

// RemoveListener removes a listener from the StockHistory stream
func (shs *stockHistoryStream) RemoveListener(sessionId string) {
	var l = shs.logger.WithFields(logrus.Fields{
		"method":          "RemoveListener",
		"param_sessionId": sessionId,
	})

	l.Infof("Removed")

	shs.RemoveListener(sessionId)
}

// SendStockHistoryUpdate sends out an update to all the listeners
func (shs *stockHistoryStream) SendStockHistoryUpdate(stockId uint32, history *models_pb.StockHistory) {
	var l = shs.logger.WithFields(logrus.Fields{
		"method": "StockHistory",
	})

	l.Infof("Sending History Broadcast")
	update := &datastreams_pb.StockHistoryUpdate{
		StockHistory: history,
	}

	shs.broadcastStream.BroadcastUpdate(update)
}
