package models

import (
	"testing"

//	"github.com/thakkarparth007/dalal-street-server/utils"
	"github.com/thakkarparth007/dalal-street-server/utils/test"
)

func TestTransactionToProto(t *testing.T) {
	lr := &Transaction{
		Id: 2,
		UserId: 20,
		StockId: 12,
		Type: OrderFillTransaction,
		StockQuantity: -20,
		Price: 300,
		Total: -300,
		CreatedAt: "2017-02-09T00:00:00",
	}

	lr_proto := lr.ToProto()

	if !testutils.AssertEqual(t, lr, lr_proto) {
		t.Fatal("Converted values not equal!")
	}
}
