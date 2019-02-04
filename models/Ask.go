package models

import (
	"database/sql/driver"
	"fmt"
	"sync"

	"github.com/Sirupsen/logrus"
	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/jinzhu/gorm"

	"github.com/delta/dalal-street-server/utils"
)

type OrderType uint8

func (ot *OrderType) Scan(value interface{}) error {
	switch string(value.([]byte)) {
	case "Limit":
		*ot = Limit
	case "Market":
		*ot = Market
	case "StopLoss":
		*ot = StopLoss
	case "StopLossActive":
		*ot = StopLossActive
	default:
		return fmt.Errorf("Invalid value for OrderType. Got %s", string(value.([]byte)))
	}
	return nil
}

func (ot OrderType) Value() (driver.Value, error) { return ot.String(), nil }

const (
	Limit OrderType = iota
	Market
	StopLoss
	StopLossActive
)

var orderTypes = [...]string{
	"Limit",
	"Market",
	"StopLoss",
	"StopLossActive",
}

func (ot OrderType) String() string {
	return orderTypes[ot]
}

func OrderTypeFromProto(pOt models_pb.OrderType) OrderType {
	if pOt == models_pb.OrderType_LIMIT {
		return Limit
	} else if pOt == models_pb.OrderType_MARKET {
		return Market
	} else if pOt == models_pb.OrderType_STOPLOSS {
		return StopLoss
	} else {
		return StopLossActive
	}
}

