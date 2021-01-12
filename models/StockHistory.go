package models

import (
	"time"

	"github.com/sirupsen/logrus"
	"github.com/jinzhu/gorm"

	"github.com/delta/dalal-street-server/proto_build/actions"
	"github.com/delta/dalal-street-server/proto_build/models"
)

type StockHistory struct {
	StockId   uint32 `gorm:"column:stockId;not null" json:"stock_id"`
	Close     uint64 `gorm:"column:close;not null" json:"close"`
	CreatedAt string `gorm:"column:createdAt;not null" json:"created_at"`
	Interval  uint32 `gorm:"column:intervalRecord;not null" json:"interval"`
	Open      uint64 `gorm:"column:open;not null" json:"open"`
	High      uint64 `gorm:"column:high;not null" json:"high"`
	Low       uint64 `gorm:"column:low;not null" json:"low"`
	Volume    uint64 `gorm:"column:volume;not null" json:"volume"`
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
		Volume:    gStockHistory.Volume,
	}
}

// Resolution enum
type Resolution uint32

const (
	OneMinute      Resolution = 1   // Range will be 60*Resolution
	FiveMinutes    Resolution = 5   // Range will be 60*Resolution
	FifteenMinutes Resolution = 15  // Range will be 60*Resolution
	ThirtyMinutes  Resolution = 30  // Range will be 60*Resolution
	SixtyMinutes   Resolution = 60  // Range will be 60*Resolution
	OneDay         Resolution = 0   // Range will be 60*Resolution
	Other          Resolution = 123 //For individual transaction entries if reqd
)

// ResolutionFromProto converts proto StockHistoryResolution to a model's Resolution value
func ResolutionFromProto(s actions_pb.StockHistoryResolution) Resolution {
	if s == actions_pb.StockHistoryResolution_OneMinute {
		return OneMinute
	} else if s == actions_pb.StockHistoryResolution_FiveMinutes {
		return FiveMinutes
	} else if s == actions_pb.StockHistoryResolution_FifteenMinutes {
		return FifteenMinutes
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
type ohlcv struct {
	open   uint64
	high   uint64
	low    uint64
	close  uint64
	volume uint64
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

func stockHistoryStreamUpdate(stockId uint32, stockHistory *StockHistory) {
	stockStream := datastreamsManager.GetStockHistoryStream(stockId)
	pbStockHistory := stockHistory.ToProto()
	stockStream.SendStockHistoryUpdate(stockId, pbStockHistory)
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

		currentMinuteOHLCV := &ohlcv{
			allStocks.m[stockId].stock.open,
			allStocks.m[stockId].stock.high,
			allStocks.m[stockId].stock.low,
			allStocks.m[stockId].stock.CurrentPrice,
			allStocks.m[stockId].stock.volume,
		}

		// Reset Open to previous Close
		// Set High,Low to Closing Price
		allStocks.m[stockId].stock.open = currentMinuteOHLCV.close
		allStocks.m[stockId].stock.high = currentMinuteOHLCV.close
		allStocks.m[stockId].stock.low = currentMinuteOHLCV.close
		allStocks.m[stockId].stock.volume = 0
		allStocks.m[stockId].Unlock()

		stkHistoryPoint := &StockHistory{
			StockId:   stockId,
			Close:     currentMinuteOHLCV.close,
			Interval:  1,
			CreatedAt: recordingTime.UTC().Format(time.RFC3339), // TODO: Change to IST,
			Open:      currentMinuteOHLCV.open,
			High:      currentMinuteOHLCV.high,
			Low:       currentMinuteOHLCV.low,
			Volume:    currentMinuteOHLCV.volume,
		}
		err := db.Save(stkHistoryPoint).Error
		if err != nil {
			l.Errorf("Error registering stock history point %+v. Error: %+v", stkHistoryPoint, err)
			return err
		}
		stockHistoryStreamUpdate(stockId, stkHistoryPoint)
	}

	l.Debugf("Recorded")

	return nil
}

func recordNMinuteOHLC(db *gorm.DB, stockId uint32, retrievedHistories []StockHistory, N Resolution, recordingTime time.Time) error {
	var l = logger.WithFields(logrus.Fields{
		"method": "recordNMinuteOHLC",
	})

	// will happen in the starting, unless initial data is loaded
	// this will cause the server to crash if not handled
	if len(retrievedHistories) == 0 {
		return nil
	}

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
	ohlcvRecord := &ohlcv{
		retrievedHistories[limitedRange].Open,
		retrievedHistories[limitedRange].Open,
		retrievedHistories[limitedRange].Open,
		retrievedHistories[0].Close,
		0,
	}
	// Iterate and find max of all max and min of all min
	for i := 0; i <= limitedRange; i++ {
		if ohlcvRecord.high < retrievedHistories[i].High {
			ohlcvRecord.high = retrievedHistories[i].High
		}
		if ohlcvRecord.low > retrievedHistories[i].Low {
			ohlcvRecord.low = retrievedHistories[i].Low
		}
		ohlcvRecord.volume += retrievedHistories[i].Volume
	}
	// Save it
	stkHistoryPoint := &StockHistory{
		StockId:   stockId,
		Close:     ohlcvRecord.close,
		Interval:  uint32(N),
		CreatedAt: recordingTime.UTC().Format(time.RFC3339), // TODO: Change to IST,
		Open:      ohlcvRecord.open,
		High:      ohlcvRecord.high,
		Low:       ohlcvRecord.low,
		Volume:    ohlcvRecord.volume,
	}

	if err := db.Save(stkHistoryPoint).Error; err != nil {
		l.Errorf("Error registering stock history point %+v. Error: %+v", stkHistoryPoint, err)
		return err
	}
	stockHistoryStreamUpdate(stockId, stkHistoryPoint)

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
	} else if currMin%15 == 0 {
		//Go through last 15 1 Minute recordings
		minReqdTime = minReqdTime.Add(-15 * time.Minute)
	} else if currMin%5 == 0 {
		//Go through last 5 1 Minute recordings
		minReqdTime = minReqdTime.Add(-5 * time.Minute)
	} else {
		return nil // no need to proceed furter
	}
	maxTimeRangeStr := minReqdTime.UTC().Format(time.RFC3339)

	// 2. now do the recording
	allStocks.RLock()
	defer allStocks.RUnlock()

	for stockId := range allStocks.m {
		var retrievedHistories []StockHistory
		// Need to do this because db = db. chains the wheres
		dbCurrStock := db.Where("intervalRecord = ? AND stockId = ? AND createdAt >= ?", 1, stockId, maxTimeRangeStr)
		dbCurrStock.Order("createdAt desc").Limit(TIMES_RESOLUTION).Find(&retrievedHistories)

		if currMin%5 == 0 {
			if err := recordNMinuteOHLC(db, stockId, retrievedHistories, 5, recordingTime); err != nil {
				return err
			}
		}
		if currMin%15 == 0 {
			if err := recordNMinuteOHLC(db, stockId, retrievedHistories, 15, recordingTime); err != nil {
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

	return nil
}

func GetStockHistory(stockId uint32, interval Resolution) ([]*StockHistory, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":   "GetStockHistory",
		"stockId":  stockId,
		"interval": interval,
	})

	l.Infof("Attempting to get stock History for stockId : %v", stockId)

	db := getDB()

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

			db := getDB()

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
