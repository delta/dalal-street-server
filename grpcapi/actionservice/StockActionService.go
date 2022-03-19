package actionservice

import (
	"fmt"

	"github.com/delta/dalal-street-server/models"
	actions_pb "github.com/delta/dalal-street-server/proto_build/actions"
	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

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
	if !models.IsUserPhoneVerified(userId) {
		return makeError(actions_pb.BuyStocksFromExchangeResponse_UserNotPhoneVerfiedError, "Your phone number has not been verified. Please verify phone number in order to play the game.")
	}

	if models.IsUserBlocked(userId) {
		return makeError(actions_pb.BuyStocksFromExchangeResponse_UserBlockedError, "Your account has been blocked due to malpractice.")
	}

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
	case models.StockBankruptError:
		return makeError(actions_pb.BuyStocksFromExchangeResponse_StockBankruptError, e.Error())
	}

	if err != nil {
		l.Errorf("Request failed due to: %+v", err)
		return makeError(actions_pb.BuyStocksFromExchangeResponse_InternalServerError, getInternalErrorMessage(err))
	}

	resp.Transaction = transaction.ToProto()

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

func (d *dalalActionService) GetStockList(ctx context.Context, req *actions_pb.GetStockListRequest) (*actions_pb.GetStockListResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetStockList",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	l.Infof("GetStockList requested")

	resp := &actions_pb.GetStockListResponse{}

	stockList := models.GetAllStocks()
	stockListProto := make(map[uint32]*models_pb.Stock)

	for stockId, stock := range stockList {
		stockListProto[stockId] = stock.ToProto()
	}

	resp.StockList = stockListProto

	resp.StatusCode = actions_pb.GetStockListResponse_OK
	resp.StatusMessage = "Success"

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalActionService) GetIpoStockList(ctx context.Context, req *actions_pb.GetIpoStockListRequest) (*actions_pb.GetIpoStockListResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetIpoStockList",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	l.Infof("GetIpoStockList requested")

	makeError := func(st actions_pb.GetIpoStockListResponse_StatusCode, msg string) (*actions_pb.GetIpoStockListResponse, error) {
		resp := &actions_pb.GetIpoStockListResponse{}
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	resp := &actions_pb.GetIpoStockListResponse{}


	ipoStockList, err := models.GetAllIpoStocks()

	if err != nil {
		return makeError(actions_pb.GetIpoStockListResponse_InternalServerError, getInternalErrorMessage(err))
	}

	ipoStockListProto := make(map[uint32]*models_pb.IpoStock)

	for ipoStockId, ipoStock := range ipoStockList {
		ipoStockListProto[ipoStockId] = ipoStock.ToProto()
	}

	resp.IpoStockList = ipoStockListProto

	resp.StatusCode = actions_pb.GetIpoStockListResponse_OK
	resp.StatusMessage = "Success"

	l.Infof("Request completed successfully")

	return resp, nil
}
