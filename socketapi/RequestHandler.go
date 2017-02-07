package socketapi

import (
	"errors"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/proto"

	"github.com/thakkarparth007/dalal-street-server/socketapi/actions"
	socketapi_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build"
	datastreams_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/datastreams"
)

func makeResponseExceptSubscribe(c *client, reqwrap *socketapi_proto.RequestWrapper) (*socketapi_proto.DalalMessage, error) {
	dm := &socketapi_proto.DalalMessage{}
	rw := &socketapi_proto.ResponseWrapper{
		RequestId: reqwrap.RequestId,
	}

	dm.MessageType = &socketapi_proto.DalalMessage_ResponseWrapper{
		ResponseWrapper: rw,
	}

	if req := reqwrap.GetBuyStocksFromExchangeRequest(); req != nil {
		rw.Response = &socketapi_proto.ResponseWrapper_BuyStocksFromExchangeResponse{
			BuyStocksFromExchangeResponse: actions.BuyStocksFromExchange(c.sess, req),
		}
	} else if req := reqwrap.GetCancelAskOrderRequest(); req != nil {
		rw.Response = &socketapi_proto.ResponseWrapper_CancelAskOrderResponse{
			CancelAskOrderResponse: actions.CancelAskOrder(c.sess, req),
		}
	} else if req := reqwrap.GetCancelBidOrderRequest(); req != nil {
		rw.Response = &socketapi_proto.ResponseWrapper_CancelBidOrderResponse{
			CancelBidOrderResponse: actions.CancelBidOrder(c.sess, req),
		}
	} else if req := reqwrap.GetLoginRequest(); req != nil {
		rw.Response = &socketapi_proto.ResponseWrapper_LoginResponse{
			LoginResponse: actions.Login(c.sess, req),
		}
	} else if req := reqwrap.GetLogoutRequest(); req != nil {
		rw.Response = &socketapi_proto.ResponseWrapper_LogoutResponse{
			LogoutResponse: actions.Logout(c.sess, req),
		}
	} else if req := reqwrap.GetMortgageStocksRequest(); req != nil {
		rw.Response = &socketapi_proto.ResponseWrapper_MortgageStocksResponse{
			MortgageStocksResponse: actions.MortgageStocks(c.sess, req),
		}
	} else if req := reqwrap.GetPlaceAskOrderRequest(); req != nil {
		rw.Response = &socketapi_proto.ResponseWrapper_PlaceAskOrderResponse{
			PlaceAskOrderResponse: actions.PlaceAskOrder(c.sess, req),
		}
	} else if req := reqwrap.GetPlaceBidOrderRequest(); req != nil {
		rw.Response = &socketapi_proto.ResponseWrapper_PlaceBidOrderResponse{
			PlaceBidOrderResponse: actions.PlaceBidOrder(c.sess, req),
		}
	} else if req := reqwrap.GetRetrieveMortgageStocksRequest(); req != nil {
		rw.Response = &socketapi_proto.ResponseWrapper_RetrieveMortgageStocksResponse{
			RetrieveMortgageStocksResponse: actions.RetrieveMortgageStocks(c.sess, req),
		}
	} else if req := reqwrap.GetUnsubscribeRequest(); req != nil {
		rw.Response = &socketapi_proto.ResponseWrapper_UnsubscribeResponse{
			UnsubscribeResponse: actions.Unsubscribe(c.sess, req),
		}
		// The ugly 'GetGet' is unfortunate, but that ugliness remains contained within
		// this file. The first Get is Protobuf's 'Get'. The second Get is part of the
		// actual request name
	} else if req := reqwrap.GetGetCompanyProfileRequest(); req != nil {
		rw.Response = &socketapi_proto.ResponseWrapper_GetCompanyProfileResponse{
			GetCompanyProfileResponse: actions.GetCompanyProfile(c.sess, req),
		}
	} else if req := reqwrap.GetGetMarketEventsRequest(); req != nil {
		rw.Response = &socketapi_proto.ResponseWrapper_GetMarketEventsResponse{
			GetMarketEventsResponse: actions.GetMarketEvents(c.sess, req),
		}
	} else if req := reqwrap.GetGetMyAsksRequest(); req != nil {
		rw.Response = &socketapi_proto.ResponseWrapper_GetMyAsksResponse{
			GetMyAsksResponse: actions.GetMyAsks(c.sess, req),
		}
	} else if req := reqwrap.GetGetMyBidsRequest(); req != nil {
		rw.Response = &socketapi_proto.ResponseWrapper_GetMyBidsResponse{
			GetMyBidsResponse: actions.GetMyBids(c.sess, req),
		}
	} else if req := reqwrap.GetGetNotificationsRequest(); req != nil {
		rw.Response = &socketapi_proto.ResponseWrapper_GetNotificationsResponse{
			GetNotificationsResponse: actions.GetNotifications(c.sess, req),
		}
	} else if req := reqwrap.GetGetTransactionsRequest(); req != nil {
		rw.Response = &socketapi_proto.ResponseWrapper_GetTransactionsResponse{
			GetTransactionsResponse: actions.GetTransactions(c.sess, req),
		}
	} else if req := reqwrap.GetGetMortgageDetailsRequest(); req != nil {
		rw.Response = &socketapi_proto.ResponseWrapper_GetMortgageDetailsResponse{
			GetMortgageDetailsResponse: actions.GetMortgageDetails(c.sess, req),
		}
	} else if req := reqwrap.GetGetLeaderboardRequest(); req != nil {
		rw.Response = &socketapi_proto.ResponseWrapper_GetLeaderboardResponse{
			GetLeaderboardResponse: actions.GetLeaderboard(c.sess, req),
		}
	} else {
		return nil, errors.New(fmt.Sprintf("Unexpected type '%T'", reqwrap))
	}

	return dm, nil
}

