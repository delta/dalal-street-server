package models

import (
	"github.com/thakkarparth007/dalal-street-server/socketapi/datastreams"
)

var isMarketOpen = false

func OpenMarket() {
	isMarketOpen = true
	notif := &Notification{
		Text: MARKET_IS_OPEN_HACKY_NOTIF,
	}
	datastreams.SendNotification(notif.ToProto())
}

func CloseMarket() {
	isMarketOpen = false
	notif := &Notification{
		Text: MARKET_IS_CLOSED_HACKY_NOTIF,
	}
	datastreams.SendNotification(notif.ToProto())
}

func IsMarketOpen() bool {
	return isMarketOpen
}
