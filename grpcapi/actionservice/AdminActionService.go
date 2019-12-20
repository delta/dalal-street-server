package actionservice

import (
	"github.com/delta/dalal-street-server/models"
	actions_pb "github.com/delta/dalal-street-server/proto_build/actions"
	"golang.org/x/net/context"
)

func (d *dalalActionService) SendNews(ctx context.Context, req *actions_pb.SendNewsRequest) (*actions_pb.SendNewsResponse, error) {
	resp := &actions_pb.SendNewsResponse{}
	// now call functions from models
	models.PerformDividendTransaction()
	resp.StatusMessage = "Done"
	resp.StatusCode = actions_pb.SendNewsResponse_OK
	return resp, nil
}
