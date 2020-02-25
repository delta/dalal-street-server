package actionservice

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/delta/dalal-street-server/models"
	actions_pb "github.com/delta/dalal-street-server/proto_build/actions"
	"golang.org/x/net/context"
)

func (d *dalalActionService) CancelOrder(ctx context.Context, req *actions_pb.CancelOrderRequest) (*actions_pb.CancelOrderResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "CancelOrder",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("CancelOrder requested")

	resp := &actions_pb.CancelOrderResponse{}
	makeError := func(st actions_pb.CancelOrderResponse_StatusCode, msg string) (*actions_pb.CancelOrderResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	if !models.IsMarketOpen() {
		return makeError(actions_pb.CancelOrderResponse_MarketClosedError, "Market is closed. You cannot cancel orders right now.")
	}

	userId := getUserId(ctx)
	if !models.IsUserPhoneVerified(userId) {
		return makeError(actions_pb.CancelOrderResponse_UserNotPhoneVerfiedError, "Your phone number has not been verified. Please verify phone number in order to play the game.")
	}

	if models.IsUserBlocked(userId) {
		return makeError(actions_pb.CancelOrderResponse_UserBlockedError, "Your account has been blocked due to malpractice.")
	}

	orderId := req.OrderId
	isAsk := req.IsAsk

	askOrder, bidOrder, err := models.CancelOrder(userId, orderId, isAsk)

	switch err.(type) {
	case models.InvalidOrderIDError:
		return makeError(actions_pb.CancelOrderResponse_InvalidOrderId, "Invalid Order ID. Cannot cancel this order.")
	case models.AlreadyClosedError:
		return makeError(actions_pb.CancelOrderResponse_InvalidOrderId, err.Error())
	}

	if err != nil {
		l.Errorf("Request failed due to %+v", err)
		return makeError(actions_pb.CancelOrderResponse_InternalServerError, getInternalErrorMessage(err))
	}

	// remove the order from matching engine
	if isAsk {
		d.matchingEngine.CancelAskOrder(askOrder)
	} else {
		d.matchingEngine.CancelBidOrder(bidOrder)
	}

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalActionService) PlaceOrder(ctx context.Context, req *actions_pb.PlaceOrderRequest) (*actions_pb.PlaceOrderResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "PlaceOrder",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	l.Infof("PlaceOrder requested")

	resp := &actions_pb.PlaceOrderResponse{}
	makeError := func(st actions_pb.PlaceOrderResponse_StatusCode, msg string) (*actions_pb.PlaceOrderResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	if !models.IsMarketOpen() {
		return makeError(actions_pb.PlaceOrderResponse_MarketClosedError, "Market Is closed. You cannot place orders right now.")
	}

	userId := getUserId(ctx)
	if !models.IsUserPhoneVerified(userId) {
		return makeError(actions_pb.PlaceOrderResponse_UserNotPhoneVerfiedError, "Your phone number has not been verified. Please verify phone number in order to play the game.")
	}

	if models.IsUserBlocked(userId) {
		return makeError(actions_pb.PlaceOrderResponse_UserBlockedError, "Your account has been blocked due to malpractice.")
	}

	var orderId uint32
	var err error

	if req.IsAsk {
		ask := &models.Ask{
			UserId:        userId,
			StockId:       req.StockId,
			OrderType:     models.OrderTypeFromProto(req.OrderType),
			Price:         req.Price,
			StockQuantity: req.StockQuantity,
		}
		orderId, err = models.PlaceAskOrder(userId, ask)
		if err == nil {
			go d.matchingEngine.AddAskOrder(ask)
		}
	} else {
		bid := &models.Bid{
			UserId:        userId,
			StockId:       req.StockId,
			OrderType:     models.OrderTypeFromProto(req.OrderType),
			Price:         req.Price,
			StockQuantity: req.StockQuantity,
		}
		orderId, err = models.PlaceBidOrder(userId, bid)
		if err == nil {
			go d.matchingEngine.AddBidOrder(bid)
		}
	}

	switch e := err.(type) {
	case models.OrderStockLimitExceeded:
		return makeError(actions_pb.PlaceOrderResponse_StockQuantityLimitExceeded, e.Error())
	case models.OrderPriceOutOfWindowError: // should update proto as well ideally. This is how it is for now.
		return makeError(actions_pb.PlaceOrderResponse_StockQuantityLimitExceeded, e.Error())
	case models.NotEnoughStocksError:
		return makeError(actions_pb.PlaceOrderResponse_NotEnoughStocksError, e.Error())
	case models.NotEnoughCashError:
		return makeError(actions_pb.PlaceOrderResponse_NotEnoughCashError, e.Error())
	case models.StockBankruptError:
		return makeError(actions_pb.PlaceOrderResponse_StockBankruptError, err.Error())
	}

	if err != nil {
		l.Errorf("Request failed due to: %+v", err)
		return makeError(actions_pb.PlaceOrderResponse_InternalServerError, getInternalErrorMessage(err))
	}

	resp.OrderId = orderId

	l.Infof("Request completed successfully")

	return resp, nil
}

//Returns open asks and open bids
func (d *dalalActionService) GetMyOpenOrders(ctx context.Context, req *actions_pb.GetMyOpenOrdersRequest) (*actions_pb.GetMyOpenOrdersResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetMyOpenOrders",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetMyOpenOrders requested")

	resp := &actions_pb.GetMyOpenOrdersResponse{}
	makeError := func(st actions_pb.GetMyOpenOrdersResponse_StatusCode, msg string) (*actions_pb.GetMyOpenOrdersResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	userId := getUserId(ctx)

	//get open ask orders
	myOpenAskOrders, err := models.GetMyOpenAsks(userId)
	if err != nil {
		l.Errorf("Request failed due to: %+v", err)
		return makeError(actions_pb.GetMyOpenOrdersResponse_InternalServerError, getInternalErrorMessage(err))
	}

	//convert open ask orders to proto
	for _, ask := range myOpenAskOrders {
		resp.OpenAskOrders = append(resp.OpenAskOrders, ask.ToProto())
	}

	//get open bid orders
	myOpenBidOrders, err := models.GetMyOpenBids(userId)
	if err != nil {
		l.Errorf("Request failed due to: %+v", err)
		return makeError(actions_pb.GetMyOpenOrdersResponse_InternalServerError, getInternalErrorMessage(err))
	}

	//convert open bid orders to proto
	for _, bid := range myOpenBidOrders {
		resp.OpenBidOrders = append(resp.OpenBidOrders, bid.ToProto())
	}

	l.Infof("Request completed successfully")

	return resp, nil
}

//Returns closed asks
func (d *dalalActionService) GetMyClosedAsks(ctx context.Context, req *actions_pb.GetMyClosedAsksRequest) (*actions_pb.GetMyClosedAsksResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetMyClosedAsks",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	l.Infof("GetMyClosedAsks requested")

	resp := &actions_pb.GetMyClosedAsksResponse{}
	makeError := func(st actions_pb.GetMyClosedAsksResponse_StatusCode, msg string) (*actions_pb.GetMyClosedAsksResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	userId := getUserId(ctx)
	lastId := req.LastOrderId
	count := req.Count

	moreExists, myClosedAskOrders, err := models.GetMyClosedAsks(userId, lastId, count)
	if err != nil {
		l.Errorf("Request failed due to: %+v", err)
		return makeError(actions_pb.GetMyClosedAsksResponse_InternalServerError, getInternalErrorMessage(err))
	}

	//Convert to proto
	resp.MoreExists = moreExists
	for _, ask := range myClosedAskOrders {
		resp.ClosedAskOrders = append(resp.ClosedAskOrders, ask.ToProto())
	}

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalActionService) GetMyClosedBids(ctx context.Context, req *actions_pb.GetMyClosedBidsRequest) (*actions_pb.GetMyClosedBidsResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetMyClosedBids",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	l.Infof("GetMyClosedBids requested")

	resp := &actions_pb.GetMyClosedBidsResponse{}
	makeError := func(st actions_pb.GetMyClosedBidsResponse_StatusCode, msg string) (*actions_pb.GetMyClosedBidsResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	userId := getUserId(ctx)
	lastId := req.LastOrderId
	count := req.Count

	moreExists, myClosedBidOrders, err := models.GetMyClosedBids(userId, lastId, count)
	if err != nil {
		l.Errorf("Request failed due to: %+v", err)
		return makeError(actions_pb.GetMyClosedBidsResponse_InternalServerError, getInternalErrorMessage(err))
	}

	//Convert to proto
	resp.MoreExists = moreExists
	for _, bid := range myClosedBidOrders {
		resp.ClosedBidOrders = append(resp.ClosedBidOrders, bid.ToProto())
	}

	l.Infof("Request completed successfully")

	return resp, nil
}
