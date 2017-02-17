package models

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"sync"
	"testing"
)
//helper function to return an Ask object
func makeAsk(userId uint32, stockId uint32, ot OrderType, stockQty uint32, price uint32) *Ask {
	return &Ask{
		UserId:        userId,
		StockId:       stockId,
		OrderType:     ot,
		StockQuantity: stockQty,
		Price:         price,
	}
}


//helper function to return a Bid object
func makeBid(userId uint32, stockId uint32, ot OrderType, stockQty uint32, price uint32) *Bid {
	return &Bid{
		UserId:        userId,
		StockId:       stockId,
		OrderType:     ot,
		StockQuantity: stockQty,
		Price:         price,
	}
}

func TestBidPQueue_init(t *testing.T) {
	pqueue := NewBidPQueue(MAXPQ)

	assert.Equal(
		t,
		len(pqueue.items), 1,
		"len(pqueue.items) == %d; want %d", len(pqueue.items), 1,
	)

	assert.Equal(
		t,
		pqueue.Size(), 0,
		"pqueue.Size() = %d; want %d", pqueue.Size(), 0,
	)

	assert.Equal(
		t,
		reflect.ValueOf(pqueue.comparator).Pointer(), reflect.ValueOf(bidComparator).Pointer(),
		"pqueue.comparator != bidComparator",
	)
}

func TestAskPQueue_init(t *testing.T) {
	pqueue := NewAskPQueue(MINPQ)

	assert.Equal(
		t,
		len(pqueue.items), 1,
		"len(pqueue.items) = %d; want %d", len(pqueue.items), 1,
	)

	assert.Equal(
		t,
		pqueue.Size(), 0,
		"pqueue.Size() = %d; want %d", pqueue.Size(), 0,
	)

	assert.Equal(
		t,
		reflect.ValueOf(pqueue.comparator).Pointer(), reflect.ValueOf(askComparator).Pointer(),
		"pqueue.comparator != askComparator",
	)
}

func TestBidPQueuePushAndPop_protects_max_order(t *testing.T) {
	pqueue := NewBidPQueue(MAXPQ)


	testcases := []struct {
		bid  *Bid
		price uint32
		quantity uint32
	}{
		{makeBid(2, 1, Limit, 5, 100), 100,5},
		{makeBid(2, 1, Limit, 2, 800), 800,2},
		{makeBid(2, 1, Limit, 3, 500), 500,3},
		{makeBid(2, 1, Limit, 11, 400), 400,11},
		{makeBid(2, 1, Limit, 10, 100), 100,10},
	}


	// Populate the test bid priority queue with dummy elements
	for i := 0; i < 5; i++ {
		pqueue.Push(testcases[i].bid, testcases[i].price, testcases[i].quantity)
	}

	var expectedPrice = []uint32{800,500,400,100,100}
	var expectedQty = []uint32{2,3,11,10,5}

	for i := 0; i <= 4; i++ {

		topBid := pqueue.Pop()
		assert.Equal(
			t,
			topBid.Price, expectedPrice[i],
			"price = %v; want %v", topBid.Price, expectedPrice[i],
		)
		assert.Equal(
			t,
			topBid.StockQuantity, expectedQty[i],
			"quantity = %v; want %v", topBid.StockQuantity, expectedQty[i],
		)
	}
}

func TestBidPQueuePushAndPop_concurrently_protects_max_order(t *testing.T) {
	var wg sync.WaitGroup

	pqueue := NewBidPQueue(MAXPQ)


	testcases := []struct {
		bid  *Bid
		price uint32
		quantity uint32
	}{
		{makeBid(2, 1, Limit, 5, 100), 100,5},
		{makeBid(2, 1, Limit, 2, 800), 800,2},
		{makeBid(2, 1, Limit, 3, 500), 500,3},
		{makeBid(2, 1, Limit, 11, 400), 400,11},
		{makeBid(2, 1, Limit, 10, 100), 100,10},
	}

	// Populate the test bid priority queue with dummy elements
	for i := 0; i < 5; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			pqueue.Push(testcases[i].bid, testcases[i].price, testcases[i].quantity)
		}(i)
	}

	wg.Wait()

	var expectedPrice = []uint32{800,500,400,100,100}
	var expectedQty = []uint32{2,3,11,10,5}

	for i := 0; i <= 4; i++ {

		topBid := pqueue.Pop()
		assert.Equal(
			t,
			topBid.Price, expectedPrice[i],
			"price = %v; want %v", topBid.Price, expectedPrice[i],
		)
		assert.Equal(
			t,
			topBid.StockQuantity, expectedQty[i],
			"quantity = %v; want %v", topBid.StockQuantity, expectedQty[i],
		)
	}
}

