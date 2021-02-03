package models

import (
	"errors"
	"fmt"

	"github.com/jinzhu/gorm"

	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/sirupsen/logrus"
)

var (
	InvalidRequestError       = errors.New("Invalid Request")
	InvalidUserError          = errors.New("Invalid User")
	InvalidCerdentialError    = errors.New("invalid credentials")
	InvalidChallengeTypeError = errors.New("challenge type not supported")
)

// DailyChallenge model
type DailyChallenge struct {
	Id            uint32 `gorm:"column:id;primary_key;AUTO_INCREMENT" json:"id"`
	MarketDay     uint32 `gorm:"column:marketDay;not null" json:"market_day"`
	ChallengeType string `gorm:"column:challengeType;not null" json:"challenge_type"`
	Value         uint64 `gorm:"column:value;not null" json:"value"`
	StockId       uint32 `gorm:"column:stockId;default null" json:"stock_id"`
	Reward        uint32 `gorm:"column:reward; not null" json:"reward"`
}

//UserState model
type UserState struct {
	Id              uint32 `gorm:"column:id;primary_key;AUTO_INCREMENT" json:"id"`
	ChallengeId     uint32 `gorm:"column:challengeId;not null" json:"challenge_id"`
	UserId          uint32 `gorm:"column:userId;not null" json:"user_id"`
	MarketDay       uint32 `gorm:"column:marketDay;not null" json:"market_day"`
	InitialValue    int64  `gorm:"column:initialValue;not null" json:"initial_value"`
	FinalValue      int64  `gorm:"column:finalValue;default null" json:"final_value"`
	IsCompleted     bool   `gorm:"column:isCompleted;default false" json:"is_completed"`
	IsRewardClamied bool   `gorm:"column:isRewardClaimed;default false" json:"is_reward_claimed"`
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

type updateUserStateQueryData struct {
	Id            uint32
	UserId        uint32
	ChallengeId   uint32
	InitialValue  int64
	ChallengeType string
	Value         uint64
	StockId       uint32
}

type getMyRewardQueryData struct {
	Id              uint32
	UserId          uint32
	FinalValue      uint32
	IsCompleted     bool
	IsRewardClaimed bool
	Marketday       uint32
	Reward          uint64
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
		Reward:        d.Reward,
	}
	return pDailyChallenge
}

func (d *UserState) ToProto() *models_pb.UserState {
	pUserState := &models_pb.UserState{
		Id:              d.Id,
		ChallengeId:     d.ChallengeId,
		UserId:          d.UserId,
		InitialValue:    d.InitialValue,
		FinalValue:      d.FinalValue,
		IsCompleted:     d.IsCompleted,
		IsRewardClamied: d.IsRewardClamied,
	}

	return pUserState
}

//GetDailyChallenges returns challenges as array for a given market day
func GetDailyChallenges(marketDay uint32) ([]*DailyChallenge, error) {

	l := logger.WithFields(logrus.Fields{
		"method":    "GetDailyChallenges",
		"marketDay": marketDay,
	})

	l.Infof("GetDailyChallenges Requested for market day:%d", marketDay)

	var dailyChallenges []*DailyChallenge

	l.Debugf("Attempting to get dailyChallenges")

	db := getDB()

	if err := db.Table("DailyChallenge").Where("marketDay = ?", marketDay).Find(&dailyChallenges).Error; err != nil {
		l.Errorf("error while querying from db")
		return nil, err
	}

	l.Infof("Successfully fetched dailyChallenges for marketday:%d", marketDay)
	return dailyChallenges, nil
}

//AddDailyChallenge add daily challenge to db, only Admin can invoke this function
func AddDailyChallenge(value uint64, marketDay uint32, stockId uint32, challengeType string, reward uint32) error {
	l := logger.WithFields(logrus.Fields{
		"method":              "AddDailyChallenge",
		"param_day":           marketDay,
		"param_value":         value,
		"param_stockid":       stockId,
		"param_challengeType": challengeType,
	})

	liveMarketDay := GetMarketDay()

	if liveMarketDay == 0 || liveMarketDay > marketDay {
		return InvalidRequestError
	}

	if challengeType == "SpecificStock" {
		if stockId == 0 {
			return InvalidRequestError
		}
	} else if stockId != 0 {
		return InvalidRequestError
	}

	db := getDB()

	l.Infof("Attempting to save daily challenge")

	dailyChallenge := &DailyChallenge{
		MarketDay:     marketDay,
		ChallengeType: challengeType,
		Value:         value,
		StockId:       stockId,
		Reward:        reward,
	}

	if stockId == 0 {
		if err := db.Table("DailyChallenge").Omit("StockId").Save(dailyChallenge).Error; err != nil {
			l.Error(err)
			return InternalServerError
		}
	} else {
		if err := db.Table("DailyChallenge").Save(dailyChallenge).Error; err != nil {
			l.Error(err)
			return InternalServerError
		}
	}

	l.Infof("successfully added daily challenge")

	return nil
}

