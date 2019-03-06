package actionservice

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"golang.org/x/net/context"

	"github.com/delta/dalal-street-server/matchingengine"
	"github.com/delta/dalal-street-server/models"
	"github.com/delta/dalal-street-server/session"

	pb "github.com/delta/dalal-street-server/proto_build"
	actions_pb "github.com/delta/dalal-street-server/proto_build/actions"
	models_pb "github.com/delta/dalal-street-server/proto_build/models"

	"github.com/delta/dalal-street-server/utils"
)

var logger *logrus.Entry

func getInternalErrorMessage(err error) string {
	if utils.IsProdEnv() {
		return "Oops! Something went wrong. Please try again in some time."
	}
	return err.Error()
}

func getUserId(ctx context.Context) uint32 {
	sess := ctx.Value("session").(session.Session)
	userId, _ := sess.Get("userId")
	userIdInt, _ := strconv.ParseUint(userId, 10, 32)
	return uint32(userIdInt)
}

type dalalActionService struct {
	matchingEngine matchingengine.MatchingEngine
}

// NewDalalActionService returns instance of DalalActionServiceServer
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
	makeError := func(st actions_pb.BuyStocksFromExchangeResponse_StatusCode, msg string) (*actions_pb.BuyStocksFromExchangeResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	if !models.IsMarketOpen() {
		return makeError(actions_pb.BuyStocksFromExchangeResponse_MarketClosedError, "Market is currently closed. You cannot buy stocks right now.")
	}

	userId := getUserId(ctx)
	stockId := req.StockId
	stockQty := req.StockQuantity

	transaction, err := models.PerformBuyFromExchangeTransaction(userId, stockId, stockQty)

	switch e := err.(type) {
	case models.BuyLimitExceededError:
		return makeError(actions_pb.BuyStocksFromExchangeResponse_BuyLimitExceededError, e.Error())
	case models.NotEnoughCashError:
		return makeError(actions_pb.BuyStocksFromExchangeResponse_NotEnoughCashError, e.Error())
	case models.NotEnoughStocksError:
		return makeError(actions_pb.BuyStocksFromExchangeResponse_NotEnoughStocksError, e.Error())
	}

	if err != nil {
		l.Errorf("Request failed due to: %+v", err)
		return makeError(actions_pb.BuyStocksFromExchangeResponse_InternalServerError, getInternalErrorMessage(err))
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
	makeError := func(st actions_pb.CancelOrderResponse_StatusCode, msg string) (*actions_pb.CancelOrderResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	if !models.IsMarketOpen() {
		return makeError(actions_pb.CancelOrderResponse_MarketClosedError, "Market is closed. You cannot cancel orders right now.")
	}

	userId := getUserId(ctx)
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
		return makeError(actions_pb.CreateBotResponse_InternalServerError, getInternalErrorMessage(err))
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
		l.Errorf("Request failed. User for Id does not exist. Error: %+v", err)
		if utils.IsProdEnv() {
			return makeError(actions_pb.GetPortfolioResponse_InvalidCredentialsError, "Invalid credentials given")
		}
		return makeError(actions_pb.GetPortfolioResponse_InvalidCredentialsError, fmt.Sprintf("User for ID does not exist: %+v", err))
	}

	stocksOwned, err := models.GetStocksOwned(user.Id)
	if err != nil {
		l.Errorf("Unable to get Stocks for User Id. Error: %+v", err)
		return makeError(actions_pb.GetPortfolioResponse_InternalServerError, "")
	}

	reservedStocksOwned, err := models.GetReservedStocksOwned(user.Id)
	if err != nil {
		l.Errorf("Unable to get Reserved Stocks for User Id. Error: %+v", err)
		return makeError(actions_pb.GetPortfolioResponse_InternalServerError, "")
	}

	resp.SessionId = sess.GetID()
	resp.User = user.ToProto()
	resp.StocksOwned = stocksOwned
	resp.ReservedStocksOwned = reservedStocksOwned

	return resp, nil
}

func writeUserDetailsToLog(ctx context.Context) {
	var l = logger.WithFields(logrus.Fields{
		"method": "writeUserDetailsToLog",
	})

	userID := getUserId(ctx)

	peerDetails, ok := peer.FromContext(ctx)
	if ok {
		err := models.AddToGeneralLog(userID, "IP", peerDetails.Addr.String())
		if err != nil {
			l.Infof("Error while writing to databaes. Error: %+v", err)
		}
	} else {
		l.Infof("Failed to log peer details")
	}

	mD, ok := metadata.FromIncomingContext(ctx)
	if ok {
		userAgent := strings.Join(mD["user-agent"], " ")
		err := models.AddToGeneralLog(userID, "User-Agent", userAgent)
		if err != nil {
			l.Infof("Error while writing to databaes. Error: %+v", err)
		}
	} else {
		l.Infof("Failed to log user-agent")
	}
}

func (d *dalalActionService) Register(ctx context.Context, req *actions_pb.RegisterRequest) (*actions_pb.RegisterResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "Register",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	l.Infof("Register requested")

	resp := &actions_pb.RegisterResponse{}
	makeError := func(st actions_pb.RegisterResponse_StatusCode, msg string) (*actions_pb.RegisterResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	err := models.RegisterUser(req.GetEmail(), req.GetPassword(), req.GetFullName())
	if err == models.AlreadyRegisteredError {
		return makeError(actions_pb.RegisterResponse_AlreadyRegisteredError, "Already registered please Login")
	}
	if err != nil {
		l.Errorf("Request failed due to: %+v", err)
		return makeError(actions_pb.RegisterResponse_InternalServerError, getInternalErrorMessage(err))
	}

	l.Infof("Done")

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
			return makeError(actions_pb.LoginResponse_InvalidCredentialsError, "Invalid Credentials")
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
		l.Errorf("Request failed due to: %+v", err)
		return makeError(actions_pb.LoginResponse_InternalServerError, getInternalErrorMessage(err))
	}

	l.Debugf("models.Login returned without error %+v", user)

	if !alreadyLoggedIn {
		if err := sess.Set("userId", strconv.Itoa(int(user.Id))); err != nil {
			l.Errorf("Request failed due to: %+v", err)
			return makeError(actions_pb.LoginResponse_InternalServerError, getInternalErrorMessage(err))
		}
	}

	writeUserDetailsToLog(ctx)

	l.Debugf("Session successfully set. UserId: %+v, Session id: %+v", user.Id, sess.GetID())

	stocksOwned, err := models.GetStocksOwned(user.Id)
	if err != nil {
		l.Errorf("Request failed due to %+v", err)
		return makeError(actions_pb.LoginResponse_InternalServerError, getInternalErrorMessage(err))
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
		"ORDER_PRICE_WINDOW":      models.ORDER_PRICE_WINDOW,
		"STARTING_CASH":           models.STARTING_CASH,
		"MORTGAGE_RETRIEVE_RATE":  models.MORTGAGE_RETRIEVE_RATE,
		"MORTGAGE_DEPOSIT_RATE":   models.MORTGAGE_DEPOSIT_RATE,
		"MARKET_EVENT_COUNT":      models.MARKET_EVENT_COUNT,
		"MY_ASK_COUNT":            models.MY_ASK_COUNT,
		"MY_BID_COUNT":            models.MY_BID_COUNT,
		"GET_NOTIFICATION_COUNT":  models.GET_NOTIFICATION_COUNT,
		"GET_TRANSACTION_COUNT":   models.GET_TRANSACTION_COUNT,
		"LEADERBOARD_COUNT":       models.LEADERBOARD_COUNT,
		"ORDER_FEE_PERCENT":       models.ORDER_FEE_PERCENT,
	}

	reservedStocksOwned, err := models.GetReservedStocksOwned(user.Id)
	if err != nil {
		l.Errorf("Unable to get Reserved Stocks for User Id. Error: %+v", err)
		return makeError(actions_pb.LoginResponse_InternalServerError, getInternalErrorMessage(err))
	}

	resp = &actions_pb.LoginResponse{
		SessionId:                sess.GetID(),
		User:                     user.ToProto(),
		StocksOwned:              stocksOwned,
		StockList:                stockListProto,
		Constants:                constantsMap,
		IsMarketOpen:             models.IsMarketOpen(),
		MarketIsClosedHackyNotif: models.MARKET_IS_CLOSED_HACKY_NOTIF,
		MarketIsOpenHackyNotif:   models.MARKET_IS_OPEN_HACKY_NOTIF,
		ReservedStocksOwned:      reservedStocksOwned,
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
	userId := getUserId(ctx)
	models.Logout(userId)
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
		return makeError(actions_pb.MortgageStocksResponse_MarketClosedError, "Market is closed. You cannot mortgage stocks right now.")
	}

	userId := getUserId(ctx)
	stockId := req.StockId
	stockQty := -int64(req.StockQuantity)

	transaction, err := models.PerformMortgageTransaction(userId, stockId, stockQty, 0)

	switch e := err.(type) {
	case models.NotEnoughStocksError:
		return makeError(actions_pb.MortgageStocksResponse_NotEnoughStocksError, e.Error())
	case models.WayTooMuchCashError:
		return makeError(actions_pb.MortgageStocksResponse_NotEnoughStocksError, e.Error())
	}
	if err != nil {
		l.Errorf("Request failed due to: %+v", err)
		return makeError(actions_pb.MortgageStocksResponse_InternalServerError, getInternalErrorMessage(err))
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
		return makeError(actions_pb.PlaceOrderResponse_MarketClosedError, "Market Is closed. You cannot place orders right now.")
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
	case models.OrderStockLimitExceeded:
		return makeError(actions_pb.PlaceOrderResponse_StockQuantityLimitExceeded, e.Error())
	case models.OrderPriceOutOfWindowError: // should update proto as well ideally. This is how it is for now.
		return makeError(actions_pb.PlaceOrderResponse_StockQuantityLimitExceeded, e.Error())
	case models.NotEnoughStocksError:
		return makeError(actions_pb.PlaceOrderResponse_NotEnoughStocksError, e.Error())
	case models.NotEnoughCashError:
		return makeError(actions_pb.PlaceOrderResponse_NotEnoughCashError, e.Error())
	}

	if err != nil {
		l.Errorf("Request failed due to: %+v", err)
		return makeError(actions_pb.PlaceOrderResponse_InternalServerError, getInternalErrorMessage(err))
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
		return makeError(actions_pb.RetrieveMortgageStocksResponse_MarketClosedError, "Market is closed. You cannot retrieve your mortgaged stocks right now.")
	}

	userID := getUserId(ctx)
	stockID := req.StockId
	stockQty := int64(req.StockQuantity)
	retrievePrice := req.RetrievePrice

	transaction, err := models.PerformMortgageTransaction(userID, stockID, stockQty, retrievePrice)

	switch e := err.(type) {
	case models.NotEnoughStocksError:
		return makeError(actions_pb.RetrieveMortgageStocksResponse_NotEnoughStocksError, e.Error())
	case models.NotEnoughCashError:
		return makeError(actions_pb.RetrieveMortgageStocksResponse_NotEnoughCashError, e.Error())
	case models.InvalidRetrievePriceError:
		return makeError(actions_pb.RetrieveMortgageStocksResponse_InvalidRetrievePriceError, e.Error())
	}
	if err != nil {
		l.Errorf("Request failed due to %+v: ", err)
		return makeError(actions_pb.RetrieveMortgageStocksResponse_InternalServerError, getInternalErrorMessage(err))
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
		l.Errorf("Request failed due to: %+v", err)
		resp.StatusCode = actions_pb.GetCompanyProfileResponse_InternalServerError
		resp.StatusMessage = getInternalErrorMessage(err)
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
		l.Errorf("Request failed due to: %+v", err)
		resp.StatusCode = actions_pb.GetStockHistoryResponse_InternalServerError
		resp.StatusMessage = getInternalErrorMessage(err)
		return resp, nil
	}

	resp.StockHistoryMap = make(map[string]*models_pb.StockHistory)

	for _, stockData := range stockHistory {
		resp.StockHistoryMap[stockData.CreatedAt] = stockData.ToProto()
	}

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
		l.Errorf("Request failed due to: %+v", err)
		resp.StatusCode = actions_pb.GetMarketEventsResponse_InternalServerError
		resp.StatusMessage = getInternalErrorMessage(err)
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
		l.Errorf("Request failed due to: %+v", err)
		resp.StatusCode = actions_pb.GetNotificationsResponse_InternalServerError
		resp.StatusMessage = getInternalErrorMessage(err)
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
		l.Errorf("Request failed due to: %+v", err)
		resp.StatusCode = actions_pb.GetTransactionsResponse_InternalServerError
		resp.StatusMessage = getInternalErrorMessage(err)
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
		l.Errorf("Request failed due to: %+v", err)
		resp.StatusCode = actions_pb.GetMortgageDetailsResponse_InternalServerError
		resp.StatusMessage = getInternalErrorMessage(err)
		return resp, nil
	}

	for _, mortgageEntry := range mortgages {
		resp.MortgageDetails = append(resp.MortgageDetails, mortgageEntry.ToProto())
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
		l.Errorf("Request failed due to: %+v", err)
		resp.StatusCode = actions_pb.GetLeaderboardResponse_InternalServerError
		resp.StatusMessage = getInternalErrorMessage(err)
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
