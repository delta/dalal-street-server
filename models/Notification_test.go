package models

import (
	"github.com/thakkarparth007/dalal-street-server/utils/test"
	"testing"
)

func TestNotificationToProto(t *testing.T) {
	o := &Notification{
		Id:        2,
		UserId:    3,
		Text:      "Hello World",
		CreatedAt: "2017-02-09T00:00:00",
	}

	o_proto := o.ToProto()

	if !testutils.AssertEqual(t, o, o_proto) {
		t.Fatal("Converted value not equal")
	}
}
