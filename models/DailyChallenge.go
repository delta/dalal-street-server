package models

import (
	"errors"
	"fmt"
	"sync"

	"github.com/jinzhu/gorm"

	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/sirupsen/logrus"
)

// MarketDay
// ischallengeOpen for now initialized as variable
// TODO: store in db
var (
	MarketDay       uint32 = 1
	IsChallengeOpen bool   = false
)

var wg sync.WaitGroup

// DailyChallenge model
type DailyChallenge struct {
	Id            uint32 `gorm:"column:id;primary_key;AUTO_INCREMENT" json:"id"`
	MarketDay     uint32 `gorm:"column:marketDay;not null" json:"market_day"`
	ChallengeType string `gorm:"column:challengeType;not null" json:"challenge_type"`
	Value         uint64 `gorm:"column:value;not null" json:"value"`
	StockId       uint32 `gorm:"column:stockId; default null" json:"stock_id"`
}

//UserState model
type UserState struct {
	ChallengeId  uint32 `gorm:"column:challengeId;not null" json:"challenge_id"`
	UserId       uint32 `gorm:"column:userId;not null" json:"user_id"`
	MarketDay    uint32 `gorm:"column:marketDay;not null" json:"market_day"`
	InitialValue int64  `gorm:"column:initialValue;not null" json:"initial_value"`
	FinalValue   int64  `gorm:"column:finalValue;default null" json:"final_value"`
	IsCompleted  bool   `gorm:"column:isCompleted;default false" json:"is_completed"`
}

type userStateQueryData struct {
	UserId     uint32
	Cash       uint64
	StockWorth int64
	Total      int64
}

type specificStockUserEntry struct {
	UserId        uint32
	StockQuantity int64
}

func (DailyChallenge) TableName() string {
	return "DailyChallenge"
}

func (UserState) TableName() string {
	return "UserState"
}

func (d *DailyChallenge) ToProto() *models_pb.DailyChallenge {
	pDailyChallenge := &models_pb.DailyChallenge{
		ChallengeId:   d.Id,
		MarketDay:     d.MarketDay,
		ChallengeType: d.ChallengeType,
		Value:         d.Value,
		StockId:       d.StockId,
	}
	return pDailyChallenge
}

//GetDailyChallenges returns challenges as array for a given market day
func GetDailyChallenges(marketDay uint32) ([]*DailyChallenge, error) {

	l := logger.WithFields(logrus.Fields{
		"method":    "GetDailyChallenges",
		"marketDay": MarketDay,
	})

	var dailyChallenges []*DailyChallenge

	l.Infof("Attempting to get dailyChallenges")

	db := getDB()

	if err := db.Table("DailyChallenge").Where("marketDay = ?", marketDay).Find(&dailyChallenges).Error; err != nil {
		l.Errorf("error while querying from db")
		return nil, err
	}

	l.Infof("Successfully fetched dailyChallenges")
	return dailyChallenges, nil
}

//AddDailyChallenge add daily challenge to db, only Admin can invoke this function
func AddDailyChallenge(value uint64, marketDay uint32, stockId uint32, challengeType string) error {
	l := logger.WithFields(logrus.Fields{
		"method":              "AddDailyChallenge",
		"param_day":           MarketDay,
		"param_value":         value,
		"param_stockid":       stockId,
		"param_challengeType": challengeType,
	})

	db := getDB()

	l.Infof("Attempting to save daily challenge")

	dailyChallenge := &DailyChallenge{
		MarketDay:     marketDay,
		ChallengeType: challengeType,
		Value:         value,
		StockId:       stockId,
	}

	if stockId == 0 {
		if err := db.Table("DailyChallenge").Omit("StockId").Save(dailyChallenge).Error; err != nil {
			l.Error(err)
			return err
		}
	} else {
		if err := db.Table("DailyChallenge").Save(dailyChallenge).Error; err != nil {
			l.Error(err)
			return err
		}
	}

	l.Infof("successfully added daily challenge")

	return nil
}

//OpenDailyChallenge opens dailyChallenge
//saves initial user state depending upon the challengetype in db for later computation while Closing dailyChallenge
//Admin can only invoke this function
//TODO: use go concurrency  to save initial userstate effciently
func OpenDailyChallenge() error {
	l := logger.WithFields(logrus.Fields{
		"method":     "OpenDailyChallenge",
		"market_day": MarketDay,
	})

	l.Infof("OpenChallenge Requested!")

	IsChallengeOpen = true

	dailyChallenges, err := GetDailyChallenges(MarketDay)

	if err != nil {
		l.Error(err)
		return err
	}

	if err := saveUsersState(dailyChallenges); err != nil {
		l.Error(err)
		return err
	}

	l.Infof("succesfully saved userstate for DailyChallenges")

	return nil
}

//CloseDailyChallenge closes dailychallenge and computes whether all the users completed them
//and reward the user who have completed their challenges
func CloseDailyChallenge() error {
	//TODO: close DailyChallenge
	// and compute dailychallenges
	IsChallengeOpen = false

	return nil
}

