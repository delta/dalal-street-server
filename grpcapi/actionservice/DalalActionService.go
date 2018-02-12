package actionservice

import (
	"fmt"
	"strconv"

	"github.com/Sirupsen/logrus"

	"golang.org/x/net/context"

	"github.com/thakkarparth007/dalal-street-server/matchingengine"
	"github.com/thakkarparth007/dalal-street-server/models"
	"github.com/thakkarparth007/dalal-street-server/session"

	"github.com/thakkarparth007/dalal-street-server/proto_build"
	"github.com/thakkarparth007/dalal-street-server/proto_build/actions"
	"github.com/thakkarparth007/dalal-street-server/proto_build/models"

	"github.com/thakkarparth007/dalal-street-server/utils"
)

var logger *logrus.Entry

func getUserId(ctx context.Context) uint32 {
	sess := ctx.Value("session").(session.Session)
	userId, _ := sess.Get("userId")
	userIdInt, _ := strconv.ParseUint(userId, 10, 32)
	return uint32(userIdInt)
}

type dalalActionService struct {
	matchingEngine matchingengine.MatchingEngine
}

func NewDalalActionService(me matchingengine.MatchingEngine) pb.DalalActionServiceServer {
	logger = utils.Logger.WithFields(logrus.Fields{
		"module": "grpcapi.actions",
	})

	return &dalalActionService{
		matchingEngine: me,
	}
}