func TestAskPQueuePushAndPop_protects_min_order(t *testing.T) {
	pqueue := NewAskPQueue(MINPQ)

	testcases := []struct {
		ask  *Ask
		price uint32
		quantity uint32
	}{
		{makeAsk(2, 1, Limit, 5, 100), 100,5},
		{makeAsk(2, 1, Limit, 2, 800), 800,2},
		{makeAsk(2, 1, Limit, 3, 500), 500,3},
		{makeAsk(2, 1, Limit, 11, 400), 400,11},
		{makeAsk(2, 1, Limit, 10, 100), 100,10},
	}


	// Populate the test ask priority queue with dummy elements
	for i := 0; i < 5; i++ {
		pqueue.Push(testcases[i].ask, testcases[i].price, testcases[i].quantity)
	}

	var expectedPrice = []uint32{100,100,400,500,800}
	var expectedQty = []uint32{10,5,11,3,2}

	for i := 0; i <= 4; i++ {

		topAsk := pqueue.Pop()
		assert.Equal(
			t,
			topAsk.Price, expectedPrice[i],
			"price = %v; want %v", topAsk.Price, expectedPrice[i],
		)
		assert.Equal(
			t,
			topAsk.StockQuantity, expectedQty[i],
			"quantity = %v; want %v", topAsk.StockQuantity, expectedQty[i],
		)

	}
}

func TestAskPQueuePushAndPop_concurrently_protects_min_order(t *testing.T) {
	var wg sync.WaitGroup

	pqueue := NewAskPQueue(MINPQ)

	testcases := []struct {
		ask  *Ask
		price uint32
		quantity uint32
	}{
		{makeAsk(2, 1, Limit, 5, 100), 100,5},
		{makeAsk(2, 1, Limit, 2, 800), 800,2},
		{makeAsk(2, 1, Limit, 3, 500), 500,3},
		{makeAsk(2, 1, Limit, 11, 400), 400,11},
		{makeAsk(2, 1, Limit, 10, 100), 100,10},
	}

	// Populate the test ask priority queue with dummy elements
	for i := 0; i < 5; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			pqueue.Push(testcases[i].ask, testcases[i].price, testcases[i].quantity)
		}(i)
	}

	wg.Wait()

	var expectedPrice = []uint32{100,100,400,500,800}
	var expectedQty = []uint32{10,5,11,3,2}


	for i := 0; i <= 4; i++ {
		topAsk := pqueue.Pop()
		assert.Equal(
			t,
			topAsk.Price, expectedPrice[i],
			"price = %v; want %v", topAsk.Price, expectedPrice[i],
		)
		assert.Equal(
			t,
			topAsk.StockQuantity, expectedQty[i],
			"quantity = %v; want %v", topAsk.StockQuantity, expectedQty[i],
		)
	}
}

func TestBidPQueueHead_returns_max_element(t *testing.T) {
	pqueue := NewBidPQueue(MAXPQ)


	pqueue.Push(makeBid(2, 1, Limit, 5, 100), 100,5)
	pqueue.Push(makeBid(2, 1, Limit, 11, 400), 400,11)

	topBid := pqueue.Head()

	// First element of the binary heap is always left empty, so container
	// size is the number of elements actually stored + 1
	assert.Equal(t, len(pqueue.items), 3, "len(pqueue.items) = %d; want %d", len(pqueue.items), 3)

	assert.Equal(t, topBid.Price, uint32(400), "pqueue.Head().price = %v; want %v", topBid.Price, uint32(400))
	assert.Equal(t, topBid.StockQuantity, uint32(11), "pqueue.Head().StockQuantity = %v; want %v", topBid.StockQuantity, uint32(11))
}

func TestAskPQueueHead_returns_min_element(t *testing.T) {
	pqueue := NewAskPQueue(MINPQ)

	pqueue.Push(makeAsk(2, 1, Limit, 5, 100), 100,5)
	pqueue.Push(makeAsk(2, 1, Limit, 11, 400), 400,11)

	topAsk := pqueue.Head()

	// First element of the binary heap is always left empty, so container
	// size is the number of elements actually stored + 1
	assert.Equal(t, len(pqueue.items), 3, "len(pqueue.items) = %d; want %d", len(pqueue.items), 3)

	assert.Equal(t, topAsk.Price, uint32(100), "pqueue.Head().price = %v; want %v", topAsk.Price, uint32(100))
	assert.Equal(t, topAsk.StockQuantity, uint32(5), "pqueue.Head().StockQuantity = %v; want %v", topAsk.StockQuantity, uint32(5))
}
