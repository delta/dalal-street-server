package models

import (
	"github.com/Sirupsen/logrus"
	"github.com/thakkarparth007/dalal-street-server/proto_build/models"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

type MarketEvent struct {
	Id           uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	StockId      uint32 `gorm:"column:stockId;not null" json:"stock_id"`
	EmotionScore int32  `gorm:"column:emotionScore;not null" json:"emotion_score"`
	Headline     string `gorm:"column:headline;not null" json:"headline"`
	Text         string `gorm:"column:text" json:"text"`
	IsGlobal     bool   `gorm:"column:isGlobal" json:"is_global"`
	CreatedAt    string `gorm:"column:createdAt;not null" json:"created_at"`
}

func (MarketEvent) TableName() string {
	return "MarketEvents"
}

func (gMarketEvent *MarketEvent) ToProto() *models_pb.MarketEvent {
	pMarketEvent := &models_pb.MarketEvent{
		Id:           gMarketEvent.Id,
		StockId:      gMarketEvent.StockId,
		Headline:     gMarketEvent.Headline,
		Text:         gMarketEvent.Text,
		EmotionScore: gMarketEvent.EmotionScore,
		IsGlobal:     gMarketEvent.IsGlobal,
		CreatedAt:    gMarketEvent.CreatedAt,
	}
	return pMarketEvent
}

func GetMarketEvents(lastId, count uint32) (bool, []*MarketEvent, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetMarketEvents",
		"lastId": lastId,
		"count":  count,
	})

	l.Infof("Attempting to get market events")

	db, err := DbOpen()
	if err != nil {
		return true, nil, err
	}
	defer db.Close()

	var marketEvents []*MarketEvent

	//set default value of count if it is zero
	if count == 0 {
		count = MARKET_EVENT_COUNT
	} else {
		count = utils.MinInt(count, MARKET_EVENT_COUNT)
	}

	//get latest events if lastId is zero
	if lastId != 0 {
		db = db.Where("id <= ?", lastId)
	}
	if err := db.Order("id desc").Limit(count).Find(&marketEvents).Error; err != nil {
		return true, nil, err
	}

	var moreExists = len(marketEvents) >= int(count)
	l.Infof("Successfully fetched market events")
	return moreExists, marketEvents, nil
}

func AddMarketEvent(stockId uint32, headline, text string, isGlobal bool) error {
	var l = logger.WithFields(logrus.Fields{
		"method":         "AddMarketEvent",
		"param_stockId":  stockId,
		"param_headline": headline,
		"param_text":     text,
		"param_isGlobal": isGlobal,
	})

	l.Infof("Attempting")

	db, err := DbOpen()
	if err != nil {
		return err
	}
	defer db.Close()

	me := &MarketEvent{
		StockId:  stockId,
		Headline: headline,
		Text:     text,
		IsGlobal: isGlobal,
	}

	if err := db.Save(me).Error; err != nil {
		l.Error(err)
		return err
	}

	l.Infof("Done")

	marketEventsStream := datastreamsManager.GetMarketEventsStream()
	marketEventsStream.SendMarketEvent(me.ToProto())

	return nil
}