type Ask struct {
	sync.Mutex
	Id                     uint32    `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserId                 uint32    `gorm:"column:userId;not null" json:"user_id"`
	StockId                uint32    `gorm:"column:stockId;not null" json:"stock_id"`
	OrderType              OrderType `gorm:"column:orderType;not null" json:"order_type"`
	Price                  uint64    `gorm:"not null" json:"price"`
	StockQuantity          uint64    `gorm:"column:stockQuantity;not null" json:"stock_quantity"`
	StockQuantityFulfilled uint64    `gorm:"column:stockQuantityFulFilled;not null" json:"stock_quantity_fulfilled"`
	IsClosed               bool      `gorm:"column:isClosed;not null" json:"is_closed"`
	CreatedAt              string    `gorm:"column:createdAt;not null" json:"created_at"`
	UpdatedAt              string    `gorm:"column:updatedAt;not null" json:"updated_at"`
}

func (Ask) TableName() string {
	return "Asks"
}

func (ask *Ask) ToProto() *models_pb.Ask {
	m := make(map[OrderType]models_pb.OrderType)
	m[Limit] = models_pb.OrderType_LIMIT
	m[Market] = models_pb.OrderType_MARKET
	m[StopLoss] = models_pb.OrderType_STOPLOSS

	pAsk := &models_pb.Ask{
		Id:                     ask.Id,
		UserId:                 ask.UserId,
		StockId:                ask.StockId,
		Price:                  ask.Price,
		OrderType:              m[ask.OrderType],
		StockQuantity:          ask.StockQuantity,
		StockQuantityFulfilled: ask.StockQuantityFulfilled,
		IsClosed:               ask.IsClosed,
		CreatedAt:              ask.CreatedAt,
		UpdatedAt:              ask.UpdatedAt,
	}

	return pAsk
}

// TriggerStoploss will set OrderType to StopLossActive if the ordertype is StopLoss
// Error is returned if that wasn't done successfully.
func (ask *Ask) TriggerStoploss() error {
	var l = logger.WithFields(logrus.Fields{
		"method":      "TriggerStoploss",
		"param_askId": ask.Id,
	})

	l.Debugf("Attempting")

	db := getDB()
	if ask.OrderType == StopLoss {
		ask.Lock()
		ask.OrderType = StopLossActive
		ask.UpdatedAt = utils.GetCurrentTimeISO8601()
		ask.Unlock()
		if err := db.Save(ask).Error; err != nil {
			l.Errorf("Error while saving data: %+v", err)
			return err
		}
		l.Debugf("Done")
		return nil
	}

	l.Errorf("Called TriggerStoploss on order of type %s", ask.OrderType.String())
	return nil // don't return any error here. Log is sufficient.
}

var asksMap = struct {
	sync.RWMutex
	m map[uint32]*Ask
}{
	sync.RWMutex{},
	make(map[uint32]*Ask),
}

func getAsk(id uint32) (*Ask, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":   "getAsk",
		"param_id": id,
	})

	l.Debugf("Attempting")

	/* Try to see if the ask is there in the map */
	asksMap.Lock()
	defer asksMap.Unlock()
	ask, ok := asksMap.m[id]
	if ok {
		l.Debugf("Found ask in asksMap")
		return ask, nil
	}

	/* Otherwise load from database */
	l.Debugf("Loading ask from database")

	db := getDB()

	ask = &Ask{}
	db.First(ask, id)

	if ask == nil {
		l.Errorf("Attempted to get non-existing Ask")
		return nil, fmt.Errorf("Ask with id %d does not exist", id)
	}

	asksMap.m[id] = ask

	l.Debugf("Loaded ask from db: %+v", ask)

	return ask, nil
}

// createAsk adds the ask to the database, fills the Id field of the ask
// and adds it to the asksMap
func createAsk(ask *Ask, tx *gorm.DB) error {
	var l = logger.WithFields(logrus.Fields{
		"method":    "CreateAsk",
		"param_ask": fmt.Sprintf("%+v", ask),
	})

	l.Debugf("Attempting")

	ask.CreatedAt = utils.GetCurrentTimeISO8601()
	ask.UpdatedAt = ask.CreatedAt

	if err := tx.Create(ask).Error; err != nil {
		return err
	}

	// add it to the asksMap
	asksMap.Lock()
	asksMap.m[ask.Id] = ask
	asksMap.Unlock()

	l.Debugf("Created ask. Id: %d", ask.Id)

	return nil
}

// AlreadyClosedError is given out when user tries to Cancel an already closed order.
// Unlikely to happen, but still possible
type AlreadyClosedError struct{ orderID uint32 }

func (e AlreadyClosedError) Error() string {
	return fmt.Sprintf("Order#%d is already closed. Cannot cancel now.", e.orderID)
}

// Marks an ask as closed and removes it from asksMap
func (ask *Ask) Close(tx *gorm.DB) error {
	var l = logger.WithFields(logrus.Fields{
		"method":    "Ask.Close",
		"param_ask": fmt.Sprintf("%+v", ask),
	})

	l.Debugf("Attempting")

	ask.Lock()
	if ask.IsClosed {
		ask.Unlock()
		return AlreadyClosedError{ask.Id}
	}
	ask.IsClosed = true
	ask.UpdatedAt = utils.GetCurrentTimeISO8601()
	ask.Unlock()

	if err := tx.Save(ask).Error; err != nil {
		l.Error(err)
		return err
	}

	// pointless to have a pointer to this now.
	asksMap.Lock()
	delete(asksMap.m, ask.Id)
	asksMap.Unlock()

	l.Debugf("Done")
	return nil
}

// GetAllOpenAsks returns all open asks. This will be called by MatchingEngine while initializing.
func GetAllOpenAsks() ([]*Ask, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetAllOpenAsks",
	})

	l.Infof("Attempting to get all open ask orders")

	db := getDB()

	var openAsks []*Ask

	//Load open ask orders from database
	if err := db.Where("isClosed = ?", 0).Find(&openAsks).Error; err != nil {
		panic("Error loading open ask orders in matching engine: " + err.Error())
	}

	asksMap.Lock()
	defer asksMap.Unlock()

	for _, ask := range openAsks {
		asksMap.m[ask.Id] = ask
	}

	l.Infof("Done")

	return openAsks, nil
}

func GetMyOpenAsks(userId uint32) ([]*Ask, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetMyOpenAsks",
		"userId": userId,
	})

	l.Infof("Attempting to get open ask orders for userId : %v", userId)

	db := getDB()

	var myOpenAsks []*Ask

	if err := db.Where("userId = ? and isClosed = ?", userId, 0).Find(&myOpenAsks).Error; err != nil {
		return nil, err
	}

	l.Infof("Successfully fetched open ask orders for userId : %v", userId)
	return myOpenAsks, nil
}

func GetMyClosedAsks(userId, lastId, count uint32) (bool, []*Ask, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetMyClosedAsks",
		"userId": userId,
		"lastId": lastId,
		"count":  count,
	})

	l.Infof("Attempting to get closed ask orders for userId : %v", userId)

	db := getDB()

	var myClosedAsks []*Ask

	//set default value of count if it is zero
	if count == 0 {
		count = MY_ASK_COUNT
	} else {
		count = utils.MinInt32(count, MY_ASK_COUNT)
	}

	//get latest events if lastId is zero
	if lastId != 0 {
		db = db.Where("id <= ?", lastId)
	}
	//get closed asks
	if err := db.Where("userId = ? and isClosed = ?", userId, 1).Order("id desc").Limit(count).Find(&myClosedAsks).Error; err != nil {
		return true, nil, err
	}

	var moreExists = len(myClosedAsks) >= int(count)
	l.Infof("Successfully fetched closed ask orders for userId : %v", userId)
	return moreExists, myClosedAsks, nil
}
