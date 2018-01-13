package datastreams

import (
	"fmt"

	"github.com/Sirupsen/logrus"

	"github.com/thakkarparth007/dalal-street-server/proto_build/datastreams"
	"github.com/thakkarparth007/dalal-street-server/proto_build/models"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

// NotificationsStream represents the interface for handling a notifications data stream
type NotificationsStream interface {
	SendNotification(n *models_pb.Notification)
	AddListener(done <-chan struct{}, update chan interface{}, userId uint32, sessionId string)
	RemoveListener(userId uint32, sessionId string)
}

// notificationsStream implements the NotificationsStream interface
type notificationsStream struct {
	logger          *logrus.Entry
	multicastStream MulticastStream
}

// newNotificationsStream creates a new NotificationStream
func newNotificationsStream() NotificationsStream {
	return &notificationsStream{
		logger: utils.Logger.WithFields(logrus.Fields{
			"module": "datastreams.NotificationsStream",
		}),
		multicastStream: NewMulticastStream(),
	}
}

// SendNotification sends a notification to all connections of a given user
func (ns *notificationsStream) SendNotification(n *models_pb.Notification) {
	var l = ns.logger.WithFields(logrus.Fields{
		"method":  "SendNotification",
		"param_n": fmt.Sprintf("%+v", n),
	})

	notifUpdate := &datastreams_pb.NotificationUpdate{
		Notification: n,
	}
	ns.multicastStream.BroadcastUpdateToGroup(n.GetUserId(), notifUpdate)
	l.Infof("Sent notification to %d", n.GetUserId())
}

// AddListener adds a listener for a given user and connection
func (ns *notificationsStream) AddListener(done <-chan struct{}, update chan interface{}, userId uint32, sessionId string) {
	var l = ns.logger.WithFields(logrus.Fields{
		"method":          "AddListener",
		"param_userId":    userId,
		"param_sessionId": sessionId,
	})

	ns.multicastStream.AddListener(userId, sessionId, &listener{
		update: update,
		done:   done,
	})

	l.Infof("Added")
}

// RemoveListener removes a given listener from the subscribers list
func (ns *notificationsStream) RemoveListener(userId uint32, sessionId string) {
	var l = ns.logger.WithFields(logrus.Fields{
		"method":          "RemoveListener",
		"param_userId":    userId,
		"param_sessionId": sessionId,
	})

	ns.multicastStream.RemoveListener(userId, sessionId)

	l.Infof("Removed")
}
