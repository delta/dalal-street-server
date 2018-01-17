package models

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/jinzhu/gorm"

	"github.com/thakkarparth007/dalal-street-server/proto_build/actions"
	"github.com/thakkarparth007/dalal-street-server/proto_build/models"
)

type StockHistory struct {
	StockId   uint32 `gorm:"column:stockId;not null" json:"stock_id"`
	Close     uint32 `gorm:"column:close;not null" json:"close"`
	CreatedAt string `gorm:"column:createdAt;not null" json:"created_at"`
	Interval  uint32 `gorm:"column:intervalRecord;not null" json:"interval"`
	Open      uint32 `gorm:"column:open;not null" json:"open"`
	High      uint32 `gorm:"high:close;not null" json:"high"`
	Low       uint32 `gorm:"low:close;not null" json:"low"`
}

func (StockHistory) TableName() string {
	return "StockHistory"
}

func (gStockHistory *StockHistory) ToProto() *models_pb.StockHistory {
	return &models_pb.StockHistory{
		StockId:   gStockHistory.StockId,
		Close:     gStockHistory.Close,
		CreatedAt: gStockHistory.CreatedAt,
		Interval:  gStockHistory.Interval,
		Open:      gStockHistory.Open,
		High:      gStockHistory.High,
		Low:       gStockHistory.Low,
	}
}

// Resolution enum
type Resolution uint32

const (
	OneMinute     Resolution = 1   // Range will be 60*Resolution
	FiveMinutes   Resolution = 5   // Range will be 60*Resolution
	TenMinutes    Resolution = 10  // Range will be 60*Resolution
	ThirtyMinutes Resolution = 30  // Range will be 60*Resolution
	SixtyMinutes  Resolution = 60  // Range will be 60*Resolution
	OneDay        Resolution = 0   // Range will be 60*Resolution
	Other         Resolution = 123 //For individual transaction entries if reqd
)

// ResolutionFromProto converts proto StockHistoryResolution to a model's Resolution value
func ResolutionFromProto(s actions_pb.StockHistoryResolution) Resolution {
	if s == actions_pb.StockHistoryResolution_OneMinute {
		return OneMinute
	} else if s == actions_pb.StockHistoryResolution_FiveMinutes {
		return FiveMinutes
	} else if s == actions_pb.StockHistoryResolution_TenMinutes {
		return TenMinutes
	} else if s == actions_pb.StockHistoryResolution_ThirtyMinutes {
		return ThirtyMinutes
	} else if s == actions_pb.StockHistoryResolution_SixtyMinutes {
		return SixtyMinutes
	} else if s == actions_pb.StockHistoryResolution_OneDay {
		return OneDay
	}
	return Other
}

// ohlc represents ohlc for a given stock
type ohlc struct {
	open  uint32
	high  uint32
	low   uint32
	close uint32
}

var stopStockHistoryRecorderChan chan struct{}

func stopStockHistoryRecorder() {
	var l = logger.WithFields(logrus.Fields{
		"method": "stopStockHistoryRecorder",
	})

	l.Info("Stopping")

	close(stopStockHistoryRecorderChan)

	l.Info("Stopped")
}

// recordOneMinuteOHLC records one minute ohlc for each stock
func recordOneMinuteOHLC(db *gorm.DB, recordingTime time.Time) error {
	var l = logger.WithFields(logrus.Fields{
		"method": "recordOneMinuteOHLC",
	})

	l.Debugf("Recording one minute intervals for all stocks.")

	// Record the current minute's ohlc first.
	allStocks.RLock()
	defer allStocks.RUnlock()

	for stockId := range allStocks.m {
		allStocks.m[stockId].Lock()

		currentMinuteOHLC := &ohlc{
			allStocks.m[stockId].stock.open,
			allStocks.m[stockId].stock.high,
			allStocks.m[stockId].stock.low,
			allStocks.m[stockId].stock.CurrentPrice,
		}

		// Reset Open to previous Close
		// Set High,Low to Closing Price
		allStocks.m[stockId].stock.open = currentMinuteOHLC.close
		allStocks.m[stockId].stock.high = currentMinuteOHLC.close
		allStocks.m[stockId].stock.low = currentMinuteOHLC.close
		allStocks.m[stockId].Unlock()

		stkHistoryPoint := &StockHistory{
			StockId:   stockId,
			Close:     currentMinuteOHLC.close,
			Interval:  1,
			CreatedAt: recordingTime.UTC().Format(time.RFC3339), // TODO: Change to IST,
			Open:      currentMinuteOHLC.open,
			High:      currentMinuteOHLC.high,
			Low:       currentMinuteOHLC.low,
		}
		err := db.Save(stkHistoryPoint).Error
		if err != nil {
			l.Errorf("Error registering stock history point %+v. Error: %+v", stkHistoryPoint, err)
			return err
		}
	}

	l.Debugf("Recorded")

	return nil
}

