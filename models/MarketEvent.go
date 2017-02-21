package models

import (
	"github.com/Sirupsen/logrus"
	models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
)

type MarketEvent struct {
	Id           uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	StockId      uint32 `gorm:"column:stockId;not null" json:"stock_id"`
	EmotionScore int32  `gorm:"column:emotionScore;not null" json:"emotion_score"`
	Headline     string `gorm:"column:headline;not null" json:"headline"`
	Text         string `gorm:"column:text" json:"text"`
	CreatedAt    string `gorm:"column:createdAt;not null" json:"created_at"`
}

func (MarketEvent) TableName() string {
	return "MarketEvents"
}

func (gMarketEvent *MarketEvent) ToProto() *models_proto.MarketEvent {
	pMarketEvent := &models_proto.MarketEvent{
		Id:           gMarketEvent.Id,
		StockId:      gMarketEvent.StockId,
		Headline:     gMarketEvent.Headline,
		Text:         gMarketEvent.Text,
		EmotionScore: gMarketEvent.EmotionScore,
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
		count = min(count, MARKET_EVENT_COUNT)
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
