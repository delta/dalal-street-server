package models

type Registration struct {
	Id         uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	Email      string `gorm:"unique;not null" json:"email"`
	Name       string `gorm:"column:fullName;not null" json:"full_name"`
	UserName   string `gorm:"column:userName;not null" json:"user_name"`
	UserId     uint32 `gorm:"column:userId;not null" json:"user_id"`
	IsPragyan  bool   `gorm:"column:isPragyan;not null" json:"is_pragyan"`
	Password   string `gorm:"column:password;not null" json:"password"`
	Country    string `gorm:"not null" json:"password"`
	IsVerified bool   `gorm:"column:isVerified;not null" json:"is_verified"`
}

func (Registration) TableName() string {
	return "Registrations"
}
