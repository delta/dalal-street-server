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

	status, err := models.PerformDividendsTransaction(stockID, dividendAmount)

	if status == "OK" {
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
