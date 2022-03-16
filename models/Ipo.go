package models

import (
	"fmt"

	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/delta/dalal-street-server/utils"
	"github.com/sirupsen/logrus"
)

type IpoStock struct {
	Id            uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	ShortName     string `gorm:"column:shortName;not null" json:"short_name"`
	FullName      string `gorm:"column:fullName;not null" json:"full_name"`
	Description   string `gorm:"column:description;not null" json:"description"`
	CreatedAt     string `gorm:"column:createdAt;not null" json:"created_at"`
	UpdatedAt     string `gorm:"column:updatedAt;not null" json:"updated_at"`
	IsBiddable    bool   `gorm:"column:isBiddable;not null" json:"is_biddable"`
	SlotPrice     uint64 `gorm:"column:slotPrice;not null"  json:"slot_price"`
	StockPrice    uint64 `gorm:"column:stockPrice;not null"  json:"stock_price"`
	SlotQuantity  uint32 `gorm:"column:slotQuantity;not null"  json:"slot_quantity"`
	StocksPerSlot uint32 `gorm:"column:stocksPerSlot;not null"  json:"stocks_per_slot"`
}

func (IpoStock) TableName() string {
	return "IpoStocks"
}

func (gIpoStock *IpoStock) ToProto() *models_pb.IpoStock {
	return &models_pb.IpoStock{
		Id:            gIpoStock.Id,
		ShortName:     gIpoStock.ShortName,
		FullName:      gIpoStock.FullName,
		Description:   gIpoStock.Description,
		SlotPrice:     gIpoStock.SlotPrice,
		StockPrice:    gIpoStock.StockPrice,
		SlotQuantity:  gIpoStock.SlotQuantity,
		StocksPerSlot: gIpoStock.StocksPerSlot,
		CreatedAt:     gIpoStock.CreatedAt,
		UpdatedAt:     gIpoStock.UpdatedAt,
		IsBiddable:    gIpoStock.IsBiddable,
	}
}

type IpoNotBiddableError struct{ IpoStockId uint32 }

func (e IpoNotBiddableError) Error() string {
	return fmt.Sprintf("IPO is not biddable for stock %d", e.IpoStockId)
}

type IpoAlreadyOpenError struct{ IpoStockId uint32 }

func (e IpoAlreadyOpenError) Error() string {
	return fmt.Sprintf("IPO bidding has already been opened for stock %d", e.IpoStockId)
}

func GetAllIpoStocks() (map[uint32]IpoStock, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetAllIpoStocks",
	})

	db := getDB()
	var allIpoStocks []IpoStock

	if err := db.Find(&allIpoStocks).Error; err != nil {
		l.Error(err)
		return make(map[uint32]IpoStock), err
	}
	var allIpoStocksMap = make(map[uint32]IpoStock)

	for _, ipoStock := range allIpoStocks {
		fmt.Println("ipoStock", ipoStock)
		allIpoStocksMap[ipoStock.Id] = ipoStock
	}

	l.Infof("all ipo stocks : %v", allIpoStocksMap)

	return allIpoStocksMap, nil
}

func AllowIpoBidding(IpoStockId uint32) error {
	var l = logger.WithFields(logrus.Fields{
		"method":           "OpenIpoBidding",
		"param_IpoStockId": IpoStockId,
	})

	l.Debugf("Attempting")

	db := getDB()

	IpoStock1 := &IpoStock{}
	if err := db.First(IpoStock1, IpoStockId).Error; err != nil {
		l.Error(err)
		return err
	}

	if IpoStock1 == nil {
		return InvalidStockIdError{}
	}

	if IpoStock1.IsBiddable {
		return IpoAlreadyOpenError{IpoStockId}
	}

	IpoStock1.IsBiddable = true
	IpoStock1.UpdatedAt = utils.GetCurrentTimeISO8601()

	if err := db.Save(IpoStock1).Error; err != nil {
		l.Error(err)
		return err
	}

	go func() {
		n := &Notification{
			UserId:      0,
			Text:        IpoStock1.FullName + " Initial public offering is listed in the market, you can start placing orders",
			IsBroadcast: true,
			CreatedAt:   utils.GetCurrentTimeISO8601(),
		}

		notificationsStream := datastreamsManager.GetNotificationsStream()
		notificationsStream.SendNotification(n.ToProto())
	}()

	return nil
}

