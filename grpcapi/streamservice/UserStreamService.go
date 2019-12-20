package streamservice

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	pb "github.com/delta/dalal-street-server/proto_build"
	datastreams_pb "github.com/delta/dalal-street-server/proto_build/datastreams"
)

func (d *dalalStreamService) GetMyOrderUpdates(req *datastreams_pb.SubscriptionId, stream pb.DalalStreamService_GetMyOrderUpdatesServer) error {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetMyOrderUpdates",
		"param_session": fmt.Sprintf("%+v", stream.Context().Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetMyOrderUpdates requested")

	subscription, err := d.getSubscription(req, datastreams_pb.DataStreamType_MY_ORDERS)
	if err != nil {
		return err
	}

	done := subscription.doneChan
	updates := make(chan interface{})

	userId := getUserId(stream.Context())
	myOrdersStream := d.datastreamsManager.GetMyOrdersStream()
	myOrdersStream.AddListener(done, updates, userId, req.Id)

loop:
	for {
		select {
		case <-done:
			break loop
		case <-stream.Context().Done():
			d.removeSubscriptionFromMap(req)
			close(done)
			break loop
		case update := <-updates:
			err := stream.Send(update.(*datastreams_pb.MyOrderUpdate))
			if err != nil {
				// log the error
				break
			}
		}
	}
	l.Infof("Request completed successfully")

	return nil
}

func (d *dalalStreamService) GetNotificationUpdates(req *datastreams_pb.SubscriptionId, stream pb.DalalStreamService_GetNotificationUpdatesServer) error {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetNotificationUpdates",
		"param_session": fmt.Sprintf("%+v", stream.Context().Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetNotificationUpdates requested")

	subscription, err := d.getSubscription(req, datastreams_pb.DataStreamType_NOTIFICATIONS)
	if err != nil {
		return err
	}

	done := subscription.doneChan
	updates := make(chan interface{})

	userId := getUserId(stream.Context())
	notificationsStream := d.datastreamsManager.GetNotificationsStream()
	notificationsStream.AddListener(done, updates, userId, req.Id)

loop:
	for {
		select {
		case <-done:
			break loop
		case <-stream.Context().Done():
			d.removeSubscriptionFromMap(req)
			close(done)
			break loop
		case update := <-updates:
			err := stream.Send(update.(*datastreams_pb.NotificationUpdate))
			if err != nil {
				// log the error
				break
			}
		}
	}
	l.Infof("Request completed successfully")

	return nil
}

func (d *dalalStreamService) GetTransactionUpdates(req *datastreams_pb.SubscriptionId, stream pb.DalalStreamService_GetTransactionUpdatesServer) error {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetTransactionsUpdates",
		"param_session": fmt.Sprintf("%+v", stream.Context().Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetTransactionsUpdates requested")

	subscription, err := d.getSubscription(req, datastreams_pb.DataStreamType_TRANSACTIONS)
	if err != nil {
		return err
	}

	done := subscription.doneChan
	updates := make(chan interface{})

	userId := getUserId(stream.Context())
	transactionsStream := d.datastreamsManager.GetTransactionsStream()
	transactionsStream.AddListener(done, updates, userId, req.Id)

loop:
	for {
		select {
		case <-done:
			break loop
		case <-stream.Context().Done():
			d.removeSubscriptionFromMap(req)
			close(done)
			break loop
		case update := <-updates:
			err := stream.Send(update.(*datastreams_pb.TransactionUpdate))
			if err != nil {
				// log the error
				break
			}
		}
	}
	l.Infof("Request completed successfully")

	return nil
}
