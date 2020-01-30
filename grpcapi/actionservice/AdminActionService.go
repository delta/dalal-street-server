package actionservice

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/delta/dalal-street-server/models"
	actions_pb "github.com/delta/dalal-street-server/proto_build/actions"
	"golang.org/x/net/context"
)

func (d *dalalActionService) SendNews(ctx context.Context, req *actions_pb.SendNewsRequest) (*actions_pb.SendNewsResponse, error) {
	resp := &actions_pb.SendNewsResponse{}
	// now call functions from models
	resp.StatusMessage = "Done"
	resp.StatusCode = actions_pb.SendNewsResponse_OK
	return resp, nil
}

func (d *dalalActionService) OpenMarket(ctx context.Context, req *actions_pb.OpenMarketRequest) (*actions_pb.OpenMarketResponse, error) {

	var l = logger.WithFields(logrus.Fields{
		"method":        "OpenMarket",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	resp := &actions_pb.OpenMarketResponse{}

	err := models.OpenMarket(req.UpdateDayHighAndLow)

	makeError := func(st actions_pb.OpenMarketResponse_StatusCode, msg string) (*actions_pb.OpenMarketResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	if err != nil {
		l.Errorf("Request failed due to %+v: ", err)
		return makeError(actions_pb.OpenMarketResponse_InternalServerError, getInternalErrorMessage(err))
	}

	resp.StatusMessage = "Done"
	resp.StatusCode = actions_pb.OpenMarketResponse_OK
	return resp, nil
}

func (d *dalalActionService) CloseMarket(ctx context.Context, req *actions_pb.CloseMarketRequest) (*actions_pb.CloseMarketResponse, error) {

	var l = logger.WithFields(logrus.Fields{
		"method":        "CloseMarket",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	resp := &actions_pb.CloseMarketResponse{}

	err := models.CloseMarket(req.UpdatePrevDayClose)

	makeError := func(st actions_pb.CloseMarketResponse_StatusCode, msg string) (*actions_pb.CloseMarketResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	if err != nil {
		l.Errorf("Request failed due to %+v: ", err)
		return makeError(actions_pb.CloseMarketResponse_InternalServerError, getInternalErrorMessage(err))
	}

	resp.StatusMessage = "Done"
	resp.StatusCode = actions_pb.CloseMarketResponse_OK
	return resp, nil
}

func (d *dalalActionService) SendNotifications(ctx context.Context, req *actions_pb.SendNotificationsRequest) (*actions_pb.SendNotificationsResponse, error) {

	var l = logger.WithFields(logrus.Fields{
		"method":        "SendNotifications",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	resp := &actions_pb.SendNotificationsResponse{}

	makeError := func(st actions_pb.SendNotificationsResponse_StatusCode, msg string) (*actions_pb.SendNotificationsResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	if req.IsGlobal && req.UserId != 0 {
		l.Errorf("Cannot send Global Notification to Non Zero Id")
		return makeError(actions_pb.SendNotificationsResponse_InternalServerError, "Cannot send Global Notification to Non Zero Id")
	}

	err := models.SendNotification(req.UserId, req.Text, req.IsGlobal)

	if err != nil {
		l.Errorf("Request failed due to %+v: ", err)
		return makeError(actions_pb.SendNotificationsResponse_InternalServerError, getInternalErrorMessage(err))
	}

	resp.StatusMessage = "Done"
	resp.StatusCode = actions_pb.SendNotificationsResponse_OK
	return resp, nil
}

func (d *dalalActionService) LoadStocks(ctx context.Context, req *actions_pb.LoadStocksRequest) (*actions_pb.LoadStocksResponse, error) {

	var l = logger.WithFields(logrus.Fields{
		"method":        "LoadStocks",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	err := models.LoadStocks()

	resp := &actions_pb.LoadStocksResponse{}

	makeError := func(st actions_pb.LoadStocksResponse_StatusCode, msg string) (*actions_pb.LoadStocksResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	if err != nil {
		l.Errorf("Request failed due to %+v: ", err)
		return makeError(actions_pb.LoadStocksResponse_InternalServerError, getInternalErrorMessage(err))
	}

	resp.StatusMessage = "Done"
	resp.StatusCode = actions_pb.LoadStocksResponse_OK
	return resp, nil
}

func (d *dalalActionService) AddStocksToExchange(ctx context.Context, req *actions_pb.AddStocksToExchangeRequest) (*actions_pb.AddStocksToExchangeResponse, error) {

	var l = logger.WithFields(logrus.Fields{
		"method":        "AddStocksToExchange",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	resp := &actions_pb.AddStocksToExchangeResponse{}

	makeError := func(st actions_pb.AddStocksToExchangeResponse_StatusCode, msg string) (*actions_pb.AddStocksToExchangeResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	stock, err := models.GetStockCopy(req.StockId)

	l.Debugf("Adding new stocks to Exchange for %s", stock.FullName)

	if err != nil {
		l.Errorf("Request failed due to %+v: ", err)
		return makeError(actions_pb.AddStocksToExchangeResponse_InternalServerError, getInternalErrorMessage(err))

	}

	err = models.AddStocksToExchange(req.StockId, req.NewStocks)

	if err != nil {
		l.Errorf("Request failed due to %+v: ", err)
		return makeError(actions_pb.AddStocksToExchangeResponse_InternalServerError, getInternalErrorMessage(err))
	}

	resp.StatusMessage = "Done"
	resp.StatusCode = actions_pb.AddStocksToExchangeResponse_OK
	return resp, nil
}

func (d *dalalActionService) UpdateStockPrice(ctx context.Context, req *actions_pb.UpdateStockPriceRequest) (*actions_pb.UpdateStockPriceResponse, error) {

	var l = logger.WithFields(logrus.Fields{
		"method":        "UpdateStockPrice",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	resp := &actions_pb.UpdateStockPriceResponse{}

	makeError := func(st actions_pb.UpdateStockPriceResponse_StatusCode, msg string) (*actions_pb.UpdateStockPriceResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	stock, err := models.GetStockCopy(req.StockId)

	l.Debugf("Adding new stocks to Exchange for %s", stock.FullName)

	if err != nil {
		l.Errorf("Request failed due to %+v: ", err)
		return makeError(actions_pb.UpdateStockPriceResponse_InternalServerError, getInternalErrorMessage(err))

	}

	err = models.UpdateStockPrice(req.StockId, req.NewPrice, 10000)

	if err != nil {
		l.Errorf("Request failed due to %+v: ", err)
		return makeError(actions_pb.UpdateStockPriceResponse_InternalServerError, getInternalErrorMessage(err))
	}

	resp.StatusMessage = "Done"
	resp.StatusCode = actions_pb.UpdateStockPriceResponse_OK
	return resp, nil
}

func (d *dalalActionService) AddMarketEvent(ctx context.Context, req *actions_pb.AddMarketEventRequest) (*actions_pb.AddMarketEventResponse, error) {

	var l = logger.WithFields(logrus.Fields{
		"method":        "AddMarketEvent",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	resp := &actions_pb.AddMarketEventResponse{}

	makeError := func(st actions_pb.AddMarketEventResponse_StatusCode, msg string) (*actions_pb.AddMarketEventResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	stock, err := models.GetStockCopy(req.StockId)

	l.Debugf("Adding Market Event for %s", stock.FullName)

	if err != nil {
		l.Errorf("Request failed due to %+v: ", err)
		return makeError(actions_pb.AddMarketEventResponse_InternalServerError, getInternalErrorMessage(err))
	}

	if req.IsGlobal && req.StockId != 0 {
		l.Errorf("Cannot send Global Notification to Non Zero Stock Id")
		return makeError(actions_pb.AddMarketEventResponse_InternalServerError, "Cannot send Global Notification to Non Zero Stock Id")
	}

	err = models.AddMarketEvent(req.StockId, req.Headline, req.Text, req.IsGlobal, req.ImageUrl)

	if err != nil {
		l.Errorf("Request failed due to %+v: ", err)
		return makeError(actions_pb.AddMarketEventResponse_InternalServerError, getInternalErrorMessage(err))
	}

	resp.StatusMessage = "Done"
	resp.StatusCode = actions_pb.AddMarketEventResponse_OK
	return resp, nil
}

func (d *dalalActionService) SendDividends(ctx context.Context, req *actions_pb.SendDividendsRequest) (*actions_pb.SendDividendsResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "SendDividends",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	l.Infof("Request for dividends sent")

	resp := &actions_pb.SendDividendsResponse{}
	makeError := func(st actions_pb.SendDividendsResponse_StatusCode, msg string) (*actions_pb.SendDividendsResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	if !models.IsMarketOpen() {
		return makeError(actions_pb.SendDividendsResponse_MarketClosedError, "Market Is closed. You cannot send dividends right now.")
	}

	stockID := req.StockId
	dividendAmount := req.DividendAmount

	err := models.PerformDividendsTransaction(stockID, dividendAmount)

	if err == nil {
		resp.StatusCode = 0
		resp.StatusMessage = "OK"

	}

	switch e := err.(type) {
	case models.InvalidStockIdError:
		return makeError(actions_pb.SendDividendsResponse_InvalidStockIdError, e.Error())
	}
	if err != nil {
		l.Errorf("Request failed due to %+v: ", err)
		return makeError(actions_pb.SendDividendsResponse_InternalServerError, getInternalErrorMessage(err))
	}

	l.Infof("Request completed successfully")

	return resp, nil
}