func recordNMinuteOHLC(db *gorm.DB, stockId uint32, retrievedHistories []StockHistory, N uint32, recordingTime time.Time) error {
	var l = logger.WithFields(logrus.Fields{
		"method": "recordNMinuteOHLC",
	})
	modifiedRange := recordingTime.Add(time.Duration(N) * -time.Minute).UTC().Format(time.RFC3339)
	//modifiedRange represents the minimum allowed time greater than which records have to be taken into account
	var limitedRange int = len(retrievedHistories) - 1
	//Find that record for which it is satisfied ie now 0 to limitedRange are acceptable values
	for i := len(retrievedHistories) - 1; i >= 0; i-- {
		if retrievedHistories[i].CreatedAt > modifiedRange {
			limitedRange = i
			break
		}
	}
	//Initialize open to open of chronologically first open within range and close to close of the chronologically last history
	ohlcRecord := &ohlc{
		retrievedHistories[limitedRange].Open,
		retrievedHistories[limitedRange].Open,
		retrievedHistories[limitedRange].Open,
		retrievedHistories[0].Close,
	}
	// Iterate and find max of all max and min of all min
	for i := 0; i <= limitedRange; i++ {
		if ohlcRecord.high < retrievedHistories[i].High {
			ohlcRecord.high = retrievedHistories[i].High
		}
		if ohlcRecord.low > retrievedHistories[i].Low {
			ohlcRecord.low = retrievedHistories[i].Low
		}
	}
	// Save it
	stkHistoryPoint := &StockHistory{
		StockId:   stockId,
		Close:     ohlcRecord.close,
		Interval:  N,
		CreatedAt: recordingTime.UTC().Format(time.RFC3339), // TODO: Change to IST,
		Open:      ohlcRecord.open,
		High:      ohlcRecord.high,
		Low:       ohlcRecord.low,
	}
	err := db.Save(stkHistoryPoint)
	if err.Error != nil {
		l.Errorf("Error registering stock history point %+v. Error: %+v", stkHistoryPoint, err.Error)
		return err.Error
	}
	return nil
}

// retrieves relevant older number of records from the DB and then writes the required (5/10/30/60) intervals to the db
func recorderHigherIntervalOHLCs(db *gorm.DB, recordingTime time.Time) error {
	// 1. retrieve the required number of records from db
	// 2. do the recording

	// 1.a Find the max time range for which we need to retrieve the records
	minReqdTime := recordingTime      // Minimum Time after which records need to be retrieved
	currMin := recordingTime.Minute() // FIXME: In case of possible errors store Minute at start and keep incrementing it
	if currMin%60 == 0 {
		//Go through last 60 1 Minute recordings
		minReqdTime = minReqdTime.Add(-60 * time.Minute)
	} else if currMin%30 == 0 {
		//Go through last 30 1 Minute recordings
		minReqdTime = minReqdTime.Add(-30 * time.Minute)
	} else if currMin%10 == 0 {
		//Go through last 10 1 Minute recordings
		minReqdTime = minReqdTime.Add(-10 * time.Minute)
	} else if currMin%5 == 0 {
		//Go through last 5 1 Minute recordings
		minReqdTime = minReqdTime.Add(-5 * time.Minute)
	} else {
		return nil // no need to proceed furter
	}
	maxTimeRangeStr := minReqdTime.UTC().Format(time.RFC3339)

	// 2. now do the recording
	allStocks.RLock()
	for stockId := range allStocks.m {
		var retrievedHistories []StockHistory

		db = db.Where("intervalRecord = ? AND stockId = ? AND createdAt >= ?", 1, stockId, maxTimeRangeStr)
		db.Order("createdAt desc").Limit(TIMES_RESOLUTION).Find(&retrievedHistories)

		if currMin%5 == 0 {
			if err := recordNMinuteOHLC(db, stockId, retrievedHistories, 5, recordingTime); err != nil {
				return err
			}
		}
		if currMin%10 == 0 {
			if err := recordNMinuteOHLC(db, stockId, retrievedHistories, 10, recordingTime); err != nil {
				return err
			}
		}
		if currMin%30 == 0 {
			if err := recordNMinuteOHLC(db, stockId, retrievedHistories, 30, recordingTime); err != nil {
				return err
			}
		}
		if currMin%60 == 0 {
			if err := recordNMinuteOHLC(db, stockId, retrievedHistories, 60, recordingTime); err != nil {
				return err
			}
		}
	}
	allStocks.RUnlock()
	return nil
}

func GetStockHistory(stockId uint32, interval Resolution) ([]*StockHistory, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":   "GetStockHistory",
		"stockId":  stockId,
		"interval": interval,
	})

	l.Infof("Attempting to get stock History for stockId : %v", stockId)

	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return nil, err
	}
	defer db.Close()

	allStocks.m[stockId].RLock()
	defer allStocks.m[stockId].RUnlock()

	var histories []*StockHistory
	//Interval 0 refers to Day interval as of now. So until Day handling decided is not handled
	if interval != 0 {
		db = db.Where("stockId = ? AND intervalRecord = ?", stockId, interval)
		db = db.Order("createdAt desc")
		db = db.Limit(TIMES_RESOLUTION)
		db = db.Find(&histories)
		if err := db.Error; err != nil {
			return nil, err
		}
	}

	return histories, nil
}

func startStockHistoryRecorder(interval time.Duration) {
	var l = logger.WithFields(logrus.Fields{
		"method": "startStockHistoryRecorder",
	})

	l.Info("Starting")

	tickerChan := time.NewTicker(interval).C
	stopStockHistoryRecorderChan = make(chan struct{})

loop:
	for {
		select {
		case <-stopStockHistoryRecorderChan:
			break loop
		case <-tickerChan:
			// grab current time now, it will be used below to record the history
			// since multiple history records might be inserted, grabbing it here to keep times consistent
			currentTime := time.Now()

			db, err := DbOpen()
			if err != nil {
				l.Error(err)
				return
			}
			defer db.Close()

			// TODO: handle errors better
			// record the current minute's interval
			if err := recordOneMinuteOHLC(db, currentTime); err != nil {
				l.Errorf("Error recording one minute interval %+v", err)
			}

			// record higher minute intervals
			if err := recorderHigherIntervalOHLCs(db, currentTime); err != nil {
				l.Errorf("Error recording one higher minute intervals %+v", err)
			}

			l.Info("Recorded history")
		}
	}
}
