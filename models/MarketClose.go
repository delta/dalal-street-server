package models

import (
	"fmt"

	"github.com/thakkarparth007/dalal-street-server/datastreams"
)

var isMarketOpen = false

func OpenMarket() {
	isMarketOpen = true

	db, err := DbOpen()
	if err != nil {
		return
	}
	defer db.Close()

	db.Exec("Update Config set isMarketOpen = true")

	notif := &Notification{
		Text: MARKET_IS_OPEN_HACKY_NOTIF,
	}
	datastreams.SendNotification(notif.ToProto())
}

func CloseMarket() {
	isMarketOpen = false

	db, err := DbOpen()
	if err != nil {
		return
	}
	defer db.Close()

	db.Exec("Update Config set isMarketOpen = false")

	notif := &Notification{
		Text: MARKET_IS_CLOSED_HACKY_NOTIF,
	}
	datastreams.SendNotification(notif.ToProto())
}

func IsMarketOpen() bool {
	return isMarketOpen
}

func init() {
	db, err := DbOpen()
	if err != nil {
		return
	}
	defer db.Close()

	resp := struct {
		Open bool
	}{}

	db.Raw("Select isMarketOpen as open from Config").Scan(&resp)
	fmt.Printf("resp %+v", resp)
	isMarketOpen = resp.Open
}
