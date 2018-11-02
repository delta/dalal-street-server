package datastreams

import (
	"github.com/Sirupsen/logrus"

	"github.com/delta/dalal-street-server/proto_build/datastreams"
	"github.com/delta/dalal-street-server/proto_build/models"
	"github.com/delta/dalal-street-server/utils"
)

// TransactionsStream defines the interface to interact with a Transactions stream
type TransactionsStream interface {
	SendTransaction(t *models_pb.Transaction)
	AddListener(done <-chan struct{}, updates chan interface{}, userId uint32, sessionId string)
	RemoveListener(userId uint32, sessionId string)
}

// transactionsStream implements the TransactionsStream
type transactionsStream struct {
	logger          *logrus.Entry
	multicastStream MulticastStream
}

// newTransactionsStream creates a new TransactionsStream
func newTransactionsStream() TransactionsStream {
	return &transactionsStream{
		logger: utils.Logger.WithFields(logrus.Fields{
			"module": "datastreams.TransactionsStream",
		}),
		multicastStream: NewMulticastStream(),
	}
}

// SendOrderUpdate sends an order update to a given user
func (os *transactionsStream) SendTransaction(t *models_pb.Transaction) {
	var l = os.logger.WithFields(logrus.Fields{
		"method":       "SendTransaction",
		"param_userId": t.UserId,
		"param_t":      t,
	})

	tUpdate := &datastreams_pb.TransactionUpdate{
		Transaction: t,
	}
	os.multicastStream.BroadcastUpdateToGroup(t.UserId, tUpdate)

	l.Infof("Sent")
}

// AddListener adds a listener to the Transactions stream
func (os *transactionsStream) AddListener(done <-chan struct{}, update chan interface{}, userId uint32, sessionId string) {
	var l = os.logger.WithFields(logrus.Fields{
		"method":          "AddListener",
		"param_userId":    userId,
		"param_sessionId": sessionId,
	})

	os.multicastStream.AddListener(userId, sessionId, &listener{
		update: update,
		done:   done,
	})

	l.Infof("Added")
}

// RemoveListener removes a listener from the Transactions stream
func (os *transactionsStream) RemoveListener(userId uint32, sessionId string) {
	var l = os.logger.WithFields(logrus.Fields{
		"method":          "RemoveListener",
		"param_userId":    userId,
		"param_sessionId": sessionId,
	})

	os.multicastStream.RemoveListener(userId, sessionId)

	l.Infof("Removed")
}
