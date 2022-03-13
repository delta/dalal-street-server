package actionservice

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/delta/dalal-street-server/models"
	actions_pb "github.com/delta/dalal-street-server/proto_build/actions"
	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/delta/dalal-street-server/session"
	"github.com/delta/dalal-street-server/utils"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func writeUserDetailsToLog(ctx context.Context) {
	var l = logger.WithFields(logrus.Fields{
		"method": "writeUserDetailsToLog",
	})

	userID := getUserId(ctx)

	peerDetails, ok := peer.FromContext(ctx)
	if ok {
		err := models.AddToGeneralLog(userID, "IP", peerDetails.Addr.String())
		if err != nil {
			l.Infof("Error while writing to databaes. Error: %+v", err)
		}
	} else {
		l.Infof("Failed to log peer details")
	}

	mD, ok := metadata.FromIncomingContext(ctx)
	if ok {
		userAgent := strings.Join(mD["user-agent"], " ")
		err := models.AddToGeneralLog(userID, "User-Agent", userAgent)
		if err != nil {
			l.Infof("Error while writing to databaes. Error: %+v", err)
		}
	} else {
		l.Infof("Failed to log user-agent")
	}
}

func (d *dalalActionService) Register(ctx context.Context, req *actions_pb.RegisterRequest) (*actions_pb.RegisterResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "Register",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
	})

	l.Infof("Register requested")

	resp := &actions_pb.RegisterResponse{}
	makeError := func(st actions_pb.RegisterResponse_StatusCode, msg string) (*actions_pb.RegisterResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	err := models.RegisterUser(req.GetEmail(), req.GetPassword(), req.GetFullName(), req.GetReferralCode())
	l.Errorf("Unable to register user due to : %v", err)
	if err == models.AlreadyRegisteredError {
		return makeError(actions_pb.RegisterResponse_AlreadyRegisteredError, "Already registered please Login")
	} else if err == models.InvalidReferralCodeError {
		return makeError(actions_pb.RegisterResponse_InvalidReferralCodeError, "Referral code is invalid")
	}
	if err != nil {
		return makeError(actions_pb.RegisterResponse_InternalServerError, getInternalErrorMessage(err))
	}

	l.Infof("Done")

	return resp, nil
}

func (d *dalalActionService) Login(ctx context.Context, req *actions_pb.LoginRequest) (*actions_pb.LoginResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "Login",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
	})

	l.Infof("Login requested")

	resp := &actions_pb.LoginResponse{}
	makeError := func(st actions_pb.LoginResponse_StatusCode, msg string) (*actions_pb.LoginResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	var (
		user            models.User
		err             error
		alreadyLoggedIn bool
	)

	sess := ctx.Value("session").(session.Session)
	if userId, ok := sess.Get("userId"); !ok {
		email := req.GetEmail()
		password := req.GetPassword()

		if email == "" || password == "" {
			return makeError(actions_pb.LoginResponse_InvalidCredentialsError, "Invalid Credentials")
		}

		user, err = models.Login(email, password)
	} else {
		alreadyLoggedIn = true
		userIdInt, err := strconv.ParseUint(userId, 10, 32)
		if err == nil {
			user, _ = models.GetUserCopy(uint32(userIdInt))
		}
	}

	switch {
	case err == models.UnauthorizedError:
		return makeError(actions_pb.LoginResponse_InvalidCredentialsError, "Incorrect username/password combination. Please use your Pragyan / Dalal Street credentials.")
	case err == models.NotRegisteredError:
		return makeError(actions_pb.LoginResponse_InvalidCredentialsError, "You have not registered for Dalal Street on the Pragyan / Dalal Street website")
	case err == models.UnverifiedUserError:
		return makeError(actions_pb.LoginResponse_EmailNotVerifiedError, "Please verify your mail to login")
	case err != nil:
		l.Errorf("Request failed due to: %+v", err)
		return makeError(actions_pb.LoginResponse_InternalServerError, getInternalErrorMessage(err))
	}

	l.Debugf("models.Login returned without error %+v", user)

	if !alreadyLoggedIn {
		if err := sess.Set("userId", strconv.Itoa(int(user.Id))); err != nil {
			l.Errorf("Request failed due to: %+v", err)
			return makeError(actions_pb.LoginResponse_InternalServerError, getInternalErrorMessage(err))
		}
	}

	writeUserDetailsToLog(ctx)

	l.Debugf("Session successfully set. UserId: %+v, Session id: %+v", user.Id, sess.GetID())

	stockList := models.GetAllStocks()
	stockListProto := make(map[uint32]*models_pb.Stock)
	for stockId, stock := range stockList {
		stockListProto[stockId] = stock.ToProto()
	}

	constantsMap := map[string]int32{
		"SHORT_SELL_BORROW_LIMIT": models.SHORT_SELL_BORROW_LIMIT,
		"BID_LIMIT":               models.BID_LIMIT,
		"ASK_LIMIT":               models.ASK_LIMIT,
		"BUY_LIMIT":               models.BUY_LIMIT,
		"MINIMUM_CASH_LIMIT":      models.MINIMUM_CASH_LIMIT,
		"BUY_FROM_EXCHANGE_LIMIT": models.BUY_FROM_EXCHANGE_LIMIT,
		"ORDER_PRICE_WINDOW":      models.ORDER_PRICE_WINDOW,
		"STARTING_CASH":           models.STARTING_CASH,
		"MORTGAGE_RETRIEVE_RATE":  models.MORTGAGE_RETRIEVE_RATE,
		"MORTGAGE_DEPOSIT_RATE":   models.MORTGAGE_DEPOSIT_RATE,
		"MARKET_EVENT_COUNT":      models.MARKET_EVENT_COUNT,
		"MY_ASK_COUNT":            models.MY_ASK_COUNT,
		"MY_BID_COUNT":            models.MY_BID_COUNT,
		"GET_NOTIFICATION_COUNT":  models.GET_NOTIFICATION_COUNT,
		"GET_TRANSACTION_COUNT":   models.GET_TRANSACTION_COUNT,
		"LEADERBOARD_COUNT":       models.LEADERBOARD_COUNT,
		"ORDER_FEE_PERCENT":       models.ORDER_FEE_PERCENT,
	}

	resp = &actions_pb.LoginResponse{
		SessionId:      sess.GetID(),
		User:           user.ToProto(),
		Constants:      constantsMap,
		IsMarketOpen:   models.IsMarketOpen(),
		VapidPublicKey: utils.GetConfiguration().PushNotificationVAPIDPublicKey,
	}

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalActionService) Logout(ctx context.Context, req *actions_pb.LogoutRequest) (*actions_pb.LogoutResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "Logout",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	l.Infof("Logout requested")

	sess := ctx.Value("session").(session.Session)
	userId := getUserId(ctx)
	models.Logout(userId)
	sess.Destroy()

	l.Infof("Request completed successfully")

	return &actions_pb.LogoutResponse{}, nil
}

