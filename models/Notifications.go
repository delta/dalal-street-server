package models

type Notifications struct {
	Id        uint32 `gorm:"primary_key;AUTO_INCREMENT"`
	UserId    uint32 `gorm:"column:userId;not null"`
	Text      string `gorm:"column:text"`
	CreatedAt string `gorm:"column:createdAt;not null"`
}

func (Notifications) TableName() string {
	return "Notifications"
}