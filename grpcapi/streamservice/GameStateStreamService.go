package streamservice

import (
	"fmt"

	"github.com/sirupsen/logrus"
	pb "github.com/delta/dalal-street-server/proto_build"
	datastreams_pb "github.com/delta/dalal-street-server/proto_build/datastreams"
)

func (d *dalalStreamService) GetGameStateUpdates(req *datastreams_pb.SubscriptionId, stream pb.DalalStreamService_GetGameStateUpdatesServer) error {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetGameStateUpdates",
		"param_session": fmt.Sprintf("%+v", stream.Context().Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetGameStateUpdates requested")

	subscription, err := d.getSubscription(req, datastreams_pb.DataStreamType_GAME_STATE)
	if err != nil {
		return err
	}

	done := subscription.doneChan
	updates := make(chan interface{})

	userId := getUserId(stream.Context())
	gameStateStream := d.datastreamsManager.GetGameStateStream()
	gameStateStream.AddListener(done, updates, userId, req.Id)

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
			err := stream.Send(update.(*datastreams_pb.GameStateUpdate))
			if err != nil {
				// log the error
				break
			}
		}
	}
	l.Infof("Request completed successfully")

	return nil
}
