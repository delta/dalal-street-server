package models

import (
	"database/sql/driver"
	"fmt"
	"sync"

	"github.com/Sirupsen/logrus"

	models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
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
)

var orderTypes = [...]string{
	"Limit",
	"Market",
	"StopLoss",
}

func (ot OrderType) String() string {
	return orderTypes[ot]
}

func OrderTypeFromProto(pOt models_proto.OrderType) OrderType {
	if pOt == models_proto.OrderType_LIMIT {
		return Limit
	} else if pOt == models_proto.OrderType_MARKET {
		return Market
	} else {
		return StopLoss
	}
}

type Ask struct {
	Id                     uint32    `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserId                 uint32    `gorm:"column:userId;not null" json:"user_id"`
	StockId                uint32    `gorm:"column:stockId;not null" json:"stock_id"`
	OrderType              OrderType `gorm:"column:orderType;not null" json:"order_type"`
	Price                  uint32    `gorm:"not null" json:"price"`
	StockQuantity          uint32    `gorm:"column:stockQuantity;not null" json:"stock_quantity"`
	StockQuantityFulfilled uint32    `gorm:"column:stockQuantityFulFilled;not null" json:"stock_quantity_fulfilled"`
	IsClosed               bool      `gorm:"column:isClosed;not null" json:"is_closed"`
	CreatedAt              string    `gorm:"column:createdAt;not null" json:"created_at"`
	UpdatedAt              string    `gorm:"column:updatedAt;not null" json:"updated_at"`
}

func (Ask) TableName() string {
	return "Asks"
}

func (gAsk *Ask) ToProto() *models_proto.Ask {
	m := make(map[OrderType]models_proto.OrderType)
	m[Limit] = models_proto.OrderType_LIMIT
	m[Market] = models_proto.OrderType_MARKET
	m[StopLoss] = models_proto.OrderType_STOPLOSS

	pAsk := &models_proto.Ask{
		Id:                     gAsk.Id,
		UserId:                 gAsk.UserId,
		StockId:                gAsk.StockId,
		Price:                  gAsk.Price,
		OrderType:              m[gAsk.OrderType],
		StockQuantity:          gAsk.StockQuantity,
		StockQuantityFulfilled: gAsk.StockQuantityFulfilled,
		IsClosed:               gAsk.IsClosed,
		CreatedAt:              gAsk.CreatedAt,
		UpdatedAt:              gAsk.UpdatedAt,
	}
	if gAsk.OrderType == Limit {
		pAsk.OrderType = models_proto.OrderType_LIMIT
	} else if gAsk.OrderType == Market {
		pAsk.OrderType = models_proto.OrderType_MARKET
	} else if gAsk.OrderType == StopLoss {
		pAsk.OrderType = models_proto.OrderType_STOPLOSS
	}

	return pAsk
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
	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return nil, err
	}
	defer db.Close()

	asksMap.m[id] = &Ask{}
	ask = asksMap.m[id]
	db.First(ask, id)

	if ask == nil {
		l.Errorf("Attempted to get non-existing Ask")
		return nil, fmt.Errorf("Ask with id %d does not exist", id)
	}

	l.Debugf("Loaded ask from db: %+v", ask)

	return ask, nil
}

/*
func getAskCopy(id uint32) (chan struct{}, *Ask, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "getAskCopy",
		"param_id": id,
	})

	var (
		a *askAndLock
		ch = make(chan struct{})
	)

	l.Debugf("Attempting")

	/* Try to see if the ask is there in the map * /
	askLocks.RLock()
	a, ok := askLocks.m[id]
	askLocks.Unlock()
	if ok {
		l.Debugf("Found ask in askLocks map. Locking.")
		a.Lock()
		go func() {
			l.Debugf("Waiting for caller to release lock")
			<-ch
			a.Unlock()
			l.Debugf("Lock released")
		}()
		return ch, a.ask, nil
	}

	/* Otherwise load from database * /
	l.Debugf("Loading ask from database")
	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return nil, nil, err
	}
	defer db.Close()

	askLocks.Lock()
	db.First(a.ask, id)
	askLocks.Unlock()

	if a.ask == nil {
		l.Errorf("Attempted to get non-existing Ask")
		return nil, nil, fmt.Errorf("Ask with id %d does not exist", id)
	}

	l.Debugf("Loaded ask from db. Locking")
	a.RLock()
	go func() {
		l.Debugf("Waiting for caller to release lock")
		<-ch
		askLocks.m[id].Unlock()
		l.Debugf("Lock released")
	}()

	l.Debugf("Ask: %+v", a.ask)

	return ch, a.ask, nil
}
*/
func createAsk(ask *Ask) error {
	var l = logger.WithFields(logrus.Fields{
		"method":    "CreateAsk",
		"param_ask": fmt.Sprintf("%+v", ask),
	})

	l.Debugf("Attempting")

	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return err
	}
	defer db.Close()

	if err := db.Create(ask).Error; err != nil {
		return err
	}

	l.Debugf("Created ask. Id: %d", ask.Id)

	return nil
}

func (ask *Ask) Close() error {
	var l = logger.WithFields(logrus.Fields{
		"method":    "Ask.Close",
		"param_ask": fmt.Sprintf("%+v", ask),
	})

	l.Debugf("Attempting")

	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return err
	}
	defer db.Close()
	ask.IsClosed = true

	if err := db.Save(ask).Error; err != nil {
		l.Error(err)
		return err
	}
	l.Debugf("Done")
	return nil
}

func GetMyAsks(userId, lastId, count uint32) (bool, []*Ask, []*Ask, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetMyAsks",
		"userId": userId,
		"lastId": lastId,
		"count":  count,
	})

	l.Infof("Attempting to get ask orders for userId : %v", userId)

	db, err := DbOpen()
	if err != nil {
		return true, nil, nil, err
	}
	defer db.Close()

	var myClosedAsks []*Ask
	var myOpenAsks []*Ask

	//get all open asks
	if err := db.Where("userId = ? and isClosed = ?", userId, 0).Find(&myOpenAsks).Error; err != nil {
		return true, nil, nil, err
	}

	//set default value of count if it is zero
	if count == 0 {
		count = MY_ASK_COUNT
	} else {
		count = min(count, MY_ASK_COUNT)
	}

	//get latest events if lastId is zero
	if lastId != 0 {
		db = db.Where("id <= ?", lastId)
	}
	//get closed asks
	if err := db.Where("userId = ? and isClosed = ?", userId, 1).Order("id desc").Limit(count).Find(&myClosedAsks).Error; err != nil {
		return true, nil, nil, err
	}

	var moreExists = len(myClosedAsks) >= int(count)
	l.Infof("Successfully fetched ask orders for userId : %v", userId)
	return moreExists, myOpenAsks, myClosedAsks, nil
}
