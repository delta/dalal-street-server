package models

import (
	"github.com/thakkarparth007/dalal-street-server/utils/test"
	"testing"
)

func TestStockToProto(t *testing.T) {
	o := &Stock{
		Id:               23,
		ShortName:        "zold",
		FullName:         "PastCry",
		Description:      "This Stock is a stock :P",
		CurrentPrice:     200,
		DayHigh:          300,
		DayLow:           100,
		AllTimeHigh:      400,
		AllTimeLow:       90,
		StocksInExchange: 123,
		StocksInMarket:   234,
		UpOrDown:         true,
		CreatedAt:        "2017-02-09T00:00:00",
		UpdatedAt:        "2017-02-09T00:00:00",
	}

	o_proto := o.ToProto()

	if !testutils.AssertEqual(t, o, o_proto) {
		t.Fatal("Converted value not equal")
	}
}
