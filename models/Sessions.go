package models

type Sessions struct {
	SessionId string `gorm:"column:sessionId;primary_key;not null"`
	Key       string `gorm:"column:key;not null"`
	Value     string `gorm:"column:value;not null"`
}

func (Sessions) TableName() string {
	return "Sessions"
}