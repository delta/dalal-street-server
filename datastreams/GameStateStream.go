package datastreams

import (
	"fmt"

	"github.com/Sirupsen/logrus"

	datastreams_pb "github.com/delta/dalal-street-server/proto_build/datastreams"
	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/delta/dalal-street-server/utils"
)

// GameStateStream represents the interface for handling a game state data stream
type GameStateStream interface {
	SendGameStateUpdate(gs *models_pb.GameState)
	AddListener(done <-chan struct{}, update chan interface{}, userId uint32, sessionId string)
	RemoveListener(userId uint32, sessionId string)
}

// GameStateStream implements the NotificationsStream interface
type gameStateStream struct {
	logger          *logrus.Entry
	multicastStream MulticastStream
}

// newGameStateStream creates a new GameStateStream
func newGameStateStream() GameStateStream {
	return &gameStateStream{
		logger: utils.Logger.WithFields(logrus.Fields{
			"module": "datastreams.GameStateStream",
		}),
		multicastStream: NewMulticastStream(),
	}
}

// SendGameStateUpdate sends a update to all connections of a given user
func (gs *gameStateStream) SendGameStateUpdate(g *models_pb.GameState) {
	var l = gs.logger.WithFields(logrus.Fields{
		"method":  "SendGameStateUpdate",
		"param_n": fmt.Sprintf("%+v", g),
	})

	gameStateUpdate := &datastreams_pb.GameStateUpdate{
		GameState: g,
	}
	// it's a broadcast. Send to everyone
	if g.GetUserId() == 0 {
		gs.multicastStream.MakeGlobalBroadcast(gameStateUpdate)
	} else {
		gs.multicastStream.BroadcastUpdateToGroup(g.GetUserId(), gameStateUpdate)
	}

	l.Infof("Game State Update to %d", g.GetUserId())
}

// AddListener adds a listener for a given user and connection
func (gs *gameStateStream) AddListener(done <-chan struct{}, update chan interface{}, userId uint32, sessionId string) {
	var l = gs.logger.WithFields(logrus.Fields{
		"method":          "AddListener",
		"param_userId":    userId,
		"param_sessionId": sessionId,
	})

	gs.multicastStream.AddListener(userId, sessionId, &listener{
		update: update,
		done:   done,
	})

	l.Infof("Added")
}

// RemoveListener removes a given listener from the subscribers list
func (gs *gameStateStream) RemoveListener(userId uint32, sessionId string) {
	var l = gs.logger.WithFields(logrus.Fields{
		"method":          "RemoveListener",
		"param_userId":    userId,
		"param_sessionId": sessionId,
	})

	gs.multicastStream.RemoveListener(userId, sessionId)

	l.Infof("Removed")
}