//OpenDailyChallenge opens dailyChallenge
//saves initial user state depending upon the challengetype in db for later computation while Closing dailyChallenge
func OpenDailyChallenge(marketDay uint32) error {
	l := logger.WithFields(logrus.Fields{
		"method":     "OpenDailyChallenge",
		"market_day": marketDay,
	})

	l.Infof("OpenChallenge Requested!")

	challengeStatus := IsDailyChallengeOpen()

	if challengeStatus == true {
		return InvalidRequestError
	}

	dailyChallenges, err := GetDailyChallenges(marketDay)

	if err != nil {
		return InternalServerError
	}

	if err := saveUsersState(dailyChallenges, marketDay); err != nil {
		l.Errorf("failed to save userstate %+e", err)
		return InternalServerError
	}

	l.Infof("succesfully saved userstate for DailyChallenges")

	if err := SetIsDailyChallengeOpen(true); err != nil {
		l.Errorf("failed to update dailyChallenge status %+e", err)
		return InternalServerError
	}

	gameStateStream := datastreamsManager.GetGameStateStream()
	g := &GameState{
		UserID: 0,
		Dc: &DailyChallengeStatus{
			IsDailyChallengeOpen: true,
		},
		GsType: DailyChallengeStatusUpdate,
	}
	gameStateStream.SendGameStateUpdate(g.ToProto())

	return nil
}

//CloseDailyChallenge closes dailychallenge and updates UserState
func CloseDailyChallenge() error {
	marketDay := GetMarketDay()

	l := logger.WithFields(logrus.Fields{
		"method":     "CloseDailyChallenge",
		"market_day": marketDay,
	})

	l.Infof("CloseDailyChallenge Requested")

	challengeStatus := IsDailyChallengeOpen()

	if challengeStatus == false {
		return InvalidRequestError
	}

	if err := updateUserState(marketDay); err != nil {
		l.Errorf("failed to update userState %+e", err)
		return InternalServerError
	}

	l.Infof("Successfully updated Userstate")

	//setting isDailyChallengeOpen to false
	if err := SetIsDailyChallengeOpen(false); err != nil {
		l.Errorf("failed to update dailyChallenge status %+e", err)
		return InternalServerError
	}
	// streaming dailyChallenge status
	gameStateStream := datastreamsManager.GetGameStateStream()
	g := &GameState{
		UserID: 0,
		Dc: &DailyChallengeStatus{
			IsDailyChallengeOpen: false,
		},
		GsType: DailyChallengeStatusUpdate,
	}
	gameStateStream.SendGameStateUpdate(g.ToProto())

	return nil
}

