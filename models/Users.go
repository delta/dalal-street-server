package models

type Users struct {
	Id            uint32          `gorm:"primary_key;AUTO_INCREMENT"`
	PragyanId     uint32          `gorm:"column:pragyanId;not null"`
	Name          string          `gorm:"not null"`
	Cash          uint32          `gorm:"not null"`
	Total         uint32          `gorm:"not null"`
	CreatedAt     string          `gorm:"column:createdAt;not null"`
	BidsMade      []Bids          `gorm:"ForeignKey:userId"`
	Notifications []Notifications `gorm:"ForeignKey:userId"`
	Transactions  []Transactions  `gorm:"ForeignKey:userId"`
}

func (Users) TableName() string {
	return "Users"
}