package models

import (
	models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
)

type MarketEvent struct {
	Id           uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	StockId      uint32 `gorm:"column:stockId;not null" json:"stock_id"`
	EmotionScore int32  `gorm:"column:emotionScore;not null" json:"emotion_score"`
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
		Text:         gMarketEvent.Text,
		EmotionScore: gMarketEvent.EmotionScore,
		CreatedAt:    gMarketEvent.CreatedAt,
	}
	return pMarketEvent
}