func (d *dalalActionService) BuyStocksFromExchange(ctx context.Context, req *actions_pb.BuyStocksFromExchangeRequest) (*actions_pb.BuyStocksFromExchangeResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "BuyStocksFromExchange",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("BuyStocksFromExchange requested")

	resp := &actions_pb.BuyStocksFromExchangeResponse{}
	makeError := func(st actions_pb.BuyStocksFromExchangeResponse_StatusCode) (*actions_pb.BuyStocksFromExchangeResponse, error) {
		resp.StatusCode = st
		return resp, nil
	}

	if !models.IsMarketOpen() {
		return makeError(actions_pb.BuyStocksFromExchangeResponse_MarketClosedError)
	}

	userId := getUserId(ctx)
	stockId := req.StockId
	stockQty := req.StockQuantity

	transaction, err := models.PerformBuyFromExchangeTransaction(userId, stockId, stockQty)

	switch err.(type) {
	case models.BuyLimitExceededError:
		return makeError(actions_pb.BuyStocksFromExchangeResponse_BuyLimitExceededError)
	case models.NotEnoughCashError:
		return makeError(actions_pb.BuyStocksFromExchangeResponse_NotEnoughCashError)
	case models.NotEnoughStocksError:
		return makeError(actions_pb.BuyStocksFromExchangeResponse_NotEnoughStocksError)
	}

	if err != nil {
		return makeError(actions_pb.BuyStocksFromExchangeResponse_InternalServerError)
	}

	resp.Transaction = transaction.ToProto()

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalActionService) CancelOrder(ctx context.Context, req *actions_pb.CancelOrderRequest) (*actions_pb.CancelOrderResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "CancelOrder",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("CancelOrder requested")

	resp := &actions_pb.CancelOrderResponse{}
	makeError := func(st actions_pb.CancelOrderResponse_StatusCode) (*actions_pb.CancelOrderResponse, error) {
		resp.StatusCode = st
		return resp, nil
	}

	if !models.IsMarketOpen() {
		return makeError(actions_pb.CancelOrderResponse_MarketClosedError)
	}

	userId := getUserId(ctx)
	orderId := req.OrderId
	isAsk := req.IsAsk

	askOrder, bidOrder, err := models.CancelOrder(userId, orderId, isAsk)

	switch err.(type) {
	case models.InvalidAskIdError:
	case models.InvalidBidIdError:
		return makeError(actions_pb.CancelOrderResponse_InvalidOrderId)
	}

	if err != nil {
		return makeError(actions_pb.CancelOrderResponse_InternalServerError)
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

func (d *dalalActionService) CreateBot(ctx context.Context, req *actions_pb.CreateBotRequest) (*actions_pb.CreateBotResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "CreateBot",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("Creating Bot")

	resp := &actions_pb.CreateBotResponse{}
	makeError := func(st actions_pb.CreateBotResponse_StatusCode, msg string) (*actions_pb.CreateBotResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	user, err := models.CreateBot(req.GetBotUserId())
	if err != nil {
		l.Errorf("Unable to Create bot models.CreateBot threw error %+v", err)
		return makeError(actions_pb.CreateBotResponse_InternalServerError, "")
	}

	resp.User = user.ToProto()

	return resp, nil
}

func (d *dalalActionService) GetPortfolio(ctx context.Context, req *actions_pb.GetPortfolioRequest) (*actions_pb.GetPortfolioResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetPortfolio",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("Getting Portfolio")

	resp := &actions_pb.GetPortfolioResponse{}
	makeError := func(st actions_pb.GetPortfolioResponse_StatusCode, msg string) (*actions_pb.GetPortfolioResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	sess := ctx.Value("session").(session.Session)
	userId := getUserId(ctx)

	user, err := models.GetUserCopy(userId)
	if err != nil {
		l.Errorf("User for Id does not exist. Error: %+v", err)
		return makeError(actions_pb.GetPortfolioResponse_InvalidCredentialsError, "")
	}

	stocksOwned, err := models.GetStocksOwned(user.Id)
	if err != nil {
		l.Errorf("Unable to get Stocks for User Id. Error: %+v", err)
		return makeError(actions_pb.GetPortfolioResponse_InternalServerError, "")
	}

	resp.SessionId = sess.GetId()
	resp.User = user.ToProto()
	resp.StocksOwned = stocksOwned

	return resp, nil
}

func (d *dalalActionService) Login(ctx context.Context, req *actions_pb.LoginRequest) (*actions_pb.LoginResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "Login",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("Login requested")

	resp := &actions_pb.LoginResponse{}
	makeError := func(st actions_pb.LoginResponse_StatusCode, msg string) (*actions_pb.LoginResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	var (
		user            models.User
		err             error
		alreadyLoggedIn bool
	)

	sess := ctx.Value("session").(session.Session)
	if userId, ok := sess.Get("userId"); !ok {
		email := req.GetEmail()
		password := req.GetPassword()

		if email == "" || password == "" {
			return makeError(actions_pb.LoginResponse_InvalidCredentialsError, "")
		}

		user, err = models.Login(email, password)
	} else {
		alreadyLoggedIn = true
		userIdInt, err := strconv.ParseUint(userId, 10, 32)
		if err == nil {
			user, err = models.GetUserCopy(uint32(userIdInt))
		}
	}

	switch {
	case err == models.UnauthorizedError:
		return makeError(actions_pb.LoginResponse_InvalidCredentialsError, "Incorrect username/password combination. Please use your Pragyan credentials.")
	case err == models.NotRegisteredError:
		return makeError(actions_pb.LoginResponse_InvalidCredentialsError, "You have not registered for Dalal Street on the Pragyan website")
	case err != nil:
		l.Errorln(err)
		return makeError(actions_pb.LoginResponse_InternalServerError, "")
	}

	l.Debugf("models.Login returned without error %+v", user)

	if !alreadyLoggedIn {
		if err := sess.Set("userId", strconv.Itoa(int(user.Id))); err != nil {
			l.Errorln(err)
			return makeError(actions_pb.LoginResponse_InternalServerError, "")
		}
	}

	l.Debugf("Session successfully set. UserId: %+v, Session id: %+v", user.Id, sess.GetId())

	stocksOwned, err := models.GetStocksOwned(user.Id)
	if err != nil {
		l.Errorln(err)
		return makeError(actions_pb.LoginResponse_InternalServerError, "")
	}

	stockList := models.GetAllStocks()
	stockListProto := make(map[uint32]*models_pb.Stock)
	for stockId, stock := range stockList {
		stockListProto[stockId] = stock.ToProto()
	}

	constantsMap := map[string]int32{
		"SHORT_SELL_BORROW_LIMIT": models.SHORT_SELL_BORROW_LIMIT,
		"BID_LIMIT":               models.BID_LIMIT,
		"ASK_LIMIT":               models.ASK_LIMIT,
		"BUY_LIMIT":               models.BUY_LIMIT,
		"MINIMUM_CASH_LIMIT":      models.MINIMUM_CASH_LIMIT,
		"BUY_FROM_EXCHANGE_LIMIT": models.BUY_FROM_EXCHANGE_LIMIT,
		"STARTING_CASH":           models.STARTING_CASH,
		"MORTGAGE_RETRIEVE_RATE":  models.MORTGAGE_RETRIEVE_RATE,
		"MORTGAGE_DEPOSIT_RATE":   models.MORTGAGE_DEPOSIT_RATE,
		"MARKET_EVENT_COUNT":      models.MARKET_EVENT_COUNT,
		"MY_ASK_COUNT":            models.MY_ASK_COUNT,
		"MY_BID_COUNT":            models.MY_BID_COUNT,
		"GET_NOTIFICATION_COUNT":  models.GET_NOTIFICATION_COUNT,
		"GET_TRANSACTION_COUNT":   models.GET_TRANSACTION_COUNT,
		"LEADERBOARD_COUNT":       models.LEADERBOARD_COUNT,
	}

	resp = &actions_pb.LoginResponse{
		SessionId:                sess.GetId(),
		User:                     user.ToProto(),
		StocksOwned:              stocksOwned,
		StockList:                stockListProto,
		Constants:                constantsMap,
		IsMarketOpen:             models.IsMarketOpen(),
		MarketIsClosedHackyNotif: models.MARKET_IS_CLOSED_HACKY_NOTIF,
		MarketIsOpenHackyNotif:   models.MARKET_IS_OPEN_HACKY_NOTIF,
	}

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalActionService) Logout(ctx context.Context, req *actions_pb.LogoutRequest) (*actions_pb.LogoutResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "Logout",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("Logout requested")

	sess := ctx.Value("session").(session.Session)
	sess.Destroy()

	l.Infof("Request completed successfully")

	return &actions_pb.LogoutResponse{}, nil
}

func (d *dalalActionService) MortgageStocks(ctx context.Context, req *actions_pb.MortgageStocksRequest) (*actions_pb.MortgageStocksResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "MortgageStocks",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("MortgageStocks requested")

	resp := &actions_pb.MortgageStocksResponse{}
	makeError := func(st actions_pb.MortgageStocksResponse_StatusCode, msg string) (*actions_pb.MortgageStocksResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	if !models.IsMarketOpen() {
		return makeError(actions_pb.MortgageStocksResponse_MarketClosedError, "")
	}

	userId := getUserId(ctx)
	stockId := req.StockId
	stockQty := -int32(req.StockQuantity)

	transaction, err := models.PerformMortgageTransaction(userId, stockId, stockQty)

	switch e := err.(type) {
	case models.NotEnoughStocksError:
		return makeError(actions_pb.MortgageStocksResponse_NotEnoughStocksError, e.Error())
	}
	if err != nil {
		return makeError(actions_pb.MortgageStocksResponse_InternalServerError, "")
	}

	resp.Transaction = transaction.ToProto()

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
		return makeError(actions_pb.PlaceOrderResponse_MarketClosedError, "")
	}

	userId := getUserId(ctx)
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
	case models.AskLimitExceededError:
	case models.BidLimitExceededError:
		return makeError(actions_pb.PlaceOrderResponse_StockQuantityLimitExceeded, e.Error())
	case models.NotEnoughStocksError:
		return makeError(actions_pb.PlaceOrderResponse_NotEnoughStocksError, e.Error())
	case models.NotEnoughCashError:
		return makeError(actions_pb.PlaceOrderResponse_NotEnoughCashError, e.Error())
	}

	if err != nil {
		return makeError(actions_pb.PlaceOrderResponse_InternalServerError, "")
	}

	resp.OrderId = orderId

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalActionService) RetrieveMortgageStocks(ctx context.Context, req *actions_pb.RetrieveMortgageStocksRequest) (*actions_pb.RetrieveMortgageStocksResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "RetrieveMortgageStocks",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("RetrieveMortgageStocks requested")

	resp := &actions_pb.RetrieveMortgageStocksResponse{}
	makeError := func(st actions_pb.RetrieveMortgageStocksResponse_StatusCode, msg string) (*actions_pb.RetrieveMortgageStocksResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	if !models.IsMarketOpen() {
		return makeError(actions_pb.RetrieveMortgageStocksResponse_MarketClosedError, "")
	}

	userId := getUserId(ctx)
	stockId := req.StockId
	stockQty := int32(req.StockQuantity)

	transaction, err := models.PerformMortgageTransaction(userId, stockId, stockQty)

	switch e := err.(type) {
	case models.NotEnoughStocksError:
		return makeError(actions_pb.RetrieveMortgageStocksResponse_NotEnoughStocksError, e.Error())
	case models.NotEnoughCashError:
		return makeError(actions_pb.RetrieveMortgageStocksResponse_NotEnoughCashError, e.Error())
	}
	if err != nil {
		return makeError(actions_pb.RetrieveMortgageStocksResponse_InternalServerError, "")
	}

	resp.Transaction = transaction.ToProto()

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalActionService) GetCompanyProfile(ctx context.Context, req *actions_pb.GetCompanyProfileRequest) (*actions_pb.GetCompanyProfileResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetCompanyProfile",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetCompanyProfile requested")

	resp := &actions_pb.GetCompanyProfileResponse{}

	stockDetails, err := models.GetCompanyDetails(req.StockId)
	if err != nil {
		resp.StatusCode = actions_pb.GetCompanyProfileResponse_InternalServerError
		return resp, nil
	}

	resp.StockDetails = stockDetails.ToProto()

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalActionService) GetStockHistory(ctx context.Context, req *actions_pb.GetStockHistoryRequest) (*actions_pb.GetStockHistoryResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetStockHistory",
		"param_session": fmt.Sprintf("%v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%v", req),
	})
	l.Infof("Getting StockHistory")

	resp := &actions_pb.GetStockHistoryResponse{}

	stockHistory, err := models.GetStockHistory(req.StockId, models.ResolutionFromProto(req.GetResolution())) // Check if this works

	if err != nil {
		resp.StatusCode = actions_pb.GetStockHistoryResponse_InternalServerError
		return resp, nil
	}

	stockHistoryMap := make(map[string]*models_pb.StockHistory)

	for _, stockData := range stockHistory {
		stockHistoryMap[stockData.CreatedAt] = stockData.ToProto()
	}

	resp.StockHistoryMap = stockHistoryMap

	l.Infof("StockHistory Returned")

	return resp, nil
}

func (d *dalalActionService) GetMarketEvents(ctx context.Context, req *actions_pb.GetMarketEventsRequest) (*actions_pb.GetMarketEventsResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetMarketEvents",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetMarketEvents requested")

	resp := &actions_pb.GetMarketEventsResponse{}

	lastId := req.LastEventId
	count := req.Count

	moreExists, marketEvents, err := models.GetMarketEvents(lastId, count)
	if err != nil {
		resp.StatusCode = actions_pb.GetMarketEventsResponse_InternalServerError
		return resp, nil
	}

	resp.MoreExists = moreExists
	for _, marketEvent := range marketEvents {
		resp.MarketEvents = append(resp.MarketEvents, marketEvent.ToProto())
	}

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
		return makeError(actions_pb.GetMyOpenOrdersResponse_InternalServerError, "")
	}

	//convert open ask orders to proto
	var myOpenAskOrdersProto []*models_pb.Ask
	for _, ask := range myOpenAskOrders {
		myOpenAskOrdersProto = append(myOpenAskOrdersProto, ask.ToProto())
	}

	//get open bid orders
	myOpenBidOrders, err := models.GetMyOpenBids(userId)

	if err != nil {
		return makeError(actions_pb.GetMyOpenOrdersResponse_InternalServerError, "")
	}

	//convert open bid orders to proto
	var myOpenBidOrdersProto []*models_pb.Bid

	for _, bid := range myOpenBidOrders {
		myOpenBidOrdersProto = append(myOpenBidOrdersProto, bid.ToProto())
	}

	resp.OpenAskOrders = myOpenAskOrdersProto
	resp.OpenBidOrders = myOpenBidOrdersProto

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
		return makeError(actions_pb.GetMyClosedAsksResponse_InternalServerError, "")
	}

	//Convert to proto
	var myClosedAskOrdersProto []*models_pb.Ask

	for _, ask := range myClosedAskOrders {
		myClosedAskOrdersProto = append(myClosedAskOrdersProto, ask.ToProto())
	}

	resp.MoreExists = moreExists
	resp.ClosedAskOrders = myClosedAskOrdersProto

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
		return makeError(actions_pb.GetMyClosedBidsResponse_InternalServerError, "")
	}

	//Convert to proto
	var myClosedBidOrdersProto []*models_pb.Bid

	for _, bid := range myClosedBidOrders {
		myClosedBidOrdersProto = append(myClosedBidOrdersProto, bid.ToProto())
	}

	resp.MoreExists = moreExists
	resp.ClosedBidOrders = myClosedBidOrdersProto

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalActionService) GetNotifications(ctx context.Context, req *actions_pb.GetNotificationsRequest) (*actions_pb.GetNotificationsResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetNotifications",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetNotifications requested")

	resp := &actions_pb.GetNotificationsResponse{}

	lastId := req.LastNotificationId
	count := req.Count

	moreExists, notifications, err := models.GetNotifications(getUserId(ctx), lastId, count)
	if err != nil {
		resp.StatusCode = actions_pb.GetNotificationsResponse_InternalServerError
		return resp, nil
	}

	//Convert to proto
	resp.MoreExists = moreExists
	for _, notification := range notifications {
		resp.Notifications = append(resp.Notifications, notification.ToProto())
	}

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalActionService) GetTransactions(ctx context.Context, req *actions_pb.GetTransactionsRequest) (*actions_pb.GetTransactionsResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetTransactions",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetTransactions requested")

	resp := &actions_pb.GetTransactionsResponse{}

	userId := getUserId(ctx)
	lastId := req.LastTransactionId
	count := req.Count

	moreExists, transactions, err := models.GetTransactions(userId, lastId, count)
	if err != nil {
		resp.StatusCode = actions_pb.GetTransactionsResponse_InternalServerError
		return resp, nil
	}

	//Convert to proto
	resp.MoreExists = moreExists
	for _, transaction := range transactions {
		resp.Transactions = append(resp.Transactions, transaction.ToProto())
	}

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalActionService) GetMortgageDetails(ctx context.Context, req *actions_pb.GetMortgageDetailsRequest) (*actions_pb.GetMortgageDetailsResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetMortgageDetails",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetMortgageDetails requested")

	resp := &actions_pb.GetMortgageDetailsResponse{}

	userId := getUserId(ctx)
	mortgages, err := models.GetMortgageDetails(userId)
	if err != nil {
		resp.StatusCode = actions_pb.GetMortgageDetailsResponse_InternalServerError
		return resp, nil
	}

	resp.MortgageMap = make(map[uint32]uint32)
	for _, mortgageDetails := range mortgages {
		resp.MortgageMap[mortgageDetails.StockId] = uint32(-mortgageDetails.StocksInBank)
	}

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalActionService) GetLeaderboard(ctx context.Context, req *actions_pb.GetLeaderboardRequest) (*actions_pb.GetLeaderboardResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetLeaderboard",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetLeaderboard requested")

	resp := &actions_pb.GetLeaderboardResponse{}

	userId := getUserId(ctx)
	startingId := req.StartingId
	count := req.Count

	leaderboard, currentUserLeaderboard, totalUsers, err := models.GetLeaderboard(userId, startingId, count)
	if err != nil {
		resp.StatusCode = actions_pb.GetLeaderboardResponse_InternalServerError
		return resp, nil
	}

	resp.MyRank = currentUserLeaderboard.Rank
	resp.TotalUsers = totalUsers
	for _, leaderboardEntry := range leaderboard {
		resp.RankList = append(resp.RankList, leaderboardEntry.ToProto())
	}

	l.Infof("Request completed successfully")

	return resp, nil
}
