package models

import (
	"testing"
	"fmt"
	"log"
	"reflect"

	"github.com/gemnasium/migrate/migrate"
	_ "github.com/gemnasium/migrate/driver/mysql"
	"gopkg.in/jarcoal/httpmock.v1"
	"github.com/Sirupsen/logrus"

	"github.com/thakkarparth007/dalal-street-server/utils"
)

func init() {
	utils.InitConfiguration("config_test.json")
	//utils.InitLogger()
	utils.Logger = logrus.New()
	utils.Logger.Level = logrus.DebugLevel
	InitModels()

	connStr := fmt.Sprintf("mysql://%s:%s@%s/%s",
		utils.Configuration.DbUser,
		utils.Configuration.DbPassword,
		utils.Configuration.DbHost,
		utils.Configuration.DbName,
	)

	allErrors, ok := migrate.UpSync(connStr, "../migrations")
	if !ok {
		log.Fatal(allErrors)
	}
}

func TestLogin(t *testing.T) {
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
		Id: 2,
		Email: "test@testmail.com",
		Name: "TestName",
		Cash: STARTING_CASH,
		Total: STARTING_CASH,
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
