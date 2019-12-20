package actionservice

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/delta/dalal-street-server/models"
	actions_pb "github.com/delta/dalal-street-server/proto_build/actions"
	"golang.org/x/net/context"
)

func (d *dalalActionService) CreateBot(ctx context.Context, req *actions_pb.CreateBotRequest) (*actions_pb.CreateBotResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "CreateBot",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("Creating Bot")

	resp := &actions_pb.CreateBotResponse{}
	makeError := func(st actions_pb.CreateBotResponse_StatusCode, msg string) (*actions_pb.CreateBotResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	user, err := models.CreateBot(req.GetBotUserId())
	if err != nil {
		l.Errorf("Unable to Create bot models.CreateBot threw error %+v", err)
		return makeError(actions_pb.CreateBotResponse_InternalServerError, getInternalErrorMessage(err))
	}

	resp.User = user.ToProto()

	return resp, nil
}
