package actions

import (
	"fmt"
	"strconv"

	"github.com/Sirupsen/logrus"

	"github.com/thakkarparth007/dalal-street-server/models"
	"github.com/thakkarparth007/dalal-street-server/session"
	"github.com/thakkarparth007/dalal-street-server/socketapi/datastreams"
	actions_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/actions"
	datastreams_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/datastreams"
	errors_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/errors"
	models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

var logger *logrus.Entry

func getUserId(sess session.Session) uint32 {
	userId, _ := sess.Get("userId")
	userIdInt, _ := strconv.ParseUint(userId, 10, 32)
	return uint32(userIdInt)
}

func InitActions() {
	logger = utils.Logger.WithFields(logrus.Fields{
		"module": "socketapi.actions",
	})
}

func BuyStocksFromExchange(sess session.Session, req *actions_proto.BuyStocksFromExchangeRequest) *actions_proto.BuyStocksFromExchangeResponse {
	var l = logger.WithFields(logrus.Fields{
		"method":        "BuyStocksFromExchange",
		"param_session": fmt.Sprintf("%+v", sess),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("BuyStocksFromExchange requested")

	resp := &actions_proto.BuyStocksFromExchangeResponse{}

	// Helpers
	var notEnoughStocksError = func(reason string) *actions_proto.BuyStocksFromExchangeResponse {
		resp.Response = &actions_proto.BuyStocksFromExchangeResponse_NotEnoughStocksError_{
			&actions_proto.BuyStocksFromExchangeResponse_NotEnoughStocksError{
				reason,
			},
		}
		return resp
	}
	var buyLimitExceedeedError = func(reason string) *actions_proto.BuyStocksFromExchangeResponse {
		resp.Response = &actions_proto.BuyStocksFromExchangeResponse_BuyLimitExceededError_{
			&actions_proto.BuyStocksFromExchangeResponse_BuyLimitExceededError{
				reason,
			},
		}
		return resp
	}
	var notEnoughCashError = func(reason string) *actions_proto.BuyStocksFromExchangeResponse {
		resp.Response = &actions_proto.BuyStocksFromExchangeResponse_NotEnoughCashError_{
			&actions_proto.BuyStocksFromExchangeResponse_NotEnoughCashError{
				reason,
			},
		}
		return resp
	}
	var badRequestError = func(reason string) *actions_proto.BuyStocksFromExchangeResponse {
		resp.Response = &actions_proto.BuyStocksFromExchangeResponse_BadRequestError{
			&errors_proto.BadRequestError{
				reason,
			},
		}
		return resp
	}
	var internalServerError = func(err error) *actions_proto.BuyStocksFromExchangeResponse {
		l.Errorf("Internal server error: '%+v'", err)
		resp.Response = &actions_proto.BuyStocksFromExchangeResponse_InternalServerError{
			&errors_proto.InternalServerError{
				"We are facing some issues on the server. Please try again in some time.",
			},
		}
		return resp
	}

	//Check for positive stock quantity
	if req.StockQuantity <= 0 {
		return badRequestError("Invalid Stock Quantity.")
	}

	userId := getUserId(sess)
	stockId := req.StockId
	stockQty := req.StockQuantity

	transaction, err := models.PerformBuyFromExchangeTransaction(userId, stockId, stockQty)

	switch e := err.(type) {
	case models.BuyLimitExceededError:
		return buyLimitExceedeedError(e.Error())
	case models.NotEnoughCashError:
		return notEnoughCashError(e.Error())
	case models.NotEnoughStocksError:
		return notEnoughStocksError(e.Error())
	}

	if err != nil {
		return internalServerError(err)
	}

	resp.Response = &actions_proto.BuyStocksFromExchangeResponse_Result{
		&actions_proto.BuyStocksFromExchangeResponse_BuyStocksFromExchangeSuccessResponse{
			Transaction: transaction.ToProto(),
		},
	}

	l.Infof("Request completed successfully")

	return resp
}

func CancelAskOrder(sess session.Session, req *actions_proto.CancelAskOrderRequest) *actions_proto.CancelAskOrderResponse {
	var l = logger.WithFields(logrus.Fields{
		"method":        "CancelAskOrder",
		"param_session": fmt.Sprintf("%+v", sess),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("CancelAskOrder requested")

	resp := &actions_proto.CancelAskOrderResponse{}

	// Helpers
	var invalidAskIdError = func(reason string) *actions_proto.CancelAskOrderResponse {
		l.Infof("Invalid credentials: '%s'", reason)
		resp.Response = &actions_proto.CancelAskOrderResponse_InvalidAskIdError_{
			&actions_proto.CancelAskOrderResponse_InvalidAskIdError{
				reason,
			},
		}
		return resp
	}
	var internalServerError = func(err error) *actions_proto.CancelAskOrderResponse {
		l.Errorf("Internal server error: '%+v'", err)
		resp.Response = &actions_proto.CancelAskOrderResponse_InternalServerError{
			&errors_proto.InternalServerError{
				"We are facing some issues on the server. Please try again in some time.",
			},
		}
		return resp
	}

	userId := getUserId(sess)
	askId := req.AskId

	err := models.CancelOrder(userId, askId, true)

	switch e := err.(type) {
	case models.InvalidAskIdError:
		return invalidAskIdError(e.Error())
	}

	if err != nil {
		return internalServerError(err)
	}

	resp.Response = &actions_proto.CancelAskOrderResponse_Result{
		&actions_proto.CancelAskOrderResponse_CancelAskOrderSuccessResponse{
			Success: true,
		},
	}

	l.Infof("Request completed successfully")

	return resp
}

func CancelBidOrder(sess session.Session, req *actions_proto.CancelBidOrderRequest) *actions_proto.CancelBidOrderResponse {
	var l = logger.WithFields(logrus.Fields{
		"method":        "CancelBidOrder",
		"param_session": fmt.Sprintf("%+v", sess),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("CancelBidOrder requested")

	resp := &actions_proto.CancelBidOrderResponse{}

	// Helpers
	var invalidBidIdError = func(reason string) *actions_proto.CancelBidOrderResponse {
		l.Infof("Invalid credentials: '%s'", reason)
		resp.Response = &actions_proto.CancelBidOrderResponse_InvalidBidIdError_{
			&actions_proto.CancelBidOrderResponse_InvalidBidIdError{
				reason,
			},
		}
		return resp
	}
	var internalServerError = func(err error) *actions_proto.CancelBidOrderResponse {
		l.Errorf("Internal server error: '%+v'", err)
		resp.Response = &actions_proto.CancelBidOrderResponse_InternalServerError{
			&errors_proto.InternalServerError{
				"We are facing some issues on the server. Please try again in some time.",
			},
		}
		return resp
	}

	userId := getUserId(sess)
	bidId := req.BidId

	err := models.CancelOrder(userId, bidId, false)

	switch e := err.(type) {
	case models.InvalidBidIdError:
		return invalidBidIdError(e.Error())
	}

	if err != nil {
		return internalServerError(err)
	}

	resp.Response = &actions_proto.CancelBidOrderResponse_Result{
		&actions_proto.CancelBidOrderResponse_CancelBidOrderSuccessResponse{
			Success: true,
		},
	}

	l.Infof("Request completed successfully")

	return resp
}

func Login(sess session.Session, req *actions_proto.LoginRequest) *actions_proto.LoginResponse {
	var l = logger.WithFields(logrus.Fields{
		"method":        "Login",
		"param_session": fmt.Sprintf("%+v", sess),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("Login requested")

	resp := &actions_proto.LoginResponse{}

	// Helpers
	var invalidCredentialsError = func(reason string) *actions_proto.LoginResponse {
		l.Infof("Invalid credentials: '%s'", reason)
		resp.Response = &actions_proto.LoginResponse_InvalidCredentialsError_{
			&actions_proto.LoginResponse_InvalidCredentialsError{
				reason,
			},
		}
		return resp
	}
	var internalServerError = func(err error) *actions_proto.LoginResponse {
		l.Errorf("Internal server error: '%+v'", err)
		resp.Response = &actions_proto.LoginResponse_InternalServerError{
			&errors_proto.InternalServerError{
				"We are facing some issues on the server. Please try again in some time.",
			},
		}
		return resp
	}

	var (
		user            models.User
		err             error
		alreadyLoggedIn bool
	)

	if userId, ok := sess.Get("userId"); !ok {
		email := req.GetEmail()
		password := req.GetPassword()

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
		return invalidCredentialsError("Incorrect username/password combination. Please use your Pragyan credentials.")
	case err == models.NotRegisteredError:
		return invalidCredentialsError("You have not registered for Dalal Street on the Pragyan website")
	case err != nil:
		return internalServerError(err)
	}

	l.Debugf("models.Login returned without error %+v", user)

	if !alreadyLoggedIn {
		if err := sess.Set("userId", strconv.Itoa(int(user.Id))); err != nil {
			return internalServerError(err)
		}
	}

	l.Debugf("Session successfully set. UserId: %+v, Session id: %+v", user.Id, sess.GetId())

	stocksOwned, err := models.GetStocksOwned(user.Id)
	if err != nil {
		return internalServerError(err)
	}

	stockList := models.GetAllStocks()
	stockListProto := make(map[uint32]*models_proto.Stock)
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

	resp.Response = &actions_proto.LoginResponse_Result{
		&actions_proto.LoginResponse_LoginSuccessResponse{
			SessionId:   sess.GetId(),
			User:        user.ToProto(),
			StocksOwned: stocksOwned,
			StockList:   stockListProto,
			Constants:   constantsMap,
		},
	}

	l.Infof("Request completed successfully")

	return resp
}

func Logout(sess session.Session, req *actions_proto.LogoutRequest) *actions_proto.LogoutResponse {
	var l = logger.WithFields(logrus.Fields{
		"method":        "Logout",
		"param_session": fmt.Sprintf("%+v", sess),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("Logout requested")

	resp := &actions_proto.LogoutResponse{}

	sess.Destroy()

	resp.Response = &actions_proto.LogoutResponse_Result{
		&actions_proto.LogoutResponse_LogoutSuccessResponse{
			Success: true,
		},
	}

	l.Infof("Request completed successfully")

	return resp
}

func MortgageStocks(sess session.Session, req *actions_proto.MortgageStocksRequest) *actions_proto.MortgageStocksResponse {
	var l = logger.WithFields(logrus.Fields{
		"method":        "MortgageStocks",
		"param_session": fmt.Sprintf("%+v", sess),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("MortgageStocks requested")

	resp := &actions_proto.MortgageStocksResponse{}

	//Helpers
	var notEnoughStocksError = func(reason string) *actions_proto.MortgageStocksResponse {
		resp.Response = &actions_proto.MortgageStocksResponse_NotEnoughStocksError_{
			&actions_proto.MortgageStocksResponse_NotEnoughStocksError{
				reason,
			},
		}
		return resp
	}
	var badRequestError = func(reason string) *actions_proto.MortgageStocksResponse {
		resp.Response = &actions_proto.MortgageStocksResponse_BadRequestError{
			&errors_proto.BadRequestError{
				reason,
			},
		}
		return resp
	}
	var internalServerError = func(err error) *actions_proto.MortgageStocksResponse {
		l.Errorf("Internal server error: '%+v'", err)
		resp.Response = &actions_proto.MortgageStocksResponse_InternalServerError{
			&errors_proto.InternalServerError{
				"We are facing some issues on the server. Please try again in some time.",
			},
		}
		return resp
	}

	//Check for non-positive stock quantity
	if req.StockQuantity <= 0 {
		return badRequestError("Invalid Stock Quantity.")
	}

	userId := getUserId(sess)
	stockId := req.StockId
	stockQty := -int32(req.StockQuantity)

	transaction, err := models.PerformMortgageTransaction(userId, stockId, stockQty)

	switch e := err.(type) {
	case models.NotEnoughStocksError:
		return notEnoughStocksError(e.Error())
	}
	if err != nil {
		return internalServerError(err)
	}

	resp.Response = &actions_proto.MortgageStocksResponse_Result{
		&actions_proto.MortgageStocksResponse_MortgageStocksSuccessResponse{
			Transaction: transaction.ToProto(),
		},
	}

	l.Infof("Request completed successfully")

	return resp
}

func PlaceAskOrder(sess session.Session, req *actions_proto.PlaceAskOrderRequest) *actions_proto.PlaceAskOrderResponse {
	var l = logger.WithFields(logrus.Fields{
		"method":        "PlaceAskOrder",
		"param_session": fmt.Sprintf("%+v", sess),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("PlaceAskOrder requested")

	resp := &actions_proto.PlaceAskOrderResponse{}

	// Helpers
	var askLimitExceedeedError = func(reason string) *actions_proto.PlaceAskOrderResponse {
		resp.Response = &actions_proto.PlaceAskOrderResponse_AskLimitExceededError_{
			&actions_proto.PlaceAskOrderResponse_AskLimitExceededError{
				reason,
			},
		}
		return resp
	}
	var notEnoughStocksError = func(reason string) *actions_proto.PlaceAskOrderResponse {
		resp.Response = &actions_proto.PlaceAskOrderResponse_NotEnoughStocksError_{
			&actions_proto.PlaceAskOrderResponse_NotEnoughStocksError{
				reason,
			},
		}
		return resp
	}
	var badRequestError = func(reason string) *actions_proto.PlaceAskOrderResponse {
		resp.Response = &actions_proto.PlaceAskOrderResponse_BadRequestError{
			&errors_proto.BadRequestError{
				reason,
			},
		}
		return resp
	}
	var internalServerError = func(err error) *actions_proto.PlaceAskOrderResponse {
		l.Errorf("Internal server error: '%+v'", err)
		resp.Response = &actions_proto.PlaceAskOrderResponse_InternalServerError{
			&errors_proto.InternalServerError{
				"We are facing some issues on the server. Please try again in some time.",
			},
		}
		return resp
	}

	//Check for positive stock quantity
	if req.StockQuantity <= 0 {
		return badRequestError("Invalid Stock Quantity.")
	}

	userId := getUserId(sess)
	ask := &models.Ask{
		UserId:        userId,
		StockId:       req.StockId,
		OrderType:     models.OrderTypeFromProto(req.OrderType),
		Price:         req.Price,
		StockQuantity: req.StockQuantity,
	}

	askId, err := models.PlaceAskOrder(userId, ask)

	switch e := err.(type) {
	case models.AskLimitExceededError:
		return askLimitExceedeedError(e.Error())
	case models.NotEnoughStocksError:
		return notEnoughStocksError(e.Error())
	}

	if err != nil {
		return internalServerError(err)
	}

	resp.Response = &actions_proto.PlaceAskOrderResponse_Result{
		&actions_proto.PlaceAskOrderResponse_PlaceAskOrderSuccessResponse{
			AskId: askId,
		},
	}

	l.Infof("Request completed successfully")

	return resp
}

func PlaceBidOrder(sess session.Session, req *actions_proto.PlaceBidOrderRequest) *actions_proto.PlaceBidOrderResponse {
	var l = logger.WithFields(logrus.Fields{
		"method":        "PlaceBidOrder",
		"param_session": fmt.Sprintf("%+v", sess),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("PlaceBidOrder requested")

	resp := &actions_proto.PlaceBidOrderResponse{}

	// Helpers
	var bidLimitExceedeedError = func(reason string) *actions_proto.PlaceBidOrderResponse {
		resp.Response = &actions_proto.PlaceBidOrderResponse_BidLimitExceededError_{
			&actions_proto.PlaceBidOrderResponse_BidLimitExceededError{
				reason,
			},
		}
		return resp
	}
	var notEnoughCashError = func(reason string) *actions_proto.PlaceBidOrderResponse {
		resp.Response = &actions_proto.PlaceBidOrderResponse_NotEnoughCashError_{
			&actions_proto.PlaceBidOrderResponse_NotEnoughCashError{
				reason,
			},
		}
		return resp
	}
	var badRequestError = func(reason string) *actions_proto.PlaceBidOrderResponse {
		resp.Response = &actions_proto.PlaceBidOrderResponse_BadRequestError{
			&errors_proto.BadRequestError{
				reason,
			},
		}
		return resp
	}
	var internalServerError = func(err error) *actions_proto.PlaceBidOrderResponse {
		l.Errorf("Internal server error: '%+v'", err)
		resp.Response = &actions_proto.PlaceBidOrderResponse_InternalServerError{
			&errors_proto.InternalServerError{
				"We are facing some issues on the server. Please try again in some time.",
			},
		}
		return resp
	}

	//Check for positive stock quantity
	if req.StockQuantity <= 0 {
		return badRequestError("Invalid Stock Quantity.")
	}

	userId := getUserId(sess)
	bid := &models.Bid{
		UserId:        userId,
		StockId:       req.StockId,
		OrderType:     models.OrderTypeFromProto(req.OrderType),
		Price:         req.Price,
		StockQuantity: req.StockQuantity,
	}

	bidId, err := models.PlaceBidOrder(userId, bid)

	switch e := err.(type) {
	case models.BidLimitExceededError:
		return bidLimitExceedeedError(e.Error())
	case models.NotEnoughCashError:
		return notEnoughCashError(e.Error())
	}

	if err != nil {
		return internalServerError(err)
	}

	resp.Response = &actions_proto.PlaceBidOrderResponse_Result{
		&actions_proto.PlaceBidOrderResponse_PlaceBidOrderSuccessResponse{
			BidId: bidId,
		},
	}

	l.Infof("Request completed successfully")

	return resp
}

func RetrieveMortgageStocks(sess session.Session, req *actions_proto.RetrieveMortgageStocksRequest) *actions_proto.RetrieveMortgageStocksResponse {
	var l = logger.WithFields(logrus.Fields{
		"method":        "RetrieveMortgageStocks",
		"param_session": fmt.Sprintf("%+v", sess),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("RetrieveMortgageStocks requested")

	resp := &actions_proto.RetrieveMortgageStocksResponse{}

	//Helpers
	var notEnoughStocksError = func(reason string) *actions_proto.RetrieveMortgageStocksResponse {
		resp.Response = &actions_proto.RetrieveMortgageStocksResponse_NotEnoughStocksError_{
			&actions_proto.RetrieveMortgageStocksResponse_NotEnoughStocksError{
				reason,
			},
		}
		return resp
	}
	var notEnoughCashError = func(reason string) *actions_proto.RetrieveMortgageStocksResponse {
		resp.Response = &actions_proto.RetrieveMortgageStocksResponse_NotEnoughCashError_{
			&actions_proto.RetrieveMortgageStocksResponse_NotEnoughCashError{
				reason,
			},
		}
		return resp
	}
	var internalServerError = func(err error) *actions_proto.RetrieveMortgageStocksResponse {
		l.Errorf("Internal server error: '%+v'", err)
		resp.Response = &actions_proto.RetrieveMortgageStocksResponse_InternalServerError{
			&errors_proto.InternalServerError{
				"We are facing some issues on the server. Please try again in some time.",
			},
		}
		return resp
	}

	userId := getUserId(sess)
	stockId := req.StockId
	stockQty := int32(req.StockQuantity)

	transaction, err := models.PerformMortgageTransaction(userId, stockId, stockQty)

	switch e := err.(type) {
	case models.NotEnoughStocksError:
		return notEnoughStocksError(e.Error())
	case models.NotEnoughCashError:
		return notEnoughCashError(e.Error())
	}
	if err != nil {
		return internalServerError(err)
	}

	resp.Response = &actions_proto.RetrieveMortgageStocksResponse_Result{
		&actions_proto.RetrieveMortgageStocksResponse_RetrieveMortgageStocksSuccessResponse{
			Transaction: transaction.ToProto(),
		},
	}

	l.Infof("Request completed successfully")

	return resp
}

func Unsubscribe(sess session.Session, req *actions_proto.UnsubscribeRequest) *actions_proto.UnsubscribeResponse {
	var l = logger.WithFields(logrus.Fields{
		"method":        "Unsubscribe",
		"param_session": fmt.Sprintf("%+v", sess),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("Unsubscribe requested")

	resp := &actions_proto.UnsubscribeResponse{}
	resp.Response = &actions_proto.UnsubscribeResponse_Result{
		&actions_proto.UnsubscribeResponse_UnsubscribeSuccessResponse{
			Success: true,
		},
	}

	var badRequestError = func(err error) *actions_proto.UnsubscribeResponse {
		l.Errorf(err.Error())
		resp.Response = &actions_proto.UnsubscribeResponse_BadRequestError{
			&errors_proto.BadRequestError{
				err.Error(),
			},
		}
		return resp
	}

	switch req.DataStreamType {
	case datastreams_proto.DataStreamType_NOTIFICATIONS:
		datastreams.UnregNotificationsListener(getUserId(sess), sess.GetId())
	case datastreams_proto.DataStreamType_STOCK_PRICES:
		datastreams.UnregStockPricesListener(sess.GetId())
	case datastreams_proto.DataStreamType_STOCK_EXCHANGE:
		datastreams.UnregStockExchangeListener(sess.GetId())
	case datastreams_proto.DataStreamType_MARKET_EVENTS:
		datastreams.UnregMarketEventsListener(sess.GetId())
	case datastreams_proto.DataStreamType_MY_ORDERS:
		datastreams.UnregOrdersListener(getUserId(sess), sess.GetId())
	case datastreams_proto.DataStreamType_TRANSACTIONS:
		datastreams.UnregTransactionsListener(getUserId(sess), sess.GetId())
	default:
		return badRequestError(fmt.Errorf("Invalid datastream id %d", req.DataStreamType))
	}

	l.Infof("Request completed successfully")

	return resp
}

func Subscribe(done <-chan struct{}, updates chan interface{}, sess session.Session, req *actions_proto.SubscribeRequest) *actions_proto.SubscribeResponse {
	var l = logger.WithFields(logrus.Fields{
		"method":        "Subscribe",
		"param_session": fmt.Sprintf("%+v", sess),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("Subscribe requested")

	resp := &actions_proto.SubscribeResponse{}

	var badRequestError = func(err error) *actions_proto.SubscribeResponse {
		l.Errorf(err.Error())
		resp.Response = &actions_proto.SubscribeResponse_BadRequestError{
			&errors_proto.BadRequestError{
				err.Error(),
			},
		}
		return resp
	}

	switch req.DataStreamType {
	case datastreams_proto.DataStreamType_NOTIFICATIONS:
		datastreams.RegNotificationsListener(done, updates, getUserId(sess), sess.GetId())
	case datastreams_proto.DataStreamType_STOCK_PRICES:
		datastreams.RegStockPricesListener(done, updates, sess.GetId())
	case datastreams_proto.DataStreamType_STOCK_EXCHANGE:
		datastreams.RegStockExchangeListener(done, updates, sess.GetId())
	case datastreams_proto.DataStreamType_MARKET_EVENTS:
		datastreams.RegMarketEventsListener(done, updates, sess.GetId())
	case datastreams_proto.DataStreamType_MY_ORDERS:
		datastreams.RegOrdersListener(done, updates, getUserId(sess), sess.GetId())
	case datastreams_proto.DataStreamType_TRANSACTIONS:
		datastreams.RegTransactionsListener(done, updates, getUserId(sess), sess.GetId())
	default:
		return badRequestError(fmt.Errorf("Invalid datastream id %d", req.DataStreamType))
	}

	resp.Response = &actions_proto.SubscribeResponse_Result{
		&actions_proto.SubscribeResponse_SubscribeSuccessResponse{
			Success: true,
		},
	}

	l.Infof("Request completed successfully")

	return resp
}

func GetCompanyProfile(sess session.Session, req *actions_proto.GetCompanyProfileRequest) *actions_proto.GetCompanyProfileResponse {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetCompanyProfile",
		"param_session": fmt.Sprintf("%+v", sess),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetCompanyProfile requested")

	resp := &actions_proto.GetCompanyProfileResponse{}

	//Helpers
	var internalServerError = func(err error) *actions_proto.GetCompanyProfileResponse {
		l.Errorf("Internal server error: '%+v'", err)
		resp.Response = &actions_proto.GetCompanyProfileResponse_InternalServerError{
			&errors_proto.InternalServerError{
				"We are facing some issues on the server. Please try again in some time.",
			},
		}
		return resp
	}

	stockId := req.StockId
	stockDetails, stockHistory, err := models.GetCompanyDetails(stockId)

	if err != nil {
		return internalServerError(err)
	}

	//Convert to proto
	stockHistoryMap := make(map[string]*models_proto.StockHistory)
	for _, stockData := range stockHistory {
		stockHistoryMap[stockData.CreatedAt] = stockData.ToProto()
	}

	resp.Response = &actions_proto.GetCompanyProfileResponse_Result{
		&actions_proto.GetCompanyProfileResponse_GetCompanyProfileSuccessResponse{
			StockDetails:    stockDetails.ToProto(),
			StockHistoryMap: stockHistoryMap,
		},
	}

	l.Infof("Request completed successfully")

	return resp
}

func GetMarketEvents(sess session.Session, req *actions_proto.GetMarketEventsRequest) *actions_proto.GetMarketEventsResponse {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetMarketEvents",
		"param_session": fmt.Sprintf("%+v", sess),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetMarketEvents requested")

	resp := &actions_proto.GetMarketEventsResponse{}

	//Helpers
	var internalServerError = func(err error) *actions_proto.GetMarketEventsResponse {
		l.Errorf("Internal server error: '%+v'", err)
		resp.Response = &actions_proto.GetMarketEventsResponse_InternalServerError{
			&errors_proto.InternalServerError{
				"We are facing some issues on the server. Please try again in some time.",
			},
		}
		return resp
	}

	lastId := req.LastEventId
	count := req.Count

	moreExists, marketEvents, err := models.GetMarketEvents(lastId, count)

	if err != nil {
		return internalServerError(err)
	}

	//Convert to proto
	marketEventsMap := make(map[uint32]*models_proto.MarketEvent)

	for _, marketEvent := range marketEvents {
		marketEventsMap[marketEvent.Id] = marketEvent.ToProto()
	}

	resp.Response = &actions_proto.GetMarketEventsResponse_Result{
		&actions_proto.GetMarketEventsResponse_GetMarketEventsSuccessResponse{
			MarketEvents: marketEventsMap,
			MoreExists:   moreExists,
		},
	}

	l.Infof("Request completed successfully")

	return resp
}

func GetMyAsks(sess session.Session, req *actions_proto.GetMyAsksRequest) *actions_proto.GetMyAsksResponse {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetMyAsks",
		"param_session": fmt.Sprintf("%+v", sess),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetMyAsks requested")

	resp := &actions_proto.GetMyAsksResponse{}

	//Helpers
	var internalServerError = func(err error) *actions_proto.GetMyAsksResponse {
		l.Errorf("Internal server error: '%+v'", err)
		resp.Response = &actions_proto.GetMyAsksResponse_InternalServerError{
			&errors_proto.InternalServerError{
				"We are facing some issues on the server. Please try again in some time.",
			},
		}
		return resp
	}

	userId := getUserId(sess)
	lastId := req.LastAskId
	count := req.Count

	moreExists, myOpenAskOrders, myClosedAskOrders, err := models.GetMyAsks(userId, lastId, count)

	if err != nil {
		return internalServerError(err)
	}

	//Convert to proto
	myOpenAskOrdersMap := make(map[uint32]*models_proto.Ask)

	for _, ask := range myOpenAskOrders {
		myOpenAskOrdersMap[ask.Id] = ask.ToProto()
	}

	myClosedAskOrdersMap := make(map[uint32]*models_proto.Ask)

	for _, ask := range myClosedAskOrders {
		myClosedAskOrdersMap[ask.Id] = ask.ToProto()
	}

	resp.Response = &actions_proto.GetMyAsksResponse_Result{
		&actions_proto.GetMyAsksResponse_GetMyAsksSuccessResponse{
			OpenAskOrders:   myOpenAskOrdersMap,
			ClosedAskOrders: myClosedAskOrdersMap,
			MoreExists:      moreExists,
		},
	}

	l.Infof("Request completed successfully")

	return resp
}

func GetMyBids(sess session.Session, req *actions_proto.GetMyBidsRequest) *actions_proto.GetMyBidsResponse {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetMyBids",
		"param_session": fmt.Sprintf("%+v", sess),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetMyBids requested")

	resp := &actions_proto.GetMyBidsResponse{}

	//Helpers
	var internalServerError = func(err error) *actions_proto.GetMyBidsResponse {
		l.Errorf("Internal server error: '%+v'", err)
		resp.Response = &actions_proto.GetMyBidsResponse_InternalServerError{
			&errors_proto.InternalServerError{
				"We are facing some issues on the server. Please try again in some time.",
			},
		}
		return resp
	}

	userId := getUserId(sess)
	lastId := req.LastBidId
	count := req.Count

	moreExists, myOpenBidOrders, myClosedBidOrders, err := models.GetMyBids(userId, lastId, count)

	if err != nil {
		return internalServerError(err)
	}

	//Convert to proto
	myOpenBidOrdersMap := make(map[uint32]*models_proto.Bid)

	for _, bid := range myOpenBidOrders {
		myOpenBidOrdersMap[bid.Id] = bid.ToProto()
	}

	myClosedBidOrdersMap := make(map[uint32]*models_proto.Bid)

	for _, bid := range myClosedBidOrders {
		myClosedBidOrdersMap[bid.Id] = bid.ToProto()
	}

	resp.Response = &actions_proto.GetMyBidsResponse_Result{
		&actions_proto.GetMyBidsResponse_GetMyBidsSuccessResponse{
			OpenBidOrders:   myOpenBidOrdersMap,
			ClosedBidOrders: myClosedBidOrdersMap,
			MoreExists:      moreExists,
		},
	}

	l.Infof("Request completed successfully")

	return resp
}

func GetNotifications(sess session.Session, req *actions_proto.GetNotificationsRequest) *actions_proto.GetNotificationsResponse {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetNotifications",
		"param_session": fmt.Sprintf("%+v", sess),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetNotifications requested")

	resp := &actions_proto.GetNotificationsResponse{}

	//Helpers
	var internalServerError = func(err error) *actions_proto.GetNotificationsResponse {
		l.Errorf("Internal server error: '%+v'", err)
		resp.Response = &actions_proto.GetNotificationsResponse_InternalServerError{
			&errors_proto.InternalServerError{
				"We are facing some issues on the server. Please try again in some time.",
			},
		}
		return resp
	}

	lastId := req.LastNotificationId
	count := req.Count

	moreExists, notifications, err := models.GetNotifications(getUserId(sess), lastId, count)

	if err != nil {
		return internalServerError(err)
	}

	//Convert to proto
	notificationsMap := make(map[uint32]*models_proto.Notification)

	for _, notification := range notifications {
		notificationsMap[notification.Id] = notification.ToProto()
	}

	resp.Response = &actions_proto.GetNotificationsResponse_Result{
		&actions_proto.GetNotificationsResponse_GetNotificationsSuccessResponse{
			Notifications: notificationsMap,
			MoreExists:    moreExists,
		},
	}

	l.Infof("Request completed successfully")

	return resp
}

func GetTransactions(sess session.Session, req *actions_proto.GetTransactionsRequest) *actions_proto.GetTransactionsResponse {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetTransactions",
		"param_session": fmt.Sprintf("%+v", sess),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetTransactions requested")

	resp := &actions_proto.GetTransactionsResponse{}

	//Helpers
	var internalServerError = func(err error) *actions_proto.GetTransactionsResponse {
		l.Errorf("Internal server error: '%+v'", err)
		resp.Response = &actions_proto.GetTransactionsResponse_InternalServerError{
			&errors_proto.InternalServerError{
				"We are facing some issues on the server. Please try again in some time.",
			},
		}
		return resp
	}

	userId := getUserId(sess)
	lastId := req.LastTransactionId
	count := req.Count

	moreExists, transactions, err := models.GetTransactions(userId, lastId, count)

	if err != nil {
		return internalServerError(err)
	}

	//Convert to proto
	transactionsMap := make(map[uint32]*models_proto.Transaction)

	for _, transaction := range transactions {
		transactionsMap[transaction.Id] = transaction.ToProto()
	}

	resp.Response = &actions_proto.GetTransactionsResponse_Result{
		&actions_proto.GetTransactionsResponse_GetTransactionsSuccessResponse{
			TransactionsMap: transactionsMap,
			MoreExists:      moreExists,
		},
	}

	l.Infof("Request completed successfully")

	return resp
}

func GetMortgageDetails(sess session.Session, req *actions_proto.GetMortgageDetailsRequest) *actions_proto.GetMortgageDetailsResponse {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetMortgageDetails",
		"param_session": fmt.Sprintf("%+v", sess),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetMortgageDetails requested")

	// mortgageMap[1] = &actions_proto.GetMortgageDetailsResponse_GetMortgageDetailsSuccessResponse_MortgageDetails{
	// 	StockId:         1,
	// 	NumStocksInBank: 12,
	// }
	resp := &actions_proto.GetMortgageDetailsResponse{}

	//Helpers
	var internalServerError = func(err error) *actions_proto.GetMortgageDetailsResponse {
		l.Errorf("Internal server error: '%+v'", err)
		resp.Response = &actions_proto.GetMortgageDetailsResponse_InternalServerError{
			&errors_proto.InternalServerError{
				"We are facing some issues on the server. Please try again in some time.",
			},
		}
		return resp
	}

	userId := getUserId(sess)

	mortgageMap := make(map[uint32]*actions_proto.GetMortgageDetailsResponse_GetMortgageDetailsSuccessResponse_MortgageDetails)
	mortgages, err := models.GetMortgageDetails(userId)
	if err != nil {
		return internalServerError(err)
	}

	for _, mortgageDetails := range mortgages {
		mortgageMap[mortgageDetails.StockId] = &actions_proto.GetMortgageDetailsResponse_GetMortgageDetailsSuccessResponse_MortgageDetails{
			StockId:         mortgageDetails.StockId,
			NumStocksInBank: uint32(-mortgageDetails.StocksInBank),
		}
	}

	resp.Response = &actions_proto.GetMortgageDetailsResponse_Result{
		&actions_proto.GetMortgageDetailsResponse_GetMortgageDetailsSuccessResponse{
			MortgageMap: mortgageMap,
		},
	}

	l.Infof("Request completed successfully")

	return resp
}

func GetLeaderboard(sess session.Session, req *actions_proto.GetLeaderboardRequest) *actions_proto.GetLeaderboardResponse {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetLeaderboard",
		"param_session": fmt.Sprintf("%+v", sess),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetLeaderboard requested")

	resp := &actions_proto.GetLeaderboardResponse{}

	//Helpers
	var internalServerError = func(err error) *actions_proto.GetLeaderboardResponse {
		l.Errorf("Internal server error: '%+v'", err)
		resp.Response = &actions_proto.GetLeaderboardResponse_InternalServerError{
			&errors_proto.InternalServerError{
				"We are facing some issues on the server. Please try again in some time.",
			},
		}
		return resp
	}

	userId := getUserId(sess)
	startingId := req.StartingId
	count := req.Count

	leaderboard, currentUserLeaderboard, totalUsers, err := models.GetLeaderboard(userId, startingId, count)
	if err != nil {
		return internalServerError(err)
	}

	rankList := make(map[uint32]*models_proto.LeaderboardRow)
	for _, leaderboardEntry := range leaderboard {
		rankList[leaderboardEntry.Id] = leaderboardEntry.ToProto()
	}
	//rankList[currentUserLeaderboard.Id] = currentUserLeaderboard.ToProto()

	resp.Response = &actions_proto.GetLeaderboardResponse_Result{
		&actions_proto.GetLeaderboardResponse_GetLeaderboardSuccessResponse{
			MyRank:     currentUserLeaderboard.Rank,
			TotalUsers: totalUsers,
			RankList:   rankList,
		},
	}

	l.Infof("Request completed successfully")

	return resp
}