func AllotSlots(IpoStockId uint32) error {
	var l = logger.WithFields(logrus.Fields{
		"method":           "AllotSlots",
		"param_IpoStockId": IpoStockId,
	})

	l.Debugf("Attempting")

	db := getDB()

	IpoStock := &IpoStock{}
	if err := db.First(IpoStock, IpoStockId).Error; err != nil {
		l.Error(err)
		return err
	}

	if IpoStock == nil {
		return InvalidStockIdError{}
	}

	if IpoStock.IsBiddable == false {
		return IpoNotBiddableError{IpoStockId}
	}

	IpoStock.IsBiddable = false
	IpoStock.UpdatedAt = utils.GetCurrentTimeISO8601()

	if err := db.Save(IpoStock).Error; err != nil {
		l.Error(err)
		return err
	}
	totalslots := IpoStock.SlotQuantity

	var openIpoBids []*IpoBid

	//Load open ipoBid orders from database
	if err := db.Where("IpoStockId = ? AND isClosed = ?", IpoStockId, 0).Find(&openIpoBids).Error; err != nil {
		l.Error(err)
		return err
	}

	l.Infof("Fetched %d open ipoBids for IpoStockId= %d. Attempting to allot these stocks", len(openIpoBids), IpoStockId)

	totalbids := len(openIpoBids)
	var subscriptionRatio = float64(totalbids) / float64(totalslots)
	var IpoStocksInMarket uint64
	var ListingPrice uint64

	if subscriptionRatio <= 1.05 && subscriptionRatio >= 0.95 {
		ListingPrice = IpoStock.StockPrice
	} else if subscriptionRatio < 1.20 && subscriptionRatio > 1.05 {
		ListingPrice = IpoStock.StockPrice * 105 / 100
	} else if subscriptionRatio < 0.95 && subscriptionRatio > 0.80 {
		ListingPrice = IpoStock.StockPrice * 95 / 100
	} else if subscriptionRatio >= 1.20 {
		ListingPrice = IpoStock.StockPrice * 110 / 100
	} else { // if subscriptionRatio <= 0.80
		ListingPrice = IpoStock.StockPrice * 90 / 100
	}

	newStock := &Stock{
		ShortName:        IpoStock.ShortName,
		FullName:         IpoStock.FullName,
		Description:      IpoStock.Description,
		CurrentPrice:     ListingPrice,
		DayHigh:          ListingPrice,
		DayLow:           ListingPrice,
		AllTimeHigh:      ListingPrice,
		AllTimeLow:       ListingPrice,
		StocksInExchange: 0,
		StocksInMarket:   IpoStocksInMarket, // this will be zero
		UpOrDown:         true,
		PreviousDayClose: ListingPrice,
		LastTradePrice:   ListingPrice,
		RealAvgPrice:     float64(ListingPrice),
		CreatedAt:        utils.GetCurrentTimeISO8601(),
		GivesDividends:   false,
		IsBankrupt:       false,
	}
	newStock.UpdatedAt = newStock.CreatedAt

	if err := db.Create(newStock).Error; err != nil {
		l.Error(err)
		return err
	}

	cost := int64(IpoStock.SlotPrice)

	if subscriptionRatio <= 1.00 {
		for _, ipoBid := range openIpoBids {

			if err := allotIpoSlotToUser(ipoBid, newStock.Id, ipoBid.SlotQuantity, IpoStock.StocksPerSlot, cost); err != nil {
				l.Error(err)
				return err
			}

		}

		IpoStocksInMarket = uint64(uint32(len(openIpoBids)) * IpoStock.StocksPerSlot)
	} else {

		var AllotedIpoBids []*IpoBid

		// select 'totalslots' number of bids randomly
		if err := db.Raw("SELECT * FROM IpoBids WHERE ipoStockId = ? AND isClosed = ? ORDER BY RAND() LIMIT ?", IpoStockId, 0, totalslots).Scan(&AllotedIpoBids).Error; err != nil {
			l.Error(err)
			return err
		}

		for _, AllotedIpoBid := range AllotedIpoBids {

			if err := allotIpoSlotToUser(AllotedIpoBid, newStock.Id, AllotedIpoBid.SlotQuantity, IpoStock.StocksPerSlot, cost); err != nil {
				l.Error(err)
				return err
			}
		}

		var UnfulIpoBids []*IpoBid

		//Load open unfulfilled ipoBid orders from database
		if err := db.Where("ipoStockId = ? AND isClosed = ? AND isFulfilled = ?", IpoStockId, 0, 0).Find(&UnfulIpoBids).Error; err != nil {
			l.Error(err)
			return err
		}

		for _, UnfulipoBid := range UnfulIpoBids {
			//  Refund slotprice amount to userid
			if err := RefundIpoSlotToUser(UnfulipoBid, newStock.Id, UnfulipoBid.SlotQuantity, IpoStock.StocksPerSlot, cost); err != nil {
				l.Error(err)
				return err
			}
		}

		IpoStocksInMarket = uint64(totalslots * IpoStock.StocksPerSlot)
	}

	// Update stock values

	newStock.StocksInMarket = IpoStocksInMarket
	newStock.UpdatedAt = utils.GetCurrentTimeISO8601()

	if err := db.Save(newStock).Error; err != nil {
		l.Error(err)
		return err
	}

	SendPushNotification(0, PushNotification{
		Title:    fmt.Sprintf("IPO Allotment has just happened for %v", IpoStock.FullName),
		Message:  fmt.Sprintf("IPO Allotment has just happened for  %v. Click here to know more.", IpoStock.FullName),
		LogoUrl:  "",
		ImageUrl: "",
	})

	LoadStocks() // reload stocks (usually called when market day opens)

	return nil
}

