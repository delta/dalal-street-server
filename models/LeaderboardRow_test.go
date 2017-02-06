package models

import (
	"testing"

	"github.com/thakkarparth007/dalal-street-server/utils/test"
)

func TestLeaderboardRowToProto(t *testing.T) {
	lr := &LeaderboardRow{
		Id:         2,
		UserId:     5,
		Rank:       1,
		Cash:       1000,
		Debt:       10,
		StockWorth: -50,
		TotalWorth: -300,
	}

	lr_proto := lr.ToProto()

	if !testutils.AssertEqual(t, lr, lr_proto) {
		t.Fatal("Converted values not equal!")
	}
}
