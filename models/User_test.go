package models

import (
	"testing"
	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/gemnasium/migrate/migrate"
	_ "github.com/gemnasium/migrate/driver/mysql"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

func TestCreateUser(t *testing.T) {
	utils.InitConfiguration("config_test.json")
	utils.InitLogger()
	InitUsers()

	connStr := fmt.Sprintf("mysql://%s:%s@%s/%s",
		utils.Configuration.DbUser,
		utils.Configuration.DbPassword,
		utils.Configuration.DbHost,
		utils.Configuration.DbName,
	)

	allErrors, ok := migrate.UpSync(connStr, "../migrations")
	if !ok {
		t.Fatal(allErrors)
	}

	u, err := CreateUser("test@testmail.com", "TestName")
	assert.Equal(t, true, nil == err, "CreateUser should not return an error")
	assert.Equal(t, true, nil != u, "CreateUser should return pointer to created user")
	assert.Equal(t, "test@testmail.com", u.Email, "CreateUser should set user.Email correctly")
	assert.Equal(t, "TestName", u.Name, "CreateUser should set user.Name correctly")

	// ugliness mandatory. If STARTING_CASH is passed as second argument
	// to assert.Equal() then that gets interpreted as int even though
	// it is a constant. u.Cash and u.Total are int32
	assert.Equal(t, true, STARTING_CASH == u.Cash, "User's cash must be set to STARTING_CASH")
	assert.Equal(t, true, STARTING_CASH == u.Total, "User's total must be set to STARTING_CASH")
	assert.Equal(t, true, 0 < u.Id, "User's id should be valid")

	_, err = CreateUser("test@testmail.com", "TestName")
	assert.Equal(t, true, nil != err, "Error should not be nil on duplicate CreateUser")

	db, err := DbOpen()
	if err != nil {
		t.Fatal("Failed opening DB for cleaning up test user")
	}
	defer db.Close()

	db.Delete(u)
	//allErrors, ok = migrate.DownSync(connStr, "../migrations")
}