//updateUsersState updates userState when dailyChallenges Closes
//similiar to saveUsersState,invoked inside CloseDailyChallenge
func updateUserState(marketday uint32) error {
	l := logger.WithFields(logrus.Fields{
		"method":     "updateUserState",
		"market_day": marketday,
	})

	l.Debugf("Attempting to update userState")

	db := getDB()

	//begin transaction
	tx := db.Begin()

	if err := tx.Error; err != nil {
		l.Error(err)
		return err
	}

	var queryResults []updateUserStateQueryData

	query := fmt.Sprintf(`SELECT U.id AS id,U.userId AS user_id, U.challengeId AS challenge_id,U.initialValue AS initial_value,D.challengeType AS challenge_type,D.value AS value,D.stockId AS stock_id
	 FROM
	UserState U LEFT JOIN DailyChallenge D ON U.challengeId = D.id WHERE U.marketDay = %d;`, marketday)

	if err := tx.Raw(query).Scan(&queryResults).Error; err != nil {
		l.Errorf("error, fetching userSate query data %+e", err)
		return err
	}

	for _, q := range queryResults {

		switch q.ChallengeType {
		case "Cash":
			userStateEntry := &UserState{
				Id: q.Id,
			}

			ch, user, err := getUserExclusively(q.UserId)
			if err != nil {
				l.Errorf("Errored : %+v ", err)
				return err
			}
			l.Debugf("Acquired")

			if int64(user.Cash+user.ReservedCash) >= q.InitialValue+int64(q.Value) {
				userStateEntry.IsCompleted = true
			}

			userStateEntry.FinalValue = int64(user.Cash)

			if err := tx.Table("UserState").Select("FinalValue", "IsCompleted").Save(userStateEntry).Error; err != nil {
				l.Errorf("failed saving userState cash Challenge type %+e", err)
				tx.Rollback()
				close(ch)
				return err
			}

			close(ch)
			l.Debugf("Released exclusive write on user")

			l.Debugf("updated userstate challenge type cash")

		case "NetWorth":
			userStateEntry := &UserState{
				Id: q.Id,
			}

			ch, user, err := getUserExclusively(q.UserId)
			if err != nil {
				l.Errorf("Errored : %+v ", err)
				return err
			}
			l.Debugf("Acquired")

			if int64(user.Total) >= q.InitialValue+int64(q.Value) {
				userStateEntry.IsCompleted = true
			}

			userStateEntry.FinalValue = int64(user.Total)

			if err := tx.Table("UserState").Select("FinalValue", "IsCompleted").Save(userStateEntry).Error; err != nil {
				l.Errorf("failed saving userState net worth Challenge type %+e", err)
				tx.Rollback()
				close(ch)
				return err
			}

			close(ch)
			l.Debugf("Released exclusive write on user")

			l.Debugf("updated userstate challenge type networth")

		case "SpecificStock":
			userStateEntry := &UserState{
				Id: q.Id,
			}

			ch, user, err := getUserExclusively(q.UserId)
			if err != nil {
				l.Errorf("Errored : %+v ", err)
				return err
			}
			l.Debugf("Acquired")

			stockQuantity, err := getSingleStockCount(user, q.StockId)

			if err != nil {
				l.Error(err)
				return err
			}

			if stockQuantity >= q.InitialValue+int64(q.Value) {
				userStateEntry.IsCompleted = true
			}

			userStateEntry.FinalValue = int64(stockQuantity)

			if err := tx.Table("UserState").Select("FinalValue", "IsCompleted").Save(userStateEntry).Error; err != nil {
				l.Errorf("failed updating userState specific stock Challenge type %+e", err)
				tx.Rollback()
				close(ch)
				return err
			}

			close(ch)
			l.Debugf("Released exclusive write on user")

			l.Debugf("updated userstate challenge type SpecificStock")

		case "StockWorth":
			userStateEntry := &UserState{
				Id: q.Id,
			}
			stockWorth, err := GetUserStockWorth(q.UserId)

			if err != nil {
				l.Error(err)
				return err
			}
			fmt.Println(stockWorth, q.Value, q.InitialValue, q.InitialValue+int64(q.Value))

			if stockWorth >= q.InitialValue+int64(q.Value) {
				userStateEntry.IsCompleted = true
			}

			userStateEntry.FinalValue = int64(stockWorth)

			if err := tx.Table("UserState").Select("FinalValue", "IsCompleted").Save(userStateEntry).Error; err != nil {
				l.Errorf("failed updating userState stockworth Challenge type %+e", err)
				tx.Rollback()
				return err
			}

			l.Debugf("updated userstate challenge type stockworth")

		default:
			l.Error("something went wrong, updating userState failed,Rolling back...")
			tx.Rollback()
			return InvalidChallengeTypeError

		}

	}
	//commit transaction
	if err := tx.Commit().Error; err != nil {
		l.Error(err)
		return err
	}

	return nil

}

