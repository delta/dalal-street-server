package datastreams

import (
	"testing"

	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/golang/mock/gomock"
)

func GetMockMarketEventsStream(t *testing.T) (*gomock.Controller, *models_pb.MarketEvent, *marketEventsStream, string) {

	var sessionID = "123456789"
	var ID uint32 = 1
	var StockID uint32 = 2
	var HeadLine string = "headline"
	var Text string = "text"
	var EmotionScore int32 = 3
	var IsGlobal bool = false
	var ImagePath string = ""   //idk what to set ... 
	var CreatedAT string = "date"
	var sizecache int32
	var unrecognized []byte
	var nounkeyLiteralStruct struct{}

	mockControl := gomock.NewController(t)

	//getting error of undefined mocks , IDK but i added comments in broadcastStream class for mocking it !!!
	mockbroadcastStream := mocks.NewBroadcastStream(mockControl)

	mockMarketEvent := &models_pb.MarketEvent{
		Id:                   ID,
		StockId:              StockID,
		Headline:             HeadLine,
		Text:                 Text,
		EmotionScore:         EmotionScore,
		IsGlobal:             IsGlobal,
		CreatedAt:            CreatedAT,
		ImagePath:            ImagePath,
		XXX_NoUnkeyedLiteral: nounkeyLiteralStruct,
		XXX_unrecognized:     unrecognized,
		XXX_sizecache:        sizecache,
	}

	mockMarketEventStream := &marketEventsStream{
		broadcastStream: mockbroadcastStream,
	}

	return mockControl, mockMarketEvent, mockMarketEventStream, sessionID
}

func TestSendMarketEvent(t *testing.T) {
	mockControl, mockMarketEvent, mockMarketEventStream, sessionID := GetMockMarketEventsStream(t)

	update := mockMarketEvent
	//not sure for EXECT()
	//unexpectedly mocks has not been generated
	mockMarketEventStream.broadcastStream.EXPECT().BroadcastUpdate(update)

}
func TestMarketEventAddListener(t *testing.T) {

	var Done <-chan struct{}
	var UpDate chan interface{}

	mockControl, mockMarketEvent, mockMarketEventStream, sessionID := GetMockMarketEventsStream(t)

	mockMarketEventStream.broadcastStream.EXPECT().AddListener(sessionID, &listener{
		update: UpDate,
		done:   Done,
	})

}
func TestMarketEventRemoveListener(t *testing.T) {

	mockbroadcastStream, _, _, sessionID := GetMockMarketEventsStream(t)

	mockbroadcastStream.broadcastStream.EXPECT().RemoveListener(sessionID)

}
