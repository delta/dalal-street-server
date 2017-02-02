package models

type Stock struct {
	Id               uint32 `gorm:"primary_key;AUTO_INCREMENT"`
	ShortName        string `gorm:"column:shortName;not null"`
	FullName         string `gorm:"column:fullName;not null"`
	Description      string `gorm:"not null"`
	CurrentPrice     uint32 `gorm:"column:currentPrice;not null"`
	DayHigh          uint32 `gorm:"column:dayHigh;not null"`
	DayLow           uint32 `gorm:"column:dayLow;not null"`
	AllTimeHigh      uint32 `gorm:"column:allTimeHigh;not null"`
	AllTimeLow       uint32 `gorm:"column:allTimeLow;not null"`
	StocksInExchange uint32 `gorm:"column:stocksInExchange;not null"`
	UpOrDown         bool   `gorm:"column:upOrDown;not null"`
	CreatedAt        string `gorm:"column:createdAt;not null"`
	UpdatedAt        string `gorm:"column:updatedAt;not null"`
}

func (Stock) TableName() string {
	return "Stocks"
}
