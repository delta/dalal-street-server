package models

type MarketEvent struct {
	Id           uint32 `gorm:"primary_key;AUTO_INCREMENT"`
	StockId      uint32 `gorm:"column:stockId;not null"`
	EmotionScore int32  `gorm:"column:emotionScore;not null"`
	Text         string `gorm:"column:text"`
	CreatedAt    string `gorm:"column:createdAt;not null"`
}

func (MarketEvent) TableName() string {
	return "MarketEvents"
}