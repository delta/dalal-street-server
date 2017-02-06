package models

import (
	"reflect"
	"testing"

	"gopkg.in/jarcoal/httpmock.v1"

	"github.com/thakkarparth007/dalal-street-server/utils/test"
)

func Test_Login(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", "https://api.pragyan.org/event/login", httpmock.NewStringResponder(200, `{"status_code":200,"message": { "user_id": "2", "user_fullname": "TestName" }}`))

	u, err := Login("test@testmail.com", "password")
	if err != nil {
		t.Fatalf("Login returned an error: %s", err)
	}

	defer func() {
		db, err := DbOpen()
		if err != nil {
			t.Fatal("Failed opening DB for cleaning up test user")
		}
		defer db.Close()

		db.Delete(u)
	}()

	exU := &User{
		Id:        2,
		Email:     "test@testmail.com",
		Name:      "TestName",
		Cash:      STARTING_CASH,
		Total:     STARTING_CASH,
		CreatedAt: u.CreatedAt,
	}
	if reflect.DeepEqual(u, exU) != true {
		t.Fatalf("Expected Login to return %+v, instead, got %+v", exU, u)
	}

	_, err = Login("test@testmail.com", "TestName")
	if err != nil {
		t.Fatalf("Login failed: '%s'", err)
	}

	//allErrors, ok = migrate.DownSync(connStr, "../migrations")
}

func TestUserToProto(t *testing.T) {
	o := &User{
		Id:        2,
		Email:     "test@testmail.com",
		Name:      "test user",
		Cash:      10000,
		Total:     -200,
		CreatedAt: "2017-06-08T00:00:00",
	}

	o_proto := o.ToProto()

	if !testutils.AssertEqual(t, o, o_proto) {
		t.Fatal("Converted values not equal!")
	}

}
