package datastreams

import (
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/delta/dalal-street-server/utils"
	"github.com/golang/mock/gomock"
)

func getMockBroadcasrStream(t *testing.T) (*gomock.Controller, *broadcastStream, *listener, string) {
	var sessionID string = "123456789"
	mockControl := gomock.NewController(t)
	mocklistener := *listener
	mbroadcastStream := &broadcastStream{
		logger: utils.Logger.WithFields(logrus.Fields{
			"module": "datastreams.BroadcastStream.test",
		}),
		listeners: &listenersMap{
			m: make(map[string]*listener),
		},
	}
	return mockControl, mbroadcastStream, mocklistener, sessionID
}

func TestAddListener(t *testing.T) {

	conifg := utils.GetConfiguration()
	utils.Init(conifg)

	mockControl, mbroadcastStream, mocklisterner, sessionID := getMockBroadcasrStream(t)
	defer mockControl.Finish()
	l := mbroadcastStream.logger.WithFields(logrus.Fields{
		"method": "Test AddListener",
	})
	mbroadcastStream.listeners.Lock()
	mbroadcastStream.listeners.m[sessionID] = mocklisterner
	mbroadcastStream.listeners.Unlock()

	l.Debugf("AddListener Tested")
}

func TestRemoveListeners(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, mbroadcastStream, mocklistener, sessionID := getMockBroadcasrStream(t)
	defer mockControl.Finish()
	l := mbroadcastStream.logger.WithFields(logrus.Fields{
		"method":          "Test RemoveListener",
		"param_sessionID": sessionID,
	})

	mbroadcastStream.listeners.Lock()
	delete(mbroadcastStream.listeners.m, sessionID)
	mbroadcastStream.listeners.Unlock()
	l.Debugf("RemoveListener Removed")

}

func TestBroadcastUpdate(t *testing.T) {
	mockControl, mbroadcastStream, mocklistener, sessionID := getMockBroadcasrStream(t)
	l := mbroadcastStream.logger.WithFields(logrus.Fields{
		"method": "TestBroadcastUpdate",
	})

	mbroadcastStream.listeners.Lock()
	//dont know what to do exactly
	mbroadcastStream.listeners.Unlock()
}

func TestGetListenerCount(t *testing.T) {
	mockControl, mbroadcastStream, mocklistener, sessionID := getMockBroadcasrStream(t)
	l := mbroadcastStream.logger.WithFields(logrus.Fields{
		"method": "TestGetListenerCount",
	})

	mbroadcastStream.listeners.RLock()
	//todo
	mbroadcastStream.listeners.RUnlock()
}
