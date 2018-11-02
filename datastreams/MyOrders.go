package datastreams

import (
	"github.com/Sirupsen/logrus"

	"github.com/delta/dalal-street-server/utils"
)

// MyOrdersStream defines the interface for interacting with the MyOrders datastream
type MyOrdersStream interface {
	SendOrder(userId uint32, ou *datastreams_pb.MyOrderUpdate)
	AddListener(done <-chan struct{}, update chan interface{}, userId uint32, sessionId string)
	RemoveListener(userId uint32, sessionId string)
}

// myOrdersStream implements the MyOrdersStream interface
type myOrdersStream struct {
	logger          *logrus.Entry
	multicastStream MulticastStream
}

// newMyOrdersStream creates a new MyOrdersStream
func newMyOrdersStream() MyOrdersStream {
	return &myOrdersStream{
		logger: utils.Logger.WithFields(logrus.Fields{
			"module": "datastreams.MyOrdersStream",
		}),
		multicastStream: NewMulticastStream(),
	}
}

// SendOrder sends an order update to a given user
func (os *myOrdersStream) SendOrder(userId uint32, ou *datastreams_pb.MyOrderUpdate) {
	var l = os.logger.WithFields(logrus.Fields{
		"method":       "SendOrder",
		"param_userId": userId,
		"param_ou":     ou,
	})

	os.multicastStream.BroadcastUpdateToGroup(userId, ou)

	l.Infof("Sent")
}

// AddListener adds a listener to the MyOrders stream
func (os *myOrdersStream) AddListener(done <-chan struct{}, update chan interface{}, userId uint32, sessionId string) {
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

// RemoveListener removes a listener from the MyOrders stream
func (os *myOrdersStream) RemoveListener(userId uint32, sessionId string) {
	var l = os.logger.WithFields(logrus.Fields{
		"method":          "RemoveListener",
		"param_userId":    userId,
		"param_sessionId": sessionId,
	})

	os.multicastStream.RemoveListener(userId, sessionId)

	l.Infof("Removed")
}
