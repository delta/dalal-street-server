package models

type Session struct {
	SessionId string `gorm:"column:sessionId;primary_key;not null"`
	Key       string `gorm:"column:key;not null"`
	Value     string `gorm:"column:value;not null"`
}

func (Session) TableName() string {
	return "Sessions"
}