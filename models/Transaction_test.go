package models

import (
	"testing"

	testutils "github.com/delta/dalal-street-server/utils/test"
	//	"github.com/delta/dalal-street-server/utils"
)

func TestTransactionToProto(t *testing.T) {
	tr := &Transaction{
		Id:                    2,
		UserId:                20,
		StockId:               12,
		Type:                  OrderFillTransaction,
		ReservedStockQuantity: 10,
		StockQuantity:         -20,
		Price:                 300,
		ReservedCashTotal:     10000,
		Total:                 -300,
		CreatedAt:             "2017-02-09T00:00:00",
	}

	trProto := tr.ToProto()

	if !testutils.AssertEqual(t, tr, trProto) {
		t.Fatal("Converted values not equal!")
	}
}
