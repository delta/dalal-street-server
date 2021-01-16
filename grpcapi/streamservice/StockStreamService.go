package streamservice

import (
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"
	pb "github.com/delta/dalal-street-server/proto_build"
	datastreams_pb "github.com/delta/dalal-street-server/proto_build/datastreams"
)

func (d *dalalStreamService) GetStockHistoryUpdates(req *datastreams_pb.SubscriptionId, stream pb.DalalStreamService_GetStockHistoryUpdatesServer) error {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetStockHistoryUpdates",
		"param_session": fmt.Sprintf("%+v", stream.Context().Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetStockHistoryUpdates requested")

	subscription, err := d.getSubscription(req, datastreams_pb.DataStreamType_STOCK_HISTORY)
	if err != nil {
		return err
	}

	subscribeReq := subscription.subscribeReq
	done := subscription.doneChan
	updates := make(chan interface{})

	stockId, _ := strconv.ParseUint(subscribeReq.DataStreamId, 10, 32)

	historyStream := d.datastreamsManager.GetStockHistoryStream(uint32(stockId))
	historyStream.AddListener(done, updates, req.Id)

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
			err := stream.Send(update.(*datastreams_pb.StockHistoryUpdate))
			if err != nil {
				// log the error
				break
			}
		}
	}
	l.Infof("Request completed successfully")

	return nil
}

func (d *dalalStreamService) GetStockExchangeUpdates(req *datastreams_pb.SubscriptionId, stream pb.DalalStreamService_GetStockExchangeUpdatesServer) error {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetStockExchangeUpdates",
		"param_session": fmt.Sprintf("%+v", stream.Context().Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetStockExchangeUpdates requested")

	subscription, err := d.getSubscription(req, datastreams_pb.DataStreamType_STOCK_EXCHANGE)
	if err != nil {
		return err
	}

	done := subscription.doneChan
	updates := make(chan interface{})

	stockExchangeStream := d.datastreamsManager.GetStockExchangeStream()
	stockExchangeStream.AddListener(done, updates, req.Id)

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
			err := stream.Send(update.(*datastreams_pb.StockExchangeUpdate))
			if err != nil {
				// log the error
				break
			}
		}
	}

	l.Infof("Request completed successfully")

	return nil
}

func (d *dalalStreamService) GetStockPricesUpdates(req *datastreams_pb.SubscriptionId, stream pb.DalalStreamService_GetStockPricesUpdatesServer) error {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetStockPricesUpdates",
		"param_session": fmt.Sprintf("%+v", stream.Context().Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetStockPricesUpdates requested")

	subscription, err := d.getSubscription(req, datastreams_pb.DataStreamType_STOCK_PRICES)
	if err != nil {
		return err
	}

	done := subscription.doneChan
	updates := make(chan interface{})

	stockPricesStream := d.datastreamsManager.GetStockPricesStream()
	stockPricesStream.AddListener(done, updates, req.Id)

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
			err := stream.Send(update.(*datastreams_pb.StockPricesUpdate))
			if err != nil {
				// log the error
				break
			}
		}
	}

	l.Infof("Request completed successfully")

	return nil
}
