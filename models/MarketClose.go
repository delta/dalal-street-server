package models

import "time"

var isMarketOpen = false

func OpenMarket(updateDayHighAndLow bool) error {
	isMarketOpen = true

	db := getDB()

	db.Exec("Update Config set isMarketOpen = true")

	notif := &Notification{
		Text: MARKET_IS_OPEN_HACKY_NOTIF,
	}
	notificationsStream := datastreamsManager.GetNotificationsStream()
	notificationsStream.SendNotification(notif.ToProto())

	if updateDayHighAndLow {
		return SetDayHighAndLow()
	}

	go startStockHistoryRecorder(time.Minute)

	return nil
}

func CloseMarket(updatePreviousDayClose bool) error {
	isMarketOpen = false

	db := getDB()

	db.Exec("Update Config set isMarketOpen = false")

	notif := &Notification{
		Text: MARKET_IS_CLOSED_HACKY_NOTIF,
	}

	notificationsStream := datastreamsManager.GetNotificationsStream()
	notificationsStream.SendNotification(notif.ToProto())

	if updatePreviousDayClose {
		return SetPreviousDayClose()
	}

	stopStockHistoryRecorder()

	return nil
}

func IsMarketOpen() bool {
	return isMarketOpen
}