//saveUsersState saves registered users cash,stockworth,Networth,specificstock quantity based on challenge type
//invoked inside OpenDailyChallenge
func saveUsersState(c []*DailyChallenge, marketday uint32) error {
	l := logger.WithFields(logrus.Fields{
		"method": "saveUsersState",
	})

	l.Debugf("Attempting to save user state")

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
	Users U LEFT JOIN Transactions T ON U.id = T.userId LEFT JOIN Stocks S ON T.stockId = S.id WHERE U.blockCount < %d GROUP BY U.id;`, config.MaxBlockCount)

	if err := tx.Raw(query).Scan(&queryResults).Error; err != nil {
		l.Errorf("error, fetching userSate query data %+e", err)
		return err
	}

	for _, challenge := range c {

		switch challenge.ChallengeType {

		case "Cash":
			var userStateEntry *UserState

			for _, u := range queryResults {
				userStateEntry = &UserState{
					ChallengeId:     challenge.Id,
					UserId:          u.UserId,
					MarketDay:       challenge.MarketDay,
					InitialValue:    int64(u.Cash),
					IsCompleted:     false,
					IsRewardClamied: false,
				}

				if err := tx.Table("UserState").Omit("FinalValue").Save(userStateEntry).Error; err != nil {
					l.Errorf("failed saving userState cash Challenge type %+e", err)
					tx.Rollback()
					return err
				}
			}

		case "NetWorth":
			var userStateEntry *UserState

			for _, u := range queryResults {
				userStateEntry = &UserState{
					ChallengeId:     challenge.Id,
					UserId:          u.UserId,
					MarketDay:       challenge.MarketDay,
					InitialValue:    u.Total,
					IsCompleted:     false,
					IsRewardClamied: false,
				}

				if err := tx.Table("UserState").Omit("FinalValue").Save(userStateEntry).Error; err != nil {
					l.Errorf("failed saving userState NetWorth Challenge type %+e", err)
					tx.Rollback()
					return err
				}
			}

		case "StockWorth":
			var userStateEntry *UserState

			for _, u := range queryResults {
				userStateEntry = &UserState{
					ChallengeId:     challenge.Id,
					UserId:          u.UserId,
					MarketDay:       challenge.MarketDay,
					InitialValue:    u.StockWorth,
					IsCompleted:     false,
					IsRewardClamied: false,
				}

				if err := tx.Table("UserState").Omit("FinalValue").Save(userStateEntry).Error; err != nil {
					l.Errorf("failed saving userState StockWorth Challenge type %+e", err)
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
					ChallengeId:     challenge.Id,
					UserId:          u.UserId,
					MarketDay:       challenge.MarketDay,
					InitialValue:    u.StockQuantity,
					IsCompleted:     false,
					IsRewardClamied: false,
				}

				if err := tx.Table("UserState").Omit("FinalValue").Save(userStateEntry).Error; err != nil {
					l.Errorf("failed saving userState SpecificStockType Challenge type %+e", err)
					tx.Rollback()
					return err
				}
			}

		default:
			l.Error("challenge type not supported, userstate not saved")
			tx.Rollback()
			return InvalidChallengeTypeError
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

	l.Debugf("getSpecificStockEntry requested")

	var results []specificStockUserEntry

	query := fmt.Sprintf(`SELECT
    U.id AS user_id,
    IFNULL(
        (
            SUM(T.stockQuantity) + SUM(T.reservedStockQuantity)
        ),
        0
    ) AS stock_quantity FROM  Users U LEFT JOIN Transactions T ON U.id = T.userId LEFT JOIN Stocks S ON T.stockId = %d WHERE U.blockCount < %d GROUP BY U.id;`, stockId, config.MaxBlockCount)

	if err := tx.Raw(query).Scan(&results).Error; err != nil {
		l.Errorf("failed fetching SpecificStockEntry %+e", err)
		tx.Rollback()
		return results, err

	}

	l.Debugf("successfully fetched specificStockUserEntry from db")

	return results, nil
}

//GetUserState returns userState
func GetUserState(marketDay, userId, challengeId uint32) (*UserState, error) {

	l := logger.WithFields(logrus.Fields{
		"method":       "GetUserState",
		"market_day":   marketDay,
		"user_id":      userId,
		"challenge_id": challengeId,
	})

	l.Debugf("GetUserState Requested!")

	var userState = &UserState{}

	db := getDB()

	if err := db.Table("UserState").Where(" userId = ? ", userId).Where("marketDay = ?", marketDay).Where("challengeId = ?", challengeId).First(&userState).Error; err != nil {
		l.Errorf("error loading userState %+e", err)
		return userState, err
	}
	l.Debugf("successfully fetched UserState")

	return userState, nil

}

//GetMyReward add reward to user as  cash
func GetMyReward(userStateId, userId uint32) (uint64, error) {

	l := logger.WithFields(logrus.Fields{
		"method":        "GetMyReward",
		"user_id":       userId,
		"user_state_id": userStateId,
	})

	l.Debugf("GetMyReward Requested")

	db := getDB()

	//begin transaction
	tx := db.Begin()

	if err := tx.Error; err != nil {
		l.Error(err)
		return 0, InternalServerError
	}

	userRewardQuery := getMyRewardQueryData{}

	query := "SELECT U.id AS id,U.userid AS user_id, U.finalValue AS final_value, U.isCompleted AS is_completed,U.isRewardClaimed AS is_reward_claimed,U.marketday AS market_day,D.reward AS reward FROM UserState U LEFT JOIN DailyChallenge D ON U.challengeId  = D.id WHERE U.id = ?"

	if err := tx.Raw(query, userStateId).Scan(&userRewardQuery).Error; err != nil {
		l.Errorf("failed fetching userRewardQuery %+e", err)
		tx.Rollback()
		return 0, InternalServerError

	}

	if userRewardQuery.UserId != userId {
		return 0, InvalidUserError
	}

	if userRewardQuery.Marketday == GetMarketDay() && IsDailyChallengeOpen() {
		return 0, InvalidRequestError
	}

	if !userRewardQuery.IsCompleted {
		return 0, InvalidCerdentialError
	}

	if userRewardQuery.IsRewardClaimed {
		return 0, InvalidRequestError
	}

	ch, user, err := getUserExclusively(userId)

	if err != nil {
		close(ch)
		return 0, InternalServerError
	}
	l.Debugf("Acquired")
	defer func() {
		close(ch)
		l.Debugf("Released exclusive write on user")
	}()

	user.Cash += userRewardQuery.Reward

	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		l.Errorf("Error updating user cash. %+e", err)
		return 0, InternalServerError
	}

	if err := tx.Table("UserState").Where("id = ?", userStateId).Update("isRewardClaimed", true).Error; err != nil {
		l.Errorf("Error updating isRewardClaimed %v", err)
		tx.Rollback()
		return 0, InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		l.Error(err)
		return 0, InternalServerError
	}
	//streams user reward credit
	gameStateStream := datastreamsManager.GetGameStateStream()
	g := &GameState{
		UserID: userId,
		Ur: &UserRewardCredit{
			Cash: user.Cash,
		},
		GsType: UserRewardCreditUpdate,
	}
	gameStateStream.SendGameStateUpdate(g.ToProto())

	l.Debugf("Successfully rewarded cash to the user")

	return userRewardQuery.Reward, nil

}

//saveNewUserState saves new user state i.e users who registers when dailyChallenge is open
func saveNewUserState(userId uint32) error {
	l := logger.WithFields(logrus.Fields{
		"method":  "GetMyReward",
		"user_id": userId,
	})

	l.Debugf("saveNewUserState Requested")

	marketDay := GetMarketDay()

	var totalChallenges []*DailyChallenge

	var i uint32

	for i = 1; i <= marketDay; i++ {
		challenges, err := GetDailyChallenges(i)

		if err != nil {
			l.Errorf("failed fetching daily challenges for day %d", i)
			return err
		}
		totalChallenges = append(totalChallenges, challenges...)
	}

	db := getDB()

	//begin transaction
	tx := db.Begin()

	if err := tx.Error; err != nil {
		l.Error(err)
		return err
	}

	for _, c := range totalChallenges {

		if c.ChallengeType == "Cash" || c.ChallengeType == "NetWorth" {
			userStateEntry := &UserState{
				ChallengeId:  c.Id,
				UserId:       userId,
				MarketDay:    c.MarketDay,
				InitialValue: STARTING_CASH,
			}

			if c.MarketDay < marketDay {
				userStateEntry.FinalValue = STARTING_CASH
				if err := tx.Table("UserState").Omit("Iscompleted", "IsRewardClaimed").Save(userStateEntry).Error; err != nil {
					l.Errorf("failed saving userState %+e", err)
					tx.Rollback()
					return err
				}
			} else {
				if err := tx.Table("UserState").Omit("FinalValue", "Iscompleted", "IsRewardClaimed").Save(userStateEntry).Error; err != nil {
					l.Errorf("failed saving userState %+e", err)
					tx.Rollback()
					return err
				}
			}

		} else if c.ChallengeType == "StockWorth" || c.ChallengeType == "SpecificStock" {

			userStateEntry := &UserState{
				ChallengeId:  c.Id,
				UserId:       userId,
				MarketDay:    c.MarketDay,
				InitialValue: 0,
			}
			if c.MarketDay < marketDay {
				userStateEntry.FinalValue = 0
				if err := tx.Table("UserState").Omit("Iscompleted", "IsRewardClaimed").Save(userStateEntry).Error; err != nil {
					l.Errorf("failed saving userState %+e", err)
					tx.Rollback()
					return err
				}
			} else {
				if err := tx.Table("UserState").Omit("FinalValue", "Iscompleted", "IsRewardClaimed").Save(userStateEntry).Error; err != nil {
					l.Errorf("failed saving userState %+e", err)
					tx.Rollback()
					return err
				}
			}
		} else {
			l.Error("challenge type not supported, userstate not saved")
			tx.Rollback()
			return InvalidChallengeTypeError
		}

	}

	//commit transaction
	if err := tx.Commit().Error; err != nil {
		l.Error(err)
		return err
	}

	l.Debugf("Done")

	return nil
}
