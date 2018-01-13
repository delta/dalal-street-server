package models

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/jinzhu/gorm"

	"github.com/thakkarparth007/dalal-street-server/proto_build/actions"
	"github.com/thakkarparth007/dalal-street-server/proto_build/models"
)

type StockHistory struct {
	StockId    uint32 `gorm:"column:stockId;not null" json:"stock_id"`
	StockPrice uint32 `gorm:"column:stockPrice;not null" json:"stock_price"`
	CreatedAt  string `gorm:"column:createdAt;not null" json:"created_at"`
	Interval   uint32 `gorm:"column:interval_record;not null" json:"interval"`
	Open       uint32 `gorm:"column:open;not null" json:"open"`
	High       uint32 `gorm:"high:close;not null" json:"high"`
	Low        uint32 `gorm:"low:close;not null" json:"low"`
}

func (StockHistory) TableName() string {
	return "StockHistory"
}

func (gStockHistory *StockHistory) ToProto() *models_pb.StockHistory {
	return &models_pb.StockHistory{
		StockId:    gStockHistory.StockId,
		StockPrice: gStockHistory.StockPrice,
		CreatedAt:  gStockHistory.CreatedAt,
		Interval:   gStockHistory.Interval,
		Open:       gStockHistory.Open,
		High:       gStockHistory.High,
		Low:        gStockHistory.Low,
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
			StockId:    stockId,
			StockPrice: currentMinuteOHLC.close,
			Interval:   1,
			CreatedAt:  recordingTime.UTC().Format(time.RFC3339), // TODO: Change to IST,
			Open:       currentMinuteOHLC.open,
			High:       currentMinuteOHLC.high,
			Low:        currentMinuteOHLC.low,
		}
		err := db.Save(stkHistoryPoint).Error
		if err != nil {
			l.Errorf("Error registering stock history point %+v. Error: %+v", stkHistoryPoint, err)
			return err
		}
	}
	allStocks.RUnlock()

	l.Debugf("Recorded")

	return nil
}

func recordNMinuteOHLC(db *gorm.DB, stockId uint32, retrievedHistories []StockHistory, N uint32, recordingTime time.Time) error {
	var l = logger.WithFields(logrus.Fields{
		"method": "recordNMinuteOHLC",
	})
	modifiedRange := recordingTime.Add(time.Duration(N) * -time.Minute).UTC().Format(time.RFC3339)
	var limitedRange int = len(retrievedHistories) - 1
	for i := 0; i < len(retrievedHistories); i++ {
		if retrievedHistories[i].CreatedAt > modifiedRange {
			limitedRange = i
			break
		}
	}
	ohlcRecord := &ohlc{
		retrievedHistories[limitedRange].StockPrice,
		retrievedHistories[limitedRange].StockPrice,
		retrievedHistories[limitedRange].StockPrice,
		retrievedHistories[0].StockPrice,
	}
	for i := limitedRange; i >= 0; i-- {
		if ohlcRecord.high < retrievedHistories[i].StockPrice {
			ohlcRecord.high = retrievedHistories[i].StockPrice
		}
		if ohlcRecord.low < retrievedHistories[i].StockPrice {
			ohlcRecord.low = retrievedHistories[i].StockPrice
		}
	}
	stkHistoryPoint := &StockHistory{
		StockId:    stockId,
		StockPrice: ohlcRecord.close,
		Interval:   1,
		CreatedAt:  recordingTime.UTC().Format(time.RFC3339), // TODO: Change to IST,
		Open:       ohlcRecord.open,
		High:       ohlcRecord.high,
		Low:        ohlcRecord.low,
	}
	err := db.Save(stkHistoryPoint).Error
	if err != nil {
		l.Errorf("Error registering stock history point %+v. Error: %+v", stkHistoryPoint, err)
		return err
	}
	return nil
}

// retrieves relevant older number of records from the DB and then writes the required (5/10/30/60) intervals to the db
func recorderHigherIntervalOHLCs(db *gorm.DB, recordingTime time.Time) error {
	// 1. retrieve the required number of records from db
	// 2. do the recording

	// 1.a Find the max time range for which we need to retrieve the records
	maxTimeRange := recordingTime     // max time range for which stock history records need to be retrieved
	currMin := recordingTime.Minute() // FIXME: In case of possible errors store Minute at start and keep incrementing it
	if currMin%60 == 0 {
		//Go through last 60 1 Minute recordings
		maxTimeRange = maxTimeRange.Add(-60 * time.Minute)
	} else if currMin%30 == 0 {
		//Go through last 30 1 Minute recordings
		maxTimeRange = maxTimeRange.Add(-30 * time.Minute)
	} else if currMin%10 == 0 {
		//Go through last 10 1 Minute recordings
		maxTimeRange = maxTimeRange.Add(-10 * time.Minute)
	} else if currMin%5 == 0 {
		//Go through last 5 1 Minute recordings
		maxTimeRange = maxTimeRange.Add(-5 * time.Minute)
	} else {
		return nil // no need to proceed furter
	}
	maxTimeRangeStr := maxTimeRange.UTC().Format(time.RFC3339)

	// 2. now do the recording
	allStocks.RLock()
	for stockId := range allStocks.m {
		var retrievedHistories []StockHistory

		db = db.Where("interval_record = ? AND stock_id = ? AND created_at >= ?", 1, stockId, maxTimeRangeStr)
		db.Order("id desc").Limit(60).Find(&retrievedHistories)

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