func makeDataStreamUpdate(update interface{}) (*socketapi_proto.DalalMessage, error) {
	dm := &socketapi_proto.DalalMessage{}
	dsuw := &socketapi_proto.DataStreamUpdateWrapper{}

	dm.MessageType = &socketapi_proto.DalalMessage_DataStreamUpdateWrapper{
		DataStreamUpdateWrapper: dsuw,
	}

	switch update.(type) {
	case *datastreams_proto.MarketDepthUpdate:
		dsuw.Update = &socketapi_proto.DataStreamUpdateWrapper_MarketDepthUpdate{
			MarketDepthUpdate: update.(*datastreams_proto.MarketDepthUpdate),
		}
	case *datastreams_proto.MarketEventUpdate:
		dsuw.Update = &socketapi_proto.DataStreamUpdateWrapper_MarketDepthUpdate{
			MarketDepthUpdate: update.(*datastreams_proto.MarketDepthUpdate),
		}
	case *datastreams_proto.NotificationUpdate:
		dsuw.Update = &socketapi_proto.DataStreamUpdateWrapper_NotificationUpdate{
			NotificationUpdate: update.(*datastreams_proto.NotificationUpdate),
		}
	case *datastreams_proto.StockExchangeUpdate:
		dsuw.Update = &socketapi_proto.DataStreamUpdateWrapper_StockExchangeUpdate{
			StockExchangeUpdate: update.(*datastreams_proto.StockExchangeUpdate),
		}
	case *datastreams_proto.StockPricesUpdate:
		dsuw.Update = &socketapi_proto.DataStreamUpdateWrapper_StockPricesUpdate{
			StockPricesUpdate: update.(*datastreams_proto.StockPricesUpdate),
		}
	case *datastreams_proto.TransactionUpdate:
		dsuw.Update = &socketapi_proto.DataStreamUpdateWrapper_TransactionUpdate{
			TransactionUpdate: update.(*datastreams_proto.TransactionUpdate),
		}
	default:
		return nil, errors.New(fmt.Sprintf("Unexpected type '%T'", update))
	}

	return dm, nil
}

// need to stop this when client closes connection
func handleRequest(c *client, reqwrap *socketapi_proto.RequestWrapper) {
	var l = socketApiLogger.WithFields(logrus.Fields{
		"method":        "client.handleRequest",
		"param_c":       c,
		"param_reqwrap": reqwrap,
	})

	// Ensure that whatever happens in this request doesn't take down the whole server!
	defer func() {
		if r := recover(); r != nil {
			l.Errorf("Failed while handling request: '%+v'", r)
		}
	}()

	if reqwrap == nil {
		l.Warnf("Got nil instead of RequestWrapper!. Not processing this.")
		return
	}

	// Handle all requests except subscription requests. They require more work.
	if r := reqwrap.GetSubscribeRequest(); r == nil {
		dm, err := makeResponseExceptSubscribe(c, reqwrap)

		if err != nil {
			l.Errorf("Unable to process request. '%+v'", err)
			return
		}

		data, err := proto.Marshal(dm)
		if err != nil {
			l.Errorf("Unable to marshal response. Response: '%+v'", dm)
			return
		}

		// Two cases are possible:
		// The client doesn't want any more data (he's disconnected). In which case we should return
		// Or, the client has received our data.
		//
		// This weird select-case is required because if we simply do 'c.send <- data' then there's a
		// possibility of this goroutine being blocked (if client is done, he won't read c.send, and
		// if the c.send's buffer is full then this write will block indefinitely.
		select {
		case <-c.done:
			return
		case c.send <- data:
			return
		}
	}

	// Everything below is to handle SubscribeRequest

	updatechan := make(chan interface{})
	done := make(chan struct{})
	defer close(done)

	dm := &socketapi_proto.DalalMessage{}
	rw := &socketapi_proto.ResponseWrapper{
		RequestId: reqwrap.RequestId,
	}

	dm.MessageType = &socketapi_proto.DalalMessage_ResponseWrapper{
		ResponseWrapper: rw,
	}

	req := reqwrap.GetSubscribeRequest()
	resp := &socketapi_proto.ResponseWrapper_SubscribeResponse{
		SubscribeResponse: actions.Subscribe(done, updatechan, c.sess, req),
	}

	rw.Response = resp

	for {
		select {
		case update, ok := <-updatechan:
			if !ok {
				l.Debugf("updatechan closed. Exiting.")
				return
			}

			dm, err := makeDataStreamUpdate(update)
			if err != nil {
				l.Errorf("Unable to convert update into datastream. Update: '%+v'", update)
				return
			}

			data, err := proto.Marshal(dm)
			if err != nil {
				l.Errorf("Error marshaling the datastreamupdate message. DalalMessage: '%+v'", dm)
			}

			select {
			case c.send <- data: // don't do anything if c.send <- data worked.
			case <-c.done:
				return
			}

		case <-c.done:
			return
		}
	}

}
