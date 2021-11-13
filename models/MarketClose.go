package models

import (
	"fmt"
	"time"
)

var isMarketOpen = false

func OpenMarket(updateDayHighAndLow bool) error {
	isMarketOpen = true

	db := getDB()

	db.Exec("Update Config set isMarketOpen = true")

	gameStateStream := datastreamsManager.GetGameStateStream()
	g := &GameState{
		UserID: 0,
		Ms: &MarketState{
			IsMarketOpen: true,
		},
		GsType: MarketStateUpdate,
	}
	gameStateStream.SendGameStateUpdate(g.ToProto())

	SendPushNotification(0, PushNotification{
		Title:   "Message from Dalal Street! The market just opened.",
		Message: "The Market just opened! Click here to begin your trading.",
		LogoUrl: fmt.Sprintf("%v/static/dalalfavicon.png", config.BackendUrl),
	})

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

	gameStateStream := datastreamsManager.GetGameStateStream()
	g := &GameState{
		UserID: 0,
		Ms: &MarketState{
			IsMarketOpen: false,
		},
		GsType: MarketStateUpdate,
	}
	gameStateStream.SendGameStateUpdate(g.ToProto())

	if updatePreviousDayClose {
		return SetPreviousDayClose()
	}

	stopStockHistoryRecorder()

	return nil
}

func IsMarketOpen() bool {
	return isMarketOpen
}
