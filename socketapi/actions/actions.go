package actions

import (
	"fmt"

	"github.com/Sirupsen/logrus"

	"github.com/thakkarparth007/dalal-street-server/models"
	"github.com/thakkarparth007/dalal-street-server/session"
	actions_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/actions"
	errors_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/errors"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

var logger *logrus.Entry

func InitActions() {
	logger = utils.Logger.WithFields(logrus.Fields{
		"module": "socketapi.actions",
	})
}

func BuyStocksFromExchange(sess session.Session, req *actions_proto.BuyStocksFromExchangeRequest) *actions_proto.BuyStocksFromExchangeResponse {

	return nil
}

func CancelAskOrder(sess session.Session, req *actions_proto.CancelAskOrderRequest) *actions_proto.CancelAskOrderResponse {
	return nil
}

func CancelBidOrder(sess session.Session, req *actions_proto.CancelBidOrderRequest) *actions_proto.CancelBidOrderResponse {
	return nil
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

	switch err {
	case models.UnauthorizedError:
		return invalidCredentialsError("Incorrect username/password combination. Please use your Pragyan credentials.")
	case models.NotRegisteredError:
		return invalidCredentialsError("You have not registered for Dalal Street on the Pragyan website")
	case models.InternalError:
		return internalServerError(err)
	}

	l.Debugf("models.Login returned without error")

	if err := sess.Set("userId", string(user.Id)); err != nil {
		return internalServerError(err)
	}

	l.Debugf("Session successfully set. UserId: %+v, Session id: %+v", user.Id, sess.GetId())

	resp.Response = &actions_proto.LoginResponse_Result{
		Result: &actions_proto.LoginResponse_LoginSuccessResponse{
			SessionId: sess.GetId(),
			User:      user.ToProto(),
			// populate other fields :)
			// StocksOwned and StockList
		},
	}

	l.Infof("Request completed successfully")

	return resp
}

func Logout(ses session.Session, req *actions_proto.LogoutRequest) *actions_proto.LogoutResponse {
	return nil
}

func MortgageStocks(sess session.Session, req *actions_proto.MortgageStocksRequest) *actions_proto.MortgageStocksResponse {
	return nil
}

func PlaceAskOrder(sess session.Session, req *actions_proto.PlaceAskOrderRequest) *actions_proto.PlaceAskOrderResponse {
	return nil
}

func PlaceBidOrder(sess session.Session, req *actions_proto.PlaceBidOrderRequest) *actions_proto.PlaceBidOrderResponse {
	return nil
}

func RetrieveMortgageStocks(sess session.Session, req *actions_proto.RetrieveMortgageStocksRequest) *actions_proto.RetrieveMortgageStocksResponse {
	return nil
}

func Unsubscribe(sess session.Session, req *actions_proto.UnsubscribeRequest) *actions_proto.UnsubscribeResponse {
	return nil
}

func Subscribe(done <-chan struct{}, updates chan interface{}, sess session.Session, req *actions_proto.SubscribeRequest) *actions_proto.SubscribeResponse {
	return nil
}

func GetCompanyProfile(sess session.Session, req *actions_proto.GetCompanyProfileRequest) *actions_proto.GetCompanyProfileResponse {
	return nil
}

func GetMarketEvents(sess session.Session, req *actions_proto.GetMarketEventsRequest) *actions_proto.GetMarketEventsResponse {
	return nil
}

func GetMyAsks(sess session.Session, req *actions_proto.GetMyAsksRequest) *actions_proto.GetMyAsksResponse {
	return nil
}

func GetMyBids(sess session.Session, req *actions_proto.GetMyBidsRequest) *actions_proto.GetMyBidsResponse {
	return nil
}

func GetNotifications(sess session.Session, req *actions_proto.GetNotificationsRequest) *actions_proto.GetNotificationsResponse {
	return nil
}

func GetTransactions(sess session.Session, req *actions_proto.GetTransactionsRequest) *actions_proto.GetTransactionsResponse {
	return nil
}

func GetMortgageDetails(sess session.Session, req *actions_proto.GetMortgageDetailsRequest) *actions_proto.GetMortgageDetailsResponse {
	return nil
}

func GetLeaderboard(sess session.Session, req *actions_proto.GetLeaderboardRequest) *actions_proto.GetLeaderboardResponse {
	return nil
}
