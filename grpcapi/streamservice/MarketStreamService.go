package streamservice

import (
	"fmt"
	"strconv"

	pb "github.com/delta/dalal-street-server/proto_build"
	datastreams_pb "github.com/delta/dalal-street-server/proto_build/datastreams"
	"github.com/sirupsen/logrus"
)

func (d *dalalStreamService) GetMarketDepthUpdates(req *datastreams_pb.SubscriptionId, stream pb.DalalStreamService_GetMarketDepthUpdatesServer) error {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetMarketDepthUpdates",
		"param_session": fmt.Sprintf("%+v", stream.Context().Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetMarketDepthUpdates requested")

	subscription, err := d.getSubscription(req, datastreams_pb.DataStreamType_MARKET_DEPTH)
	if err != nil {
		return err
	}

	subscribeReq := subscription.subscribeReq
	done := subscription.doneChan
	updates := make(chan interface{})

	stockId, _ := strconv.ParseUint(subscribeReq.DataStreamId, 10, 32)

	depthStream := d.datastreamsManager.GetMarketDepthStream(uint32(stockId))
	depthStream.AddListener(done, updates, req.Id)

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
			err := stream.Send(update.(*datastreams_pb.MarketDepthUpdate))
			if err != nil {
				// log the error
				break
			}
		}
	}
	l.Infof("Request completed successfully")

	return nil
}

func (d *dalalStreamService) GetMarketEventUpdates(req *datastreams_pb.SubscriptionId, stream pb.DalalStreamService_GetMarketEventUpdatesServer) error {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetMarketEventUpdates",
		"param_session": fmt.Sprintf("%+v", stream.Context().Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetMarketEventUpdates requested")

	subscription, err := d.getSubscription(req, datastreams_pb.DataStreamType_MARKET_EVENTS)
	if err != nil {
		return err
	}

	done := subscription.doneChan
	updates := make(chan interface{})

	marketEventsStream := d.datastreamsManager.GetMarketEventsStream()
	marketEventsStream.AddListener(done, updates, req.Id)

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
			err := stream.Send(update.(*datastreams_pb.MarketEventUpdate))
			if err != nil {
				// log the error
				break
			}
		}
	}
	l.Infof("Request completed successfully")

	return nil
}
