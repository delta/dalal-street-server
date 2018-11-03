package models

import (
	"testing"

	"github.com/delta/dalal-street-server/utils/test"
)

func TestOrderFillToProto(t *testing.T) {
	o := &OrderFill{
		TransactionId: 200,
		BidId:         345,
		AskId:         345,
	}

	oProto := o.ToProto()

	if !testutils.AssertEqual(t, o, oProto) {
		t.Fatal("Converted value not equal")
	}
}
