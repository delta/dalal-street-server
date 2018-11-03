package models

import (
	"testing"

	"github.com/delta/dalal-street-server/utils/test"
)

func TestNotificationToProto(t *testing.T) {
	o := &Notification{
		Id:          2,
		UserId:      3,
		Text:        "Hello World",
		IsBroadcast: true,
		CreatedAt:   "2017-02-09T00:00:00",
	}

	oProto := o.ToProto()

	if !testutils.AssertEqual(t, o, oProto) {
		t.Fatal("Converted value not equal")
	}
}
