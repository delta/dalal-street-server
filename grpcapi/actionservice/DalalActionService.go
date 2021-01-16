package actionservice

import (
	"strconv"

	"github.com/sirupsen/logrus"

	"golang.org/x/net/context"

	"github.com/delta/dalal-street-server/matchingengine"
	"github.com/delta/dalal-street-server/session"

	pb "github.com/delta/dalal-street-server/proto_build"

	"github.com/delta/dalal-street-server/utils"
)

var logger *logrus.Entry

func getInternalErrorMessage(err error) string {
	if utils.IsProdEnv() {
		return "Oops! Something went wrong. Please try again in some time."
	}
	return err.Error()
}

func getUserId(ctx context.Context) uint32 {
	sess := ctx.Value("session").(session.Session)
	userId, _ := sess.Get("userId")
	userIdInt, _ := strconv.ParseUint(userId, 10, 32)
	return uint32(userIdInt)
}

type dalalActionService struct {
	matchingEngine matchingengine.MatchingEngine
}

// NewDalalActionService returns instance of DalalActionServiceServer
func NewDalalActionService(me matchingengine.MatchingEngine) pb.DalalActionServiceServer {
	logger = utils.Logger.WithFields(logrus.Fields{
		"module": "grpcapi.actions",
	})

	return &dalalActionService{
		matchingEngine: me,
	}
}
