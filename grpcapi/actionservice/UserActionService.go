package actionservice

import (
	"fmt"

	"github.com/delta/dalal-street-server/models"
	actions_pb "github.com/delta/dalal-street-server/proto_build/actions"
	"github.com/delta/dalal-street-server/session"
	"github.com/delta/dalal-street-server/utils"
	"github.com/sirupsen/logrus"
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

func (d dalalActionService) GetDailyLeaderboard(ctx context.Context, req *actions_pb.GetDailyLeaderboardRequest) (*actions_pb.GetDailyLeaderboardResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetDailyLeaderboard",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("GetDailyLeaderboard requested")

	resp := &actions_pb.GetDailyLeaderboardResponse{}

	userId := getUserId(ctx)
	startingId := req.StartingId
	count := req.Count

	leaderboard, currentUserRow, totalUsers, err := models.GetDailyLeaderboard(userId, startingId, count)
	if err != nil {
		l.Errorf("Request failed due to: %+v", err)
		resp.StatusCode = actions_pb.GetDailyLeaderboardResponse_InternalServerError
		resp.StatusMessage = getInternalErrorMessage(err)
		return resp, nil
	}

	resp.MyRank = currentUserRow.Rank
	resp.MyTotalWorth = currentUserRow.TotalWorth
	resp.TotalUsers = totalUsers
	for _, leaderboardRow := range leaderboard {
		resp.RankList = append(resp.RankList, leaderboardRow.ToProto())
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
		return makeError(actions_pb.ForgotPasswordResponse_InvalidCredentialsError, "E-Mail not registered or registered through Pragyan.If registered through pragyan visit pragyan website to change password")
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

func (d *dalalActionService) GetReferralCode(ctx context.Context, req *actions_pb.GetReferralCodeRequest) (*actions_pb.GetReferralCodeResponse, error) {

	var l = logger.WithFields(logrus.Fields{
		"method":        "GetReferralCode",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("Get ReferralCode requested")

	resp := &actions_pb.GetReferralCodeResponse{}

	makeError := func(st actions_pb.GetReferralCodeResponse_StatusCode, msg string) (*actions_pb.GetReferralCodeResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	usrEmail := req.Email
	referralCode, err := models.GetReferralCode(usrEmail)

	l.Debugf("Referral Code has been generated, %v", referralCode)

	switch {
	case err == models.UserNotFoundError:
		return makeError(actions_pb.GetReferralCodeResponse_InvalidUserError, "user does not exist")
	case err != nil:
		l.Errorf("Request failed due to : %v\n", err)
		return makeError(actions_pb.GetReferralCodeResponse_InternalServerError, getInternalErrorMessage(err))
	default:
		resp.ReferralCode = referralCode
		resp.StatusMessage = "success"
		return resp, nil
	}
}

<<<<<<< HEAD
func (d *dalalActionService) GetDailyChallenges(ctx context.Context, req *actions_pb.GetDailyChallengesRequest) (*actions_pb.GetDailyChallengesResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetDailyChallenges",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	l.Debugf("GetDailyChallenges Requested")

	res := &actions_pb.GetDailyChallengesResponse{}

	makeError := func(st actions_pb.GetDailyChallengesResponse_StatusCode, msg string) (*actions_pb.GetDailyChallengesResponse, error) {
		res.StatusCode = st
		res.StatusMessage = msg
		return res, nil
	}
	marketday := req.MarketDay

	if marketday <= 0 {
		return makeError(actions_pb.GetDailyChallengesResponse_InvalidRequestError, "invalid request, marketday not supported")
	}
	DailyChallenges, err := models.GetDailyChallenges(marketday)

	if err != nil {
		l.Errorf("failed to load daily challenges")
		return makeError(actions_pb.GetDailyChallengesResponse_InternalServerError, getInternalErrorMessage(err))
	}

	for _, challenge := range DailyChallenges {
		res.DailyChallenges = append(res.DailyChallenges, challenge.ToProto())
	}

	res.StatusCode = actions_pb.GetDailyChallengesResponse_OK
	res.StatusMessage = "Done"
	return res, nil
}

func (d *dalalActionService) GetMyUserState(ctx context.Context, req *actions_pb.GetMyUserStateRequest) (*actions_pb.GetMyUserStateResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetMyUserState",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Debugf("GetMyUserState requested")

	res := &actions_pb.GetMyUserStateResponse{}

	makeError := func(st actions_pb.GetMyUserStateResponse_StatusCode, msg string) (*actions_pb.GetMyUserStateResponse, error) {
		res.StatusCode = st
		res.StatusMessage = msg
		return res, nil
	}

	userId := getUserId(ctx)

	userState, err := models.GetUserState(userId, req.ChallengeId)

	if err != nil {
		l.Errorf("Error %+e", err)
		return makeError(actions_pb.GetMyUserStateResponse_InternalServerError, getInternalErrorMessage(err))
	}
	res.UserState = userState.ToProto()
	res.StatusCode = actions_pb.GetMyUserStateResponse_OK
	res.StatusMessage = "Done"

	l.Debugf("Done")

	return res, nil

}

func (d *dalalActionService) GetMyReward(ctx context.Context, req *actions_pb.GetMyRewardRequest) (*actions_pb.GetMyRewardResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "GetMyReward",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Debugf("GetMyUserState requested")

	res := &actions_pb.GetMyRewardResponse{}

	makeError := func(st actions_pb.GetMyRewardResponse_StatusCode, msg string) (*actions_pb.GetMyRewardResponse, error) {
		res.StatusCode = st
		res.StatusMessage = msg
		return res, nil
	}

	userId := getUserId(ctx)

	reward, err := models.GetMyReward(req.UserStateId, userId)

	if err == models.InternalServerError {
		return makeError(actions_pb.GetMyRewardResponse_InternalServerError, getInternalErrorMessage(err))
	} else if err == models.InvalidUserError {
		return makeError(actions_pb.GetMyRewardResponse_InvalidUserError, "Invalid user")
	} else if err == models.InvalidCerdentialError {
		return makeError(actions_pb.GetMyRewardResponse_InvalidCerdentialError, "better luck next time")
	} else if err == models.InvalidRequestError {
		return makeError(actions_pb.GetMyRewardResponse_InvalidRequestError, "Invalid request")
	}

	res.Reward = reward
	res.StatusCode = actions_pb.GetMyRewardResponse_OK
	res.StatusMessage = "Done"

	return res, nil

}

func (d *dalalActionService) GetDailyChallengeConfig(ctx context.Context, req *actions_pb.GetDailyChallengeConfigRequest) (*actions_pb.GetDailyChallengeConfigResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetDailyChallengeConfig",
	})

	l.Debugf("GetDailyChallengeConfig Requested")

	res := &actions_pb.GetDailyChallengeConfigResponse{}

	makeError := func(st actions_pb.GetDailyChallengeConfigResponse_StatusCode, msg string) (*actions_pb.GetDailyChallengeConfigResponse, error) {
		res.StatusCode = st
		res.StatusMessage = msg
		return res, nil
	}

	config, totalMarketDays, err := models.GetDailyChallengeConfig()

	if err != nil {
		return makeError(actions_pb.GetDailyChallengeConfigResponse_InternalServerError, getInternalErrorMessage(err))
	}

	res.MarketDay = config.MarketDay
	res.IsDailyChallengOpen = config.IsDailyChallengeOpen
	res.TotalMarketDays = totalMarketDays
	res.StatusCode = actions_pb.GetDailyChallengeConfigResponse_OK
	res.StatusMessage = "Done"

	return res, nil
=======
func (d *dalalActionService) AddUserSubscription(ctx context.Context,req *actions_pb.AddUserSubscriptionRequest) (*actions_pb.AddUserSubscriptionResponse,error) {

	var l = logger.WithFields(logrus.Fields{
		"method":        "AddUserSubscription",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})
	l.Infof("Add User Subscription requsted")

	resp := &actions_pb.AddUserSubscriptionResponse{}

	makeError := func(st actions_pb.AddUserSubscriptionResponse_StatusCode, msg string) (*actions_pb.AddUserSubscriptionResponse,error) {
		resp.StatusCode = st;
		resp.StatusMessage = msg;
		return resp, nil
	}

	email := req.Email
	// data contains endpoint and keys in string format
	data := req.Data

	err := models.AddUserSubscription(email,data) 

	switch  {
	case err == models.UserNotFoundError:
		return makeError(actions_pb.AddUserSubscriptionResponse_InvalidUserError , "User does not exist")
	case err != nil:
		l.Errorf("Request failed due to : %v\n", err)
		return makeError(actions_pb.AddUserSubscriptionResponse_InternalServerError, getInternalErrorMessage(err))
	}	
	return resp,nil;
>>>>>>> d6c020f ([Add]: rpc for storing user subscription details)

}
