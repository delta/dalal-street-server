package models

import (
	"testing"

	testutils "github.com/delta/dalal-street-server/utils/test"
)

func TestMarketEventToProto(t *testing.T) {
	o := &MarketEvent{
		Id:           2,
		StockId:      3,
		Headline:     "Hello",
		Text:         "Hello World",
		IsGlobal:     true,
		EmotionScore: -54,
		ImagePath:    "bitcoin_1516197589.jpg",
		CreatedAt:    "2017-02-09T00:00:00",
	}

	oProto := o.ToProto()

	if !testutils.AssertEqual(t, o, oProto) {
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
		ImagePath:    "bitcoin_1516197589.jpg",
		CreatedAt:    "2017-02-09T00:00:00",
	}
	db := getDB()
	defer func() {
		db.Exec("DELETE FROM MarketEvents")
	}()
	count := 0
	for ; marketEvent.Id <= MARKET_EVENT_COUNT+1; marketEvent.Id++ {
		db.Save(marketEvent)
		count++
	}
	var lastId uint32 = 0
	for hasMore := true; hasMore; {
		dbHasMore, retrievedEvents, err := GetMarketEvents(lastId, 13, 0)
		if err != nil {
			t.Fatalf("GetMarketEvents returned an error %v", err)
		}
		count -= len(retrievedEvents)
		lastId = retrievedEvents[len(retrievedEvents)-1].Id - 1
		hasMore = dbHasMore
	}
	if count != 0 {
		t.Fatalf("Inserted and Recovered events not equal. Added-Received = %v", count)
	}
	_, single, err := GetMarketEvents(2, 1, 0)
	if len(single) != 1 {
		t.Fatalf("More than 1 Event Obtained")
	}
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
		Id:        1,
		StockId:   3,
		Headline:  "Hello",
		Text:      "Hello World",
		IsGlobal:  true,
		ImagePath: "bitcoin_1516197589.jpg",
	}
	db := getDB()
	defer func() {
		db.Exec("DELETE FROM MarketEvents")
	}()

	err := AddMarketEvent(3, "Hello", "Hello World", true, "http://www.valuewalk.com/wp-content/uploads/2018/01/bitcoin_1516197589.jpg")
	if err != nil {
		t.Fatalf("AddMarketEvent failed with error: %+v", err)
	}

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

func Test_UpdateMarketEvent(t *testing.T) {
	marketEvent := &MarketEvent{
		Id:        1,
		StockId:   3,
		Headline:  "Hello_new",
		Text:      "Hello World_new",
		IsGlobal:  true,
		ImagePath: "bitcoin_1516197589.jpg",
	}

	db := getDB()
	defer func() {
		db.Exec("DELETE FROM MarketEvents")
	}()

	// Add a market event with an "incorrect" set of details
	err := AddMarketEvent(2, "Hello_old", "Hello World_old", true, "http://www.valuewalk.com/wp-content/uploads/2018/01/bitcoin_1516197589.jpg")
	if err != nil {
		t.Fatalf("AddMarketEvent failed with error (in Test_UpdateMarketEvent): %+v", err)
	}

	// Update the market event with the "correct" set of details
	err = UpdateMarketEvent(3, 1, "Hello_new", "Hello World_new", true, "http://www.valuewalk.com/wp-content/uploads/2018/01/bitcoin_1516197589.jpg")
	if err != nil {
		t.Fatalf("Update MarketEvent failed with error: %+v", err)
	}

	retrievedEvent := &MarketEvent{}
	db.First(retrievedEvent)
	if retrievedEvent == nil {
		t.Fatalf("Added/Updated Event Not Found")
	}
	marketEvent.CreatedAt = retrievedEvent.CreatedAt
	marketEvent.Id = retrievedEvent.Id
	if !testutils.AssertEqual(t, retrievedEvent, marketEvent) {
		t.Fatalf("Expected %v but got %v", marketEvent, retrievedEvent)
	}
}
