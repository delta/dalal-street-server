package actions

import (
	"fmt"
	"strconv"

	"github.com/Sirupsen/logrus"

	"github.com/thakkarparth007/dalal-street-server/models"
	"github.com/thakkarparth007/dalal-street-server/session"
	actions_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/actions"
	errors_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/errors"
	models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

var logger *logrus.Entry

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
	resp.Response = &actions_proto.BuyStocksFromExchangeResponse_Result{
		&actions_proto.BuyStocksFromExchangeResponse_BuyStocksFromExchangeSuccessResponse{
			TradingPrice: 123,
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
		l.Infof("Internal server error: '%+v'", err)
		resp.Response = &actions_proto.LoginResponse_InternalServerError{
			&errors_proto.InternalServerError{
				"We are facing some issues on the server. Please try again in some time.",
			},
		}
		return resp
	}

	email := req.GetEmail()
	password := req.GetPassword()

	user, err := models.Login(email, password)

	switch {
	case err == models.UnauthorizedError:
		return invalidCredentialsError("Incorrect username/password combination. Please use your Pragyan credentials.")
	case err == models.NotRegisteredError:
		return invalidCredentialsError("You have not registered for Dalal Street on the Pragyan website")
	case err != nil:
		return internalServerError(err)
	}

	l.Debugf("models.Login returned without error %+v", user)

	if err := sess.Set("userId", strconv.Itoa(int(user.Id))); err != nil {
		return internalServerError(err)
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

	resp.Response = &actions_proto.LoginResponse_Result{
		&actions_proto.LoginResponse_LoginSuccessResponse{
			SessionId:   sess.GetId(),
			User:        user.ToProto(),
			StocksOwned: stocksOwned,
			StockList:   stockListProto,
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
	resp.Response = &actions_proto.MortgageStocksResponse_Result{
		&actions_proto.MortgageStocksResponse_MortgageStocksSuccessResponse{
			Success:      true,
			TradingPrice: 123,
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
	resp.Response = &actions_proto.PlaceAskOrderResponse_Result{
		&actions_proto.PlaceAskOrderResponse_PlaceAskOrderSuccessResponse{
			AskId: 123,
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
	resp.Response = &actions_proto.PlaceBidOrderResponse_Result{
		&actions_proto.PlaceBidOrderResponse_PlaceBidOrderSuccessResponse{
			BidId: 123,
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
	resp.Response = &actions_proto.RetrieveMortgageStocksResponse_Result{
		&actions_proto.RetrieveMortgageStocksResponse_RetrieveMortgageStocksSuccessResponse{
			Success:      true,
			TradingPrice: 123,
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
	stockDetails := &models.Stock{
		Id:               23,
		ShortName:        "zold",
		FullName:         "PastCry",
		Description:      "This Stock is a stock :P",
		CurrentPrice:     200,
		DayHigh:          300,
		DayLow:           100,
		AllTimeHigh:      400,
		AllTimeLow:       90,
		StocksInExchange: 123,
		StocksInMarket:   234,
		UpOrDown:         true,
		CreatedAt:        "2017-02-09T00:00:00",
		UpdatedAt:        "2017-02-09T00:00:00",
	}
	stockHistoryMap := make(map[string]*models_proto.StockHistory)
	stockHistoryMap["2017-02-09T00:00:00"] = (&models.StockHistory{
		StockId:    3,
		StockPrice: 23,
		CreatedAt:  "2017-02-09T00:00:00",
	}).ToProto()
	resp := &actions_proto.GetCompanyProfileResponse{}
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

	marketEventsMap := make(map[uint32]*models_proto.MarketEvent)
	marketEventsMap[1] = (&models.MarketEvent{
		Id:           2,
		StockId:      3,
		Text:         "Hello World",
		EmotionScore: -54,
		CreatedAt:    "2017-02-09T00:00:00",
	}).ToProto()
	resp := &actions_proto.GetMarketEventsResponse{}
	resp.Response = &actions_proto.GetMarketEventsResponse_Result{
		&actions_proto.GetMarketEventsResponse_GetMarketEventsSuccessResponse{
			MarketEvents: marketEventsMap,
			MoreExists:   false,
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

	asksMap := make(map[uint32]*models_proto.Ask)
	asksMap[1] = (&models.Ask{
		Id:                     2,
		UserId:                 2,
		StockId:                3,
		Price:                  5,
		OrderType:              models.Market,
		StockQuantity:          20,
		StockQuantityFulfilled: 20,
		IsClosed:               true,
		CreatedAt:              "2017-02-09T00:00:00",
		UpdatedAt:              "2017-02-09T00:00:00",
	}).ToProto()
	resp := &actions_proto.GetMyAsksResponse{}
	resp.Response = &actions_proto.GetMyAsksResponse_Result{
		&actions_proto.GetMyAsksResponse_GetMyAsksSuccessResponse{
			AskOrders:  asksMap,
			MoreExists: false,
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

	bidsMap := make(map[uint32]*models_proto.Bid)
	bidsMap[1] = (&models.Bid{
		Id:                     2,
		UserId:                 2,
		StockId:                3,
		Price:                  5,
		OrderType:              models.Market,
		StockQuantity:          20,
		StockQuantityFulfilled: 20,
		IsClosed:               true,
		CreatedAt:              "2017-02-09T00:00:00",
		UpdatedAt:              "2017-02-09T00:00:00",
	}).ToProto()
	resp := &actions_proto.GetMyBidsResponse{}
	resp.Response = &actions_proto.GetMyBidsResponse_Result{
		&actions_proto.GetMyBidsResponse_GetMyBidsSuccessResponse{
			BidOrders:  bidsMap,
			MoreExists: false,
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

	notificationsMap := make(map[uint32]*models_proto.Notification)
	notificationsMap[1] = (&models.Notification{
		Id:        2,
		UserId:    3,
		Text:      "Hello World",
		CreatedAt: "2017-02-09T00:00:00",
	}).ToProto()
	resp := &actions_proto.GetNotificationsResponse{}
	resp.Response = &actions_proto.GetNotificationsResponse_Result{
		&actions_proto.GetNotificationsResponse_GetNotificationsSuccessResponse{
			Notifications: notificationsMap,
			MoreExists:    false,
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

	transactionsMap := make(map[uint32]*models_proto.Transaction)
	transactionsMap[1] = (&models.Transaction{
		Id:            2,
		UserId:        20,
		StockId:       12,
		Type:          models.OrderFillTransaction,
		StockQuantity: -20,
		Price:         300,
		Total:         -300,
		CreatedAt:     "2017-02-09T00:00:00",
	}).ToProto()
	resp := &actions_proto.GetTransactionsResponse{}
	resp.Response = &actions_proto.GetTransactionsResponse_Result{
		&actions_proto.GetTransactionsResponse_GetTransactionsSuccessResponse{
			TransactionsMap: transactionsMap,
			MoreExists:      false,
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

	mortgageMap := make(map[uint32]*actions_proto.GetMortgageDetailsResponse_GetMortgageDetailsSuccessResponse_MortgageDetails)
	mortgageMap[1] = &actions_proto.GetMortgageDetailsResponse_GetMortgageDetailsSuccessResponse_MortgageDetails{
		StockId:         1,
		NumStocksInBank: 12,
	}
	resp := &actions_proto.GetMortgageDetailsResponse{}
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

	rankList := make(map[uint32]*models_proto.LeaderboardRow)
	rankList[1] = (&models.LeaderboardRow{
		Id:         2,
		UserId:     5,
		Rank:       1,
		Cash:       1000,
		Debt:       10,
		StockWorth: -50,
		TotalWorth: -300,
	}).ToProto()
	resp := &actions_proto.GetLeaderboardResponse{}
	resp.Response = &actions_proto.GetLeaderboardResponse_Result{
		&actions_proto.GetLeaderboardResponse_GetLeaderboardSuccessResponse{
			MyRank:          2,
			TotalUsers:      4,
			TotalPages:      2,
			RankList:        rankList,
			UpdatedBefore:   3,
			NextUpdateAfter: 5,
		},
	}

	l.Infof("Request completed successfully")

	return resp
}