func (d *dalalActionService) ResendVerificationEmail(ctx context.Context, req *actions_pb.ResendVerificationEmailRequest) (*actions_pb.ResendVerificationEmailResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "ResendVerificationEmail",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	l.Infof("Resned verification email requested")

	resp := &actions_pb.ResendVerificationEmailResponse{}
	makeError := func(st actions_pb.ResendVerificationEmailResponse_StatusCode, msg string) (*actions_pb.ResendVerificationEmailResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	if err := models.ResendVerificationEmail(req.GetEmail()); err != nil {
		l.Errorf("Got the error : %s", err)
		if err == models.MaximumEmailCountReached {
			return makeError(actions_pb.ResendVerificationEmailResponse_MaxEmailResendCountReached, "Maximum email limits reached")
		} else if err == models.UserNotFoundError {
			return makeError(actions_pb.ResendVerificationEmailResponse_MaxEmailResendCountReached, "Email not registered")
		} else if err == models.EmailAlreadyVerified {
			return makeError(actions_pb.ResendVerificationEmailResponse_MaxEmailResendCountReached, "Mail already verified, you can login now!")
		} else {
			return makeError(actions_pb.ResendVerificationEmailResponse_InternalServerError, getInternalErrorMessage(err))
		}
	}

	return &actions_pb.ResendVerificationEmailResponse{}, nil
}

func (d *dalalActionService) AddPhone(ctx context.Context, req *actions_pb.AddPhoneRequest) (*actions_pb.AddPhoneResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "AddPhone",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	l.Infof("Add Phone Requested")

	phoneNo := req.GetPhoneNumber()
	userId := getUserId(ctx)

	if phoneNo[0] != '+' {
		phoneNo = "+" + phoneNo
	}

	resp := &actions_pb.AddPhoneResponse{}
	makeError := func(st actions_pb.AddPhoneResponse_StatusCode, msg string) (*actions_pb.AddPhoneResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	if models.IsUserOTPBlocked(userId) {
		return makeError(actions_pb.AddPhoneResponse_UserOTPBlockedError, "We have detected an attempt to bruteforce OTP from your account, and has thus been permanently blocked.")
	}

	err := models.SendOTP(userId, phoneNo)

	if err == models.PhoneNoAlreadyTakenError {
		return makeError(actions_pb.AddPhoneResponse_PhoneNoAlreadyTakenError, "Phone number already in use.")
	} else if err == models.SendSMSError || err == models.InternalServerError {
		return makeError(actions_pb.AddPhoneResponse_InternalServerError, getInternalErrorMessage(err))
	}

	return makeError(actions_pb.AddPhoneResponse_OK, "OTP sent successfully")
}

func (d *dalalActionService) VerifyPhone(ctx context.Context, req *actions_pb.VerifyOTPRequest) (*actions_pb.VerifyOTPResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "VerifyPhone",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	l.Infof("Verify OTP Requested")

	otpNo := req.GetOtp()
	phone := req.GetPhone()
	userId := getUserId(ctx)

	if phone[0] != '+' {
		phone = "+" + phone
	}

	resp := &actions_pb.VerifyOTPResponse{}
	makeError := func(st actions_pb.VerifyOTPResponse_StatusCode, msg string) (*actions_pb.VerifyOTPResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	if models.IsUserOTPBlocked(userId) {
		return makeError(actions_pb.VerifyOTPResponse_UserOTPBlockedError, "We have detected an attempt to bruteforce OTP from your account, and has thus been permanently blocked.")
	}

	err := models.VerifyOTP(userId, otpNo, phone)

	if err == models.OTPExpiredError {
		return makeError(actions_pb.VerifyOTPResponse_OTPExpiredError, "OTP expired, please verify with new OTP.")
	} else if err == models.OTPMismatchError {
		return makeError(actions_pb.VerifyOTPResponse_OTPMismatchError, "Invalid OTP entered")
	} else if err == models.InvalidPhoneNumberError || err == models.InternalServerError {
		return makeError(actions_pb.VerifyOTPResponse_InternalServerError, getInternalErrorMessage(err))
	}

	if userCash, err := models.AddExtraCredit(userId); err != nil {
		// Already verified referral when registering, so only internal-error possible
		return makeError(actions_pb.VerifyOTPResponse_InternalServerError, getInternalErrorMessage(err))
	} else {
		resp.UserCash = userCash
	}

	return makeError(actions_pb.VerifyOTPResponse_OK, "OTP verification successful.")
}
