package streamservice

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/Sirupsen/logrus"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/thakkarparth007/dalal-street-server/datastreams"
	"github.com/thakkarparth007/dalal-street-server/proto_build"
	"github.com/thakkarparth007/dalal-street-server/proto_build/datastreams"
	"github.com/thakkarparth007/dalal-street-server/session"

	"github.com/thakkarparth007/dalal-street-server/utils"
)

var logger *logrus.Entry

func getUserId(ctx context.Context) uint32 {
	sess := ctx.Value("session").(session.Session)
	userId, _ := sess.Get("userId")
	userIdInt, _ := strconv.ParseUint(userId, 10, 32)
	return uint32(userIdInt)
}

type subscription struct {
	subscribeReq *datastreams_pb.SubscribeRequest
	doneChan     chan struct{}
}
type perStreamTypeSubscriptionMap struct {
	sync.RWMutex
	m map[string]*subscription
}
type dalalStreamService struct {
	datastreamsManager datastreams.Manager
	subscriptionsMap   map[datastreams_pb.DataStreamType]*perStreamTypeSubscriptionMap
}

// NewDalalStreamService creates a new DalalStreamServer instnance
func NewDalalStreamService(dsm datastreams.Manager) pb.DalalStreamServiceServer {
	logger = utils.Logger.WithFields(logrus.Fields{
		"module": "grpcapi.actions",
	})

	dss := &dalalStreamService{
		datastreamsManager: dsm,
		subscriptionsMap:   make(map[datastreams_pb.DataStreamType]*perStreamTypeSubscriptionMap),
	}

	types := []datastreams_pb.DataStreamType{
		datastreams_pb.DataStreamType_MARKET_DEPTH,
		datastreams_pb.DataStreamType_MARKET_EVENTS,
		datastreams_pb.DataStreamType_MY_ORDERS,
		datastreams_pb.DataStreamType_NOTIFICATIONS,
		datastreams_pb.DataStreamType_STOCK_EXCHANGE,
		datastreams_pb.DataStreamType_STOCK_PRICES,
		datastreams_pb.DataStreamType_TRANSACTIONS,
		datastreams_pb.DataStreamType_STOCK_HISTORY,
	}

	for _, t := range types {
		dss.subscriptionsMap[t] = &perStreamTypeSubscriptionMap{
			sync.RWMutex{},
			make(map[string]*subscription),
		}
	}

	return dss
}

func (d *dalalStreamService) Unsubscribe(ctx context.Context, req *datastreams_pb.UnsubscribeRequest) (*datastreams_pb.UnsubscribeResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "Unsubscribe",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("Unsubscribe requested")

	resp := &datastreams_pb.UnsubscribeResponse{}
	id := req.SubscriptionId.Id
	dataStreamType := req.SubscriptionId.DataStreamType

	d.subscriptionsMap[dataStreamType].Lock()
	subscription, ok := d.subscriptionsMap[dataStreamType].m[id]
	if !ok {
		d.subscriptionsMap[dataStreamType].Unlock()
		return resp, nil
	}
	delete(d.subscriptionsMap[dataStreamType].m, id)
	d.subscriptionsMap[dataStreamType].Unlock()

	// closing the done channel will automatically remove the user from the stream
	close(subscription.doneChan)

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalStreamService) Subscribe(ctx context.Context, req *datastreams_pb.SubscribeRequest) (*datastreams_pb.SubscribeResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "Subscribe",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("Subscribe requested")

	resp := &datastreams_pb.SubscribeResponse{}

	subscriptionId := utils.RandString(32)
	d.subscriptionsMap[req.DataStreamType].Lock()
	// Infinite loop, but should ideally terminate in one iteration.
	for {
		_, ok := d.subscriptionsMap[req.DataStreamType].m[subscriptionId]
		if !ok {
			break
		}
	}
	d.subscriptionsMap[req.DataStreamType].m[subscriptionId] = &subscription{
		subscribeReq: req,
		doneChan:     make(chan struct{}),
	}
	d.subscriptionsMap[req.DataStreamType].Unlock()

	resp.SubscriptionId = &datastreams_pb.SubscriptionId{Id: subscriptionId, DataStreamType: req.DataStreamType}

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalStreamService) getSubscription(req *datastreams_pb.SubscriptionId, dsType datastreams_pb.DataStreamType) (*subscription, error) {
	id := req.Id
	dataStreamType := req.DataStreamType

	if dataStreamType != dsType {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid subscription id")
	}

	d.subscriptionsMap[dataStreamType].RLock()
	subscription, ok := d.subscriptionsMap[dataStreamType].m[id]
	d.subscriptionsMap[dataStreamType].RUnlock()
	if !ok {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid subscription id")
	}

	return subscription, nil
}

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
