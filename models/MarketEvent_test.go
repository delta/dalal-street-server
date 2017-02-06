package models

import (
	"github.com/thakkarparth007/dalal-street-server/utils/test"
	"testing"
)

func TestMarketEventToProto(t *testing.T) {
	o := &MarketEvent{
		Id:           2,
		StockId:      3,
		Text:         "Hello World",
		EmotionScore: -54,
		CreatedAt:    "2017-02-09T00:00:00",
	}

	o_proto := o.ToProto()

	if !testutils.AssertEqual(t, o, o_proto) {
		t.Fatal("Converted value not equal")
	}
}
