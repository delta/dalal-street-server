package models

import (
	"fmt"
	"os"

	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/delta/dalal-street-server/utils"
	"github.com/sirupsen/logrus"
)

type MarketEvent struct {
	Id           uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	StockId      uint32 `gorm:"column:stockId" json:"stock_id"`
	EmotionScore int32  `gorm:"column:emotionScore;not null" json:"emotion_score"`
	Headline     string `gorm:"column:headline;not null" json:"headline"`
	Text         string `gorm:"column:text" json:"text"`
	IsGlobal     bool   `gorm:"column:isGlobal" json:"is_global"`
	ImagePath    string `gorm:"column:imagePath" json:"image_path"`
	CreatedAt    string `gorm:"column:createdAt;not null" json:"created_at"`
}

func (MarketEvent) TableName() string {
	return "MarketEvents"
}

func (gMarketEvent *MarketEvent) ToProto() *models_pb.MarketEvent {
	pMarketEvent := &models_pb.MarketEvent{
		Id:           gMarketEvent.Id,
		StockId:      gMarketEvent.StockId,
		Headline:     gMarketEvent.Headline,
		Text:         gMarketEvent.Text,
		EmotionScore: gMarketEvent.EmotionScore,
		IsGlobal:     gMarketEvent.IsGlobal,
		ImagePath:    gMarketEvent.ImagePath,
		CreatedAt:    gMarketEvent.CreatedAt,
	}
	return pMarketEvent
}

func GetMarketEvents(lastId, count, stockId uint32) (bool, []*MarketEvent, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetMarketEvents",
		"lastId": lastId,
		"count":  count,
	})

	l.Infof("Attempting to get market events")

	db := getDB()

	var marketEvents []*MarketEvent

	// fetching all the market events of the company
	// only if stockId is sent in req
	if stockId != 0 {
		if err := db.Find(&marketEvents, "stockId = ?", stockId).Error; err != nil {
			return true, nil, err
		}

		return false, marketEvents, nil
	}

	//set default value of count if it is zero
	if count == 0 {
		count = MARKET_EVENT_COUNT
	} else {
		count = utils.MinInt32(count, MARKET_EVENT_COUNT)
	}

	//get latest events if lastId is zero
	if lastId != 0 {
		db = db.Where("id <= ?", lastId)
	}
	if err := db.Order("id desc").Limit(count).Find(&marketEvents).Error; err != nil {
		return true, nil, err
	}

	var moreExists = len(marketEvents) >= int(count)
	l.Infof("Successfully fetched market events")
	return moreExists, marketEvents, nil
}

func AddMarketEvent(stockId uint32, headline, text string, isGlobal bool, imageURL string) error {
	var l = logger.WithFields(logrus.Fields{
		"method":         "AddMarketEvent",
		"param_stockId":  stockId,
		"param_headline": headline,
		"param_text":     text,
		"param_isGlobal": isGlobal,
		"param_imageURL": imageURL,
	})

	l.Infof("Attempting")

	err := utils.DownloadImage(imageURL)
	if err != nil {
		l.Error(err)
		return err
	}
	var basename = utils.GetImageBasename(imageURL)

	db := getDB()

	me := &MarketEvent{
		StockId:   stockId,
		Headline:  headline,
		Text:      text,
		IsGlobal:  isGlobal,
		ImagePath: basename,
		CreatedAt: utils.GetCurrentTimeISO8601(),
	}

	SendPushNotification(0, PushNotification{
		Title:    "Message from dalal street, something interesting just happened.",
		Message:  fmt.Sprintf("%v. Click here to know more.", headline),
		LogoUrl:  "",
		ImageUrl: imageURL,
	})

	if err = db.Save(me).Error; err != nil {
		l.Error(err)
		return err
	}

	l.Infof("Done")

	marketEventsStream := datastreamsManager.GetMarketEventsStream()
	marketEventsStream.SendMarketEvent(me.ToProto())

	return nil
}

func UpdateMarketEvent(stockId, oldNewsId uint32, headline, text string, isGlobal bool, imageURL string) error {

	if oldNewsId != 0 {

		var l = logger.WithFields(logrus.Fields{
			"method":          "UpdateMarketEvent",
			"param_stockId":   stockId,
			"param_headline":  headline,
			"param_text":      text,
			"param_isGlobal":  isGlobal,
			"param_imageURL":  imageURL,
			"param_oldNewsId": oldNewsId,
		})

		var basename = utils.GetImageBasename(imageURL)

		me := &MarketEvent{
			StockId:   stockId,
			Headline:  headline,
			Text:      text,
			IsGlobal:  isGlobal,
			ImagePath: basename,
		}

		db := getDB()

		l.Infof("Attempting to update existing market event ")

		OldEvent := &MarketEvent{}
		if err := db.First(OldEvent, oldNewsId).Error; err != nil {
			l.Error(err)
			return err
		}

		// If Image has to be changed
		if OldEvent.ImagePath != basename {
			// Delete old Image
			err := os.Remove(utils.GetImageBasePath() + OldEvent.ImagePath)
			if err != nil {
				l.Error(err)
				return err
			}

			// Download new image
			err = utils.DownloadImage(imageURL)
			if err != nil {
				l.Error(err)
				return err
			}
		}

		if err := db.Model(&OldEvent).Updates(me).Error; err != nil {
			l.Error(err)
			return err
		}
		// Update DB with new MarketEvent

		l.Infof("Done")

		marketEventsStream := datastreamsManager.GetMarketEventsStream()
		marketEventsStream.SendMarketEvent(me.ToProto())

	}
	return nil
}
