package streamservice

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/Sirupsen/logrus"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/delta/dalal-street-server/datastreams"
	pb "github.com/delta/dalal-street-server/proto_build"
	datastreams_pb "github.com/delta/dalal-street-server/proto_build/datastreams"
	"github.com/delta/dalal-street-server/session"

	"github.com/delta/dalal-street-server/utils"
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

// will be called whenever Unsubscribe is called, or when doneChan gets closed due to network error or something
// returns a subscription. It'll be nil in case the subscription doesn't exist. If it exists, it means
// that was called by Unsubscribe. So it can close the doneChan. Otherwise this got called from one
// of the GetXUpdates()...and that happened because doneChan got closed due to network issue.
func (d *dalalStreamService) removeSubscriptionFromMap(subId *datastreams_pb.SubscriptionId) *subscription {
	id := subId.Id
	dataStreamType := subId.DataStreamType

	d.subscriptionsMap[dataStreamType].Lock()
	subscription, ok := d.subscriptionsMap[dataStreamType].m[id]
	if !ok {
		d.subscriptionsMap[dataStreamType].Unlock()
		return nil
	}
	delete(d.subscriptionsMap[dataStreamType].m, id)
	d.subscriptionsMap[dataStreamType].Unlock()
	return subscription
}

func (d *dalalStreamService) Unsubscribe(ctx context.Context, req *datastreams_pb.UnsubscribeRequest) (*datastreams_pb.UnsubscribeResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "Unsubscribe",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("Unsubscribe requested")

	resp := &datastreams_pb.UnsubscribeResponse{}
	subscription := d.removeSubscriptionFromMap(req.GetSubscriptionId())

	if subscription != nil {
		// closing the done channel will automatically remove the user from the stream
		close(subscription.doneChan)
	}

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
