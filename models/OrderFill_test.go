package models

import (
	"testing"

	"github.com/thakkarparth007/dalal-street-server/utils/test"
)

func TestOrderFillToProto(t *testing.T) {
	o := &OrderFill{
		TransactionId: 200,
		BidId:         345,
		AskId:         345,
	}

	o_proto := o.ToProto()

	if !testutils.AssertEqual(t, o, o_proto) {
		t.Fatal("Converted value not equal")
	}
}
