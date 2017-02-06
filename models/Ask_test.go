package models

import (
	"github.com/thakkarparth007/dalal-street-server/utils/test"
	"testing"
)

func TestAskToProto(t *testing.T) {
	o := &Ask{
		Id:                     2,
		UserId:                 2,
		StockId:                3,
		Price:                  5,
		OrderType:              Market,
		StockQuantity:          20,
		StockQuantityFulfilled: 20,
		IsClosed:               true,
		CreatedAt:              "2017-02-09T00:00:00",
		UpdatedAt:              "2017-02-09T00:00:00",
	}

	o_proto := o.ToProto()

	if !testutils.AssertEqual(t, o, o_proto) {
		t.Fatal("Converted value not equal")
	}
}