//saveUsersState saves registered users cash,stockworth,Networth,specificstock quantity based on challenge type
//invoked inside OpenDailyChallenge
func saveUsersState(c []*DailyChallenge) error {
	l := logger.WithFields(logrus.Fields{
		"method": "saveUsersState",
	})

	l.Infof("Attempting to save user state")

	var queryResults []userStateQueryData

	db := getDB()

	//begin transaction
	tx := db.Begin()

	if err := tx.Error; err != nil {
		l.Error(err)
		return err
	}

	//query to get cash,Stockworth,NetWorth for all the non-blocked users
	query := fmt.Sprintf(`
	SELECT U.id as user_id, U.cash + U.reservedCash as cash,
	 ifNull((SUM(cast(S.currentPrice AS signed) * cast(T.stockQuantity AS signed)) + SUM(cast(S.currentPrice AS signed) * cast(T.reservedStockQuantity AS signed)) ),0) AS stock_worth,
	 ifnull((U.cash + U.reservedCash + SUM(cast(S.currentPrice AS signed) * cast(T.stockQuantity AS signed)) + SUM(cast(S.currentPrice AS signed) * cast(T.reservedStockQuantity AS signed))),U.cash) AS total
	 FROM
	Users U LEFT JOIN Transactions T ON U.id = T.userId LEFT JOIN Stocks S ON T.stockId = S.id WHERE U.blockCount < %d GROUP BY U.id;`, 3)

	if err := tx.Raw(query).Scan(&queryResults).Error; err != nil {
		l.Errorf("error when fetching userSate query data %+e", err)
		return err
	}

	for _, challenge := range c {

		switch challenge.ChallengeType {

		case "Cash":
			var userStateEntry *UserState

			for _, u := range queryResults {
				userStateEntry = &UserState{
					ChallengeId:  challenge.Id,
					UserId:       u.UserId,
					MarketDay:    challenge.MarketDay,
					InitialValue: int64(u.Cash),
				}

				if err := tx.Table("UserState").Omit("FinalValue", "Iscompleted").Save(userStateEntry).Error; err != nil {
					l.Errorf("failed Updating userState cash Challenge type %+e", err)
					tx.Rollback()
					return err
				}
			}

		case "NetWorth":
			var userStateEntry *UserState

			for _, u := range queryResults {
				userStateEntry = &UserState{
					ChallengeId:  challenge.Id,
					UserId:       u.UserId,
					MarketDay:    challenge.MarketDay,
					InitialValue: u.Total,
				}

				if err := tx.Table("UserState").Omit("FinalValue", "Iscompleted").Save(userStateEntry).Error; err != nil {
					l.Errorf("failed Updating userState NetWorth Challenge type %+e", err)
					tx.Rollback()
					return err
				}
			}

		case "StockWorth":
			var userStateEntry *UserState

			for _, u := range queryResults {
				userStateEntry = &UserState{
					ChallengeId:  challenge.Id,
					UserId:       u.UserId,
					MarketDay:    challenge.MarketDay,
					InitialValue: u.StockWorth,
				}

				if err := tx.Table("UserState").Omit("FinalValue", "Iscompleted").Save(userStateEntry).Error; err != nil {
					l.Errorf("failed Updating userState StockWorth Challenge type %+e", err)
					tx.Rollback()
					return err
				}
			}

		case "SpecificStock":
			var userStateEntry *UserState

			result, err := getSpecificStocksEntry(challenge.StockId, tx)

			if err != nil {
				l.Error(err)
				tx.Rollback()
				return err
			}

			for _, u := range result {
				userStateEntry = &UserState{
					ChallengeId:  challenge.Id,
					UserId:       u.UserId,
					MarketDay:    challenge.MarketDay,
					InitialValue: u.StockQuantity,
				}

				if err := tx.Table("UserState").Omit("FinalValue", "Iscompleted").Save(userStateEntry).Error; err != nil {
					l.Errorf("failed Updating userState SpecificStockType Challenge type %+e", err)
					tx.Rollback()
					return err
				}
			}

		default:
			l.Error("challenge type not supported")
			return errors.New("challenge type not supported")
		}

	}
	//commit transaction
	if err := tx.Commit().Error; err != nil {
		l.Error(err)
		return err
	}

	return nil
}

//getSpecificStocksEntry returns rows of userid and stockQuantity for an stockId
func getSpecificStocksEntry(stockId uint32, tx *gorm.DB) ([]specificStockUserEntry, error) {

	l := logger.WithFields(logrus.Fields{
		"method":   "getSpecificStocksEntry",
		"stock_id": stockId,
	})

	l.Infof("getSpecificStockEntry requested")

	var results []specificStockUserEntry

	query := fmt.Sprintf(`SELECT U.id AS user_id,
	IFNULL( ( SUM( CAST(T.stockQuantity AS SIGNED) ) + SUM( CAST( T.reservedStockQuantity AS SIGNED ) ) ), 0 ) AS stock_quantity
	 FROM
	Users U LEFT JOIN Transactions T ON U.id = T.userId LEFT JOIN Stocks S ON T.stockId = S.id WHERE S.id = %d
	 GROUP BY U.id;`, stockId)

	if err := tx.Raw(query).Scan(&results).Error; err != nil {
		l.Errorf("failed fetching SpecificStockEntry %+e", err)
		tx.Rollback()
		return nil, err

	}

	l.Infof("successfully fetched specificStockUserEntry from db")

	return results, nil
}
