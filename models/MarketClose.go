package models

var isMarketOpen = false

func OpenMarket() {
	isMarketOpen = true

	db := getDB()

	db.Exec("Update Config set isMarketOpen = true")

	notif := &Notification{
		Text: MARKET_IS_OPEN_HACKY_NOTIF,
	}
	notificationsStream := datastreamsManager.GetNotificationsStream()
	notificationsStream.SendNotification(notif.ToProto())
}

func CloseMarket() {
	isMarketOpen = false

	db := getDB()

	db.Exec("Update Config set isMarketOpen = false")

	notif := &Notification{
		Text: MARKET_IS_CLOSED_HACKY_NOTIF,
	}

	notificationsStream := datastreamsManager.GetNotificationsStream()
	notificationsStream.SendNotification(notif.ToProto())
}

func IsMarketOpen() bool {
	return isMarketOpen
}
