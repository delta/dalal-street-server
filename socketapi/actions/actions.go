package actions

import (
	"github.com/Sirupsen/logrus"

	"github.com/thakkarparth007/dalal-street-server/session"
	"github.com/thakkarparth007/dalal-street-server/utils"
	actions_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto/actions"
)

var logger *logrus.Entry

func InitActions() {
	logger = utils.Logger.WithFields(logrus.Fields{
		"module": "socketapi.actions",
	})
}

func BuyStocksFromExchange(sess *session.Session, req *actions_proto.BuyStocksFromExchangeRequest) *actions_proto.BuyStocksFromExchangeResponse {

	return nil
}

func CancelAskOrder(sess *session.Session, req *actions_proto.CancelAskOrderRequest) *actions_proto.CancelAskOrderResponse {
	return nil
}

func CancelBidOrder(sess *session.Session, req *actions_proto.CancelBidOrderRequest) *actions_proto.CancelBidOrderResponse {
	return nil
}

func Login(sess *session.Session, req *actions_proto.LoginRequest) *actions_proto.LoginResponse {
	return nil
}

func Logout(ses *session.Session, req *actions_proto.LogoutRequest) *actions_proto.LogoutResponse {
	return nil
}

func MortgageStocks(sess *session.Session, req *actions_proto.MortgageStocksRequest) *actions_proto.MortgageStocksResponse {
	return nil
}

func PlaceAskOrder(sess *session.Session, req *actions_proto.PlaceAskOrderRequest) *actions_proto.PlaceAskOrderResponse {
	return nil
}

func PlaceBidOrder(sess *session.Session, req *actions_proto.PlaceBidOrderRequest) *actions_proto.PlaceBidOrderResponse {
	return nil
}

func RetrieveMortgageStocks(sess *session.Session, req *actions_proto.RetrieveMortgageStocksRequest) *actions_proto.RetrieveMortgageStocksResponse {
	return nil
}

func Unsubscribe(sess *session.Session, req *actions_proto.UnsubscribeRequest) *actions_proto.UnsubscribeResponse {
	return nil
}

func Subscribe(done <-chan struct{}, updates chan interface{}, sess *session.Session, req *actions_proto.SubscribeRequest) *actions_proto.SubscribeResponse {
	return nil
}

