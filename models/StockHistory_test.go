package models

import (
	"testing"

	"github.com/thakkarparth007/dalal-street-server/utils/test"
)

func TestStockHistoryToProto(t *testing.T) {
	o := &StockHistory{
		StockId:    3,
		StockPrice: 23,
		CreatedAt:  "2017-02-09T00:00:00",
		Interval:   3,
		Open:       500,
		High:       1000,
		Low:        100,
	}

	o_proto := o.ToProto()

	if !testutils.AssertEqual(t, o, o_proto) {
		t.Fatal("Converted value not equal")
	}
}
