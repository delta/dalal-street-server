package models

import (
	"testing"

	"github.com/thakkarparth007/dalal-street-server/utils/test"
)

func TestMortgageDetailToProto(t *testing.T) {
	o := &MortgageDetail{
		Id:           1,
		UserId:       20,
		StockId:      300,
		StocksInBank: 10,
	}

	o_proto := o.ToProto()

	if !testutils.AssertEqual(t, o, o_proto) {
		t.Fatal("Converted values not equal!")
	}
}
