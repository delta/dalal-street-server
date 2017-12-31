package models

import (
	"testing"

	"github.com/thakkarparth007/dalal-street-server/utils/test"
)

func TestMarketEventToProto(t *testing.T) {
	o := &MarketEvent{
		Id:           2,
		StockId:      3,
		Headline:     "Hello",
		Text:         "Hello World",
		IsGlobal:     true,
		EmotionScore: -54,
		CreatedAt:    "2017-02-09T00:00:00",
	}

	o_proto := o.ToProto()

	if !testutils.AssertEqual(t, o, o_proto) {
		t.Fatal("Converted value not equal")
	}
}

func Test_GetMarketEvents(t *testing.T) {
	marketEvent := &MarketEvent{
		Id:           1,
		StockId:      3,
		Headline:     "Hello",
		Text:         "Hello World",
		IsGlobal:     true,
		EmotionScore: -54,
		CreatedAt:    "2017-02-09T00:00:00",
	}
	db, err := DbOpen()
	if err != nil {
		t.Fatalf("Opening Database for inserting MarketEvents failed with error %v", err)
	}
	defer func() {
		db.Exec("DELETE FROM MarketEvents")
		db.Close()
	}()
	count := 0
	for ; marketEvent.Id <= MARKET_EVENT_COUNT+1; marketEvent.Id++ {
		db.Save(marketEvent)
		count++
	}
	var lastId uint32 = 0
	for hasMore := true; hasMore; {
		dbHasMore, retrievedEvents, err := GetMarketEvents(lastId, 13)
		if err != nil {
			t.Fatalf("GetMarketEvents returned an error %v", err)
		}

		for _, v := range retrievedEvents {
			lastId = v.Id - 1
			count--
		}
		hasMore = dbHasMore
	}
	if count != 0 {
		t.Fatalf("Inserted and Recovered events not equal. Added-Recieved = %v", count)
	}
	_, single, err := GetMarketEvents(2, 1)
	if err != nil {
		t.Fatalf("GetMarketEvents returned an error %v", err)
	}
	marketEvent.Id = 2
	if !testutils.AssertEqual(t, marketEvent, single[0]) {
		t.Fatalf("Expected %v but got %v", marketEvent, single[0])
	}
}
func Test_AddMarketEvent(t *testing.T) {
	marketEvent := &MarketEvent{
		Id:       1,
		StockId:  3,
		Headline: "Hello",
		Text:     "Hello World",
		IsGlobal: true,
	}
	db, err := DbOpen()
	if err != nil {
		t.Fatalf("Error Opening the Db")
	}
	defer func() {
		db.Exec("DELETE FROM MarketEvents")
		db.Close()
	}()
	if err != nil {
		t.Fatalf("GetMarketEvents returned an error %v", err)
	}

	AddMarketEvent(3, "Hello", "Hello World", true)
	retrievedEvent := &MarketEvent{}
	db.First(retrievedEvent)
	if retrievedEvent == nil {
		t.Fatalf("Added Event Not Found")
	}
	marketEvent.CreatedAt = retrievedEvent.CreatedAt
	marketEvent.Id = retrievedEvent.Id
	if !testutils.AssertEqual(t, retrievedEvent, marketEvent) {
		t.Fatalf("Expected %v but got %v", marketEvent, retrievedEvent)
	}
}
