package actions

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"gopkg.in/jarcoal/httpmock.v1"

	//"github.com/thakkarparth007/dalal-street-server/utils/test"
	"github.com/thakkarparth007/dalal-street-server/models"
	actions_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/actions"
	models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
)

func TestLogin(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", "https://api.pragyan.org/event/login", httpmock.NewStringResponder(200, `{"status_code":200,"message": { "user_id": "2", "user_fullname": "TestName" }}`))

	req := &actions_proto.LoginRequest{
		Email:    "test@testmail.com",
		Password: "password",
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSession := NewMockSession(mockCtrl)
	mockSession.EXPECT().Set("userId", gomock.Any())
	mockSession.EXPECT().GetId().AnyTimes()

	resp := Login(mockSession, req)

	if err := resp.GetInvalidCredentialsError(); err != nil {
		t.Fatalf("Login returned invalid credentials error error: %s", err)
	} else if err := resp.GetInternalServerError(); err != nil {
		t.Fatalf("Login failed with internal server error: %s", err)
	} else if res := resp.GetResult(); res == nil {
		t.Fatalf("Login did not return error, nor did it return result: %+v", res)
	}

	res := resp.GetResult()

	exU := &models_proto.User{
		Id:        2,
		Email:     "test@testmail.com",
		Name:      "TestName",
		Cash:      models.STARTING_CASH,
		Total:     models.STARTING_CASH,
		CreatedAt: res.User.CreatedAt,
	}
	if reflect.DeepEqual(res.User, exU) != true {
		t.Fatalf("Expected Login to return %+v, instead, got %+v", exU, res.User)
	}

	//allErrors, ok = migrate.DownSync(connStr, "../migrations")
}
