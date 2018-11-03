package models

import (
	"testing"

	//	"github.com/delta/dalal-street-server/utils"

	"github.com/delta/dalal-street-server/utils/test"
)

func TestTransactionToProto(t *testing.T) {
	tr := &Transaction{
		Id:            2,
		UserId:        20,
		StockId:       12,
		Type:          OrderFillTransaction,
		StockQuantity: -20,
		Price:         300,
		Total:         -300,
		CreatedAt:     "2017-02-09T00:00:00",
	}

	trProto := tr.ToProto()

	if !testutils.AssertEqual(t, tr, trProto) {
		t.Fatal("Converted values not equal!")
	}
}