func allotIpoSlotToUser(ipoBid *IpoBid, newStockId, SlotQuantity, StocksPerSlot uint32, cost int64) error {
	l := logger.WithFields(logrus.Fields{
		"method":   "allotIpoSlotToUser",
		"IpoBidId": ipoBid.Id,
	})

	db := getDB()
	tx := db.Begin()

	// allot 1 slot worth of stocks to userid
	AllotIpoTransaction := GetTransactionRef(ipoBid.UserId, newStockId, IpoAllotmentTransaction, 0, int64(SlotQuantity*StocksPerSlot), 0, -cost, 0)

	ipoBid.IsFulfilled = true
	ipoBid.IsClosed = true
	ipoBid.UpdatedAt = utils.GetCurrentTimeISO8601()

	ch, AllotedUser, err := getUserExclusively(ipoBid.UserId)
	l.Debugf("Acquired")

	defer func() {
		close(ch)
		l.Debugf("Released exclusive write on user")
	}()

	if err != nil {
		l.Errorf("Error acquiring user. Failing. %+v", err)
		return InternalServerError
	}

	oldReservedCash := AllotedUser.ReservedCash
	AllotedUser.ReservedCash -= uint64(cost)

	l.Infof("Saving AllotIpoTransaction, IpoStockId : %d, SlotQuantity : %d, UserId : %d, Cost: %d", ipoBid.IpoStockId, ipoBid.SlotQuantity, ipoBid.UserId, cost)

	if err := tx.Save(ipoBid).Error; err != nil {
		AllotedUser.ReservedCash = oldReservedCash
		l.Error(err)
		tx.Rollback()
		return err
	}

	if err := tx.Save(&AllotedUser).Error; err != nil {
		AllotedUser.ReservedCash = oldReservedCash
		l.Error(err)
		tx.Rollback()
		return err
	}

	if err := tx.Create(AllotIpoTransaction).Error; err != nil {
		AllotedUser.ReservedCash = oldReservedCash
		l.Error(err)
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		AllotedUser.ReservedCash = oldReservedCash
		l.Errorf("Error committing transaction %+v", err)
		tx.Rollback()
		return err
	}

	transactionsStream := datastreamsManager.GetTransactionsStream()
	transactionsStream.SendTransaction(AllotIpoTransaction.ToProto())

	return nil
}

func RefundIpoSlotToUser(ipoBid *IpoBid, newStockId, SlotQuantity, StocksPerSlot uint32, cost int64) error {
	l := logger.WithFields(logrus.Fields{
		"method":   "RefundIpoSlotToUser",
		"IpoBidId": ipoBid.Id,
	})

	IpoRefundTransaction := GetTransactionRef(ipoBid.UserId, newStockId, IpoAllotmentTransaction, 0, 0, 0, cost, -cost)

	ipoBid.IsClosed = true
	ipoBid.UpdatedAt = utils.GetCurrentTimeISO8601()

	db := getDB()
	tx := db.Begin()

	ch, AllotedUser, err := getUserExclusively(ipoBid.UserId)
	l.Debugf("Acquired")

	defer func() {
		close(ch)
		l.Debugf("Released exclusive write on user")
	}()

	if err != nil {
		l.Errorf("Error acquiring user. Failing. %+v", err)
		return InternalServerError
	}

	oldReservedCash := AllotedUser.ReservedCash
	oldCash := AllotedUser.Cash

	AllotedUser.Cash += uint64(cost)
	AllotedUser.ReservedCash -= uint64(cost)

	l.Infof("Saving IpoRefundTransaction, IpoStockId : %d, SlotQuantity : %d, UserId : %d, Cost: %d", ipoBid.IpoStockId, ipoBid.SlotQuantity, ipoBid.UserId, cost)

	if err := tx.Save(ipoBid).Error; err != nil {
		AllotedUser.Cash = oldCash
		AllotedUser.ReservedCash = oldReservedCash
		l.Error(err)
		return err
	}

	if err := tx.Save(&AllotedUser).Error; err != nil {
		AllotedUser.Cash = oldCash
		AllotedUser.ReservedCash = oldReservedCash
		l.Error(err)
		return err
	}

	if err := tx.Create(IpoRefundTransaction).Error; err != nil {
		AllotedUser.Cash = oldCash
		AllotedUser.ReservedCash = oldReservedCash
		l.Error(err)
		return err
	}

	if err := tx.Commit().Error; err != nil {
		AllotedUser.Cash = oldCash
		AllotedUser.ReservedCash = oldReservedCash
		l.Errorf("Error committing transaction %+v", err)
		tx.Rollback()
		return err
	}

	transactionsStream := datastreamsManager.GetTransactionsStream()
	transactionsStream.SendTransaction(IpoRefundTransaction.ToProto())

	return nil
}
