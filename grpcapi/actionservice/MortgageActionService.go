package actionservice

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/delta/dalal-street-server/models"
	actions_pb "github.com/delta/dalal-street-server/proto_build/actions"
	"golang.org/x/net/context"
)

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
	if !models.IsUserPhoneVerified(userId) {
		return makeError(actions_pb.MortgageStocksResponse_UserNotPhoneVerfiedError, "Your phone number has not been verified. Please verify phone number in order to play the game.")
	}

	if models.IsUserBlocked(userId) {
		return makeError(actions_pb.MortgageStocksResponse_UserBlockedError, "Your account has been blocked due to malpractice.")
	}

	stockId := req.StockId
	stockQty := -int64(req.StockQuantity)

	transaction, err := models.PerformMortgageTransaction(userId, stockId, stockQty, 0)

	switch e := err.(type) {
	case models.NotEnoughStocksError:
		return makeError(actions_pb.MortgageStocksResponse_NotEnoughStocksError, e.Error())
	case models.WayTooMuchCashError:
		return makeError(actions_pb.MortgageStocksResponse_NotEnoughStocksError, e.Error())
	case models.StockBankruptError:
		return makeError(actions_pb.MortgageStocksResponse_StockBankruptError, e.Error())
	}
	if err != nil {
		l.Errorf("Request failed due to: %+v", err)
		return makeError(actions_pb.MortgageStocksResponse_InternalServerError, getInternalErrorMessage(err))
	}

	resp.Transaction = transaction.ToProto()

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
	if !models.IsUserPhoneVerified(userID) {
		return makeError(actions_pb.RetrieveMortgageStocksResponse_UserNotPhoneVerfiedError, "Your phone number has not been verified. Please verify phone number in order to play the game.")
	}
	if models.IsUserBlocked(userID) {
		return makeError(actions_pb.RetrieveMortgageStocksResponse_UserBlockedError, "Your account has been blocked due to malpractice.")
	}

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
	case models.StockBankruptError:
		return makeError(actions_pb.RetrieveMortgageStocksResponse_StockBankruptError, e.Error())
	}
	if err != nil {
		l.Errorf("Request failed due to %+v: ", err)
		return makeError(actions_pb.RetrieveMortgageStocksResponse_InternalServerError, getInternalErrorMessage(err))
	}

	resp.Transaction = transaction.ToProto()

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
