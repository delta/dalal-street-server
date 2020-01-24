package actionservice

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/delta/dalal-street-server/models"
	actions_pb "github.com/delta/dalal-street-server/proto_build/actions"
	"github.com/delta/dalal-street-server/session"
	"github.com/delta/dalal-street-server/utils"
	"golang.org/x/net/context"
)

func (d *dalalActionService) GetPortfolio(ctx context.Context, req *actions_pb.GetPortfolioRequest) (*actions_pb.GetPortfolioResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetPortfolio",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	l.Infof("Getting Portfolio")

	resp := &actions_pb.GetPortfolioResponse{}
	makeError := func(st actions_pb.GetPortfolioResponse_StatusCode, msg string) (*actions_pb.GetPortfolioResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	sess := ctx.Value("session").(session.Session)
	userId := getUserId(ctx)

	user, err := models.GetUserCopy(userId)
	if err != nil {
		l.Errorf("Request failed. User for Id does not exist. Error: %+v", err)
		if utils.IsProdEnv() {
			return makeError(actions_pb.GetPortfolioResponse_InvalidCredentialsError, "Invalid credentials given")
		}
		return makeError(actions_pb.GetPortfolioResponse_InvalidCredentialsError, fmt.Sprintf("User for ID does not exist: %+v", err))
	}

	stocksOwned, err := models.GetStocksOwned(user.Id)
	if err != nil {
		l.Errorf("Unable to get Stocks for User Id. Error: %+v", err)
		return makeError(actions_pb.GetPortfolioResponse_InternalServerError, "")
	}

	reservedStocksOwned, err := models.GetReservedStocksOwned(user.Id)
	if err != nil {
		l.Errorf("Unable to get Reserved Stocks for User Id. Error: %+v", err)
		return makeError(actions_pb.GetPortfolioResponse_InternalServerError, "")
	}

	resp.SessionId = sess.GetID()
	resp.User = user.ToProto()
	resp.StocksOwned = stocksOwned
	resp.ReservedStocksOwned = reservedStocksOwned

	return resp, nil
}

func (d *dalalActionService) GetMarketEvents(ctx context.Context, req *actions_pb.GetMarketEventsRequest) (*actions_pb.GetMarketEventsResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetMarketEvents",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetMarketEvents requested")

	resp := &actions_pb.GetMarketEventsResponse{}

	lastId := req.LastEventId
	count := req.Count

	moreExists, marketEvents, err := models.GetMarketEvents(lastId, count)
	if err != nil {
		l.Errorf("Request failed due to: %+v", err)
		resp.StatusCode = actions_pb.GetMarketEventsResponse_InternalServerError
		resp.StatusMessage = getInternalErrorMessage(err)
		return resp, nil
	}

	resp.MoreExists = moreExists
	for _, marketEvent := range marketEvents {
		resp.MarketEvents = append(resp.MarketEvents, marketEvent.ToProto())
	}

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalActionService) GetNotifications(ctx context.Context, req *actions_pb.GetNotificationsRequest) (*actions_pb.GetNotificationsResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetNotifications",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetNotifications requested")

	resp := &actions_pb.GetNotificationsResponse{}

	lastId := req.LastNotificationId
	count := req.Count

	moreExists, notifications, err := models.GetNotifications(getUserId(ctx), lastId, count)
	if err != nil {
		l.Errorf("Request failed due to: %+v", err)
		resp.StatusCode = actions_pb.GetNotificationsResponse_InternalServerError
		resp.StatusMessage = getInternalErrorMessage(err)
		return resp, nil
	}

	//Convert to proto
	resp.MoreExists = moreExists
	for _, notification := range notifications {
		resp.Notifications = append(resp.Notifications, notification.ToProto())
	}

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalActionService) GetTransactions(ctx context.Context, req *actions_pb.GetTransactionsRequest) (*actions_pb.GetTransactionsResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetTransactions",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	l.Infof("GetTransactions requested")

	resp := &actions_pb.GetTransactionsResponse{}

	userId := getUserId(ctx)
	lastId := req.LastTransactionId
	count := req.Count

	moreExists, transactions, err := models.GetTransactions(userId, lastId, count)
	if err != nil {
		l.Errorf("Request failed due to: %+v", err)
		resp.StatusCode = actions_pb.GetTransactionsResponse_InternalServerError
		resp.StatusMessage = getInternalErrorMessage(err)
		return resp, nil
	}

	//Convert to proto
	resp.MoreExists = moreExists
	for _, transaction := range transactions {
		resp.Transactions = append(resp.Transactions, transaction.ToProto())
	}

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalActionService) GetLeaderboard(ctx context.Context, req *actions_pb.GetLeaderboardRequest) (*actions_pb.GetLeaderboardResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetLeaderboard",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetLeaderboard requested")

	resp := &actions_pb.GetLeaderboardResponse{}

	userId := getUserId(ctx)
	startingId := req.StartingId
	count := req.Count

	leaderboard, currentUserLeaderboard, totalUsers, err := models.GetLeaderboard(userId, startingId, count)
	if err != nil {
		l.Errorf("Request failed due to: %+v", err)
		resp.StatusCode = actions_pb.GetLeaderboardResponse_InternalServerError
		resp.StatusMessage = getInternalErrorMessage(err)
		return resp, nil
	}

	resp.MyRank = currentUserLeaderboard.Rank
	resp.TotalUsers = totalUsers
	for _, leaderboardEntry := range leaderboard {
		resp.RankList = append(resp.RankList, leaderboardEntry.ToProto())
	}

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalActionService) ForgotPassword(ctx context.Context, req *actions_pb.ForgotPasswordRequest) (*actions_pb.ForgotPasswordResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "ForgotPassword",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("Forgot Password requested")

	resp := &actions_pb.ForgotPasswordResponse{}

	makeError := func(st actions_pb.ForgotPasswordResponse_StatusCode, msg string) (*actions_pb.ForgotPasswordResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	email := req.Email

	message, err := models.PasswordReset(email)

	switch {
	case err == models.UnauthorizedError:
		return makeError(actions_pb.ForgotPasswordResponse_InvalidCredentialsError, "E-Mail not registered, Try registering first")
	case err == models.PragyanUserError:
		return makeError(actions_pb.ForgotPasswordResponse_PragyanUserError, "You have registered using Pragyan Account. Try changing Password on Pragyan Website")
	case err != nil:
		l.Errorf("Request failed due to: %+v", err)
		return makeError(actions_pb.ForgotPasswordResponse_InternalServerError, getInternalErrorMessage(err))
	}
	resp.StatusMessage = message

	return resp, nil
}

func (d *dalalActionService) ChangePassword(ctx context.Context, req *actions_pb.ChangePasswordRequest) (*actions_pb.ChangePasswordResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "Change Password",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("Change Password requested")

	resp := &actions_pb.ChangePasswordResponse{}

	makeError := func(st actions_pb.ChangePasswordResponse_StatusCode, msg string) (*actions_pb.ChangePasswordResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	tempPassword := req.TempPassword
	newPassword := req.NewPassword
	confirmPassword := req.ConfirmPassword

	message, err := models.ChangePassword(tempPassword, newPassword, confirmPassword)

	switch {
	case err == models.InvalidTemporaryPasswordError:
		return makeError(actions_pb.ChangePasswordResponse_InvalidTemporaryPasswordError, "Incorrect temporary password")
	case err == models.TemporaryPasswordExpiredError:
		return makeError(actions_pb.ChangePasswordResponse_TemporaryPasswordExpiredError, "Temporary Password Expired")
	case err == models.PasswordMismatchError:
		return makeError(actions_pb.ChangePasswordResponse_PasswordMismatchError, "Passwords don't match")
	case err != nil:
		l.Errorf("Request failed due to: %+v", err)
		return makeError(actions_pb.ChangePasswordResponse_InternalServerError, getInternalErrorMessage(err))
	}

	resp.StatusMessage = message

	return resp, nil
}
