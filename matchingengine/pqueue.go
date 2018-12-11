package matchingengine

import (
	"sync"

	"github.com/delta/dalal-street-server/models"
)

// PQType represents a priority queue ordering kind (see MAXPQ and MINPQ)
type PQType int

// Constants to determine whether to create Max Priority Queue or Min Priority Queue
const (
	MAXPQ PQType = iota
	MINPQ
)

type bidItem struct {
	value    *models.Bid
	price    uint32
	quantity uint32
}

type askItem struct {
	value    *models.Ask
	price    uint32
	quantity uint32
}

// BidPQueue is a heap priority queue data structure implementation for Bids.
// It can be max or min ordered. It is synchronized and is safe for concurrent operations.
type BidPQueue struct {
	sync.RWMutex
	items      []*bidItem
	elemsCount int
	comparator func(factors, factors) bool
}

// AskPQueue is a heap priority queue data structure implementation for Asks.
// It can be max or min ordered. It is synchronized and is safe for concurrent operations.
type AskPQueue struct {
	sync.RWMutex
	items      []*askItem
	elemsCount int
	comparator func(factors, factors) bool
}

type factors struct {
	oType    models.OrderType
	price    uint32
	placedAt string
	quantity uint32
}

func newBidItem(value *models.Bid, price uint32, quantity uint32) *bidItem {
	return &bidItem{
		value:    value,
		price:    price,
		quantity: quantity,
	}
}

func newAskItem(value *models.Ask, price uint32, quantity uint32) *askItem {
	return &askItem{
		value:    value,
		price:    price,
		quantity: quantity,
	}
}

// NewBidPQueue creates a new priority queue with the ordering type specified by PQType
// and comprises of Bid items
func NewBidPQueue(pqType PQType) *BidPQueue {
	var cmp func(factors, factors) bool

	if pqType == MAXPQ {
		cmp = bidComparator
	} else {
		cmp = askComparator
	}

	items := make([]*bidItem, 1)
	items[0] = nil // Heap queue first element should always be nil

	return &BidPQueue{
		items:      items,
		elemsCount: 0,
		comparator: cmp,
	}
}

// NewAskPQueue creates a new priority queue with the ordering type specified by PQType
// and comprises of Ask items
func NewAskPQueue(pqType PQType) *AskPQueue {
	var cmp func(factors, factors) bool

	if pqType == MAXPQ {
		cmp = bidComparator
	} else {
		cmp = askComparator
	}

	items := make([]*askItem, 1)
	items[0] = nil // Heap queue first element should always be nil

	return &AskPQueue{
		items:      items,
		elemsCount: 0,
		comparator: cmp,
	}
}

// Push the value item into the priority queue with provided priority.
func (pq *BidPQueue) Push(value *models.Bid) {
	item := newBidItem(value, value.Price, value.StockQuantity)

	pq.Lock()
	pq.items = append(pq.items, item)
	pq.elemsCount++
	pq.swim(pq.size())
	pq.Unlock()
}

// Push the value item into the priority queue with provided priority.
func (pq *AskPQueue) Push(value *models.Ask) {
	item := newAskItem(value, value.Price, value.StockQuantity)

	pq.Lock()
	pq.items = append(pq.items, item)
	pq.elemsCount++
	pq.swim(pq.size())
	pq.Unlock()
}

// Pop and returns the highest/lowest priority item (depending on whether
// you're using a MINPQ or MAXPQ) from the priority queue
func (pq *BidPQueue) Pop() *models.Bid {
	pq.Lock()
	defer pq.Unlock()

	if pq.size() < 1 {
		return nil
	}

	max := pq.items[1]

	pq.exch(1, pq.size())
	pq.items = pq.items[0:pq.size()]
	pq.elemsCount--
	pq.sink(1)

	return max.value
}

// Pop and returns the highest/lowest priority item (depending on whether
// you're using a MINPQ or MAXPQ) from the priority queue
func (pq *AskPQueue) Pop() *models.Ask {
	pq.Lock()
	defer pq.Unlock()

	if pq.size() < 1 {
		return nil
	}

	max := pq.items[1]

	pq.exch(1, pq.size())
	pq.items = pq.items[0:pq.size()]
	pq.elemsCount--
	pq.sink(1)

	return max.value
}

// Head returns the highest/lowest priority item (depending on whether
// you're using a MINPQ or MAXPQ) from the priority queue
func (pq *BidPQueue) Head() *models.Bid {
	pq.RLock()
	defer pq.RUnlock()

	if pq.size() < 1 {
		return nil
	}

	headValue := pq.items[1].value

	return headValue
}

// Head returns the highest/lowest priority item (depending on whether
// you're using a MINPQ or MAXPQ) from the priority queue
func (pq *AskPQueue) Head() *models.Ask {
	pq.RLock()
	defer pq.RUnlock()

	if pq.size() < 1 {
		return nil
	}

	headValue := pq.items[1].value

	return headValue
}

// Size returns the number of elements present in the priority queue
func (pq *BidPQueue) Size() int {
	pq.RLock()
	defer pq.RUnlock()
	return pq.size()
}

// Size returns the number of elements present in the priority queue
func (pq *AskPQueue) Size() int {
	pq.RLock()
	defer pq.RUnlock()
	return pq.size()
}

// Empty checks if queue is empty
func (pq *BidPQueue) Empty() bool {
	pq.RLock()
	defer pq.RUnlock()
	return pq.size() == 0
}

// Empty checks if queue is empty
func (pq *AskPQueue) Empty() bool {
	pq.RLock()
	defer pq.RUnlock()
	return pq.size() == 0
}

func (pq *BidPQueue) size() int {
	return pq.elemsCount
}

func (pq *AskPQueue) size() int {
	return pq.elemsCount
}

/*
 *	bidComparator performs the following actions:
 *		- If exactly one of them is Market/StopLossActive, it has priority
 *		- If both are Market/StopLossActive, the older order has priority
 *		- If both are Limit orders, higher price has priority
 *		  If prices are equal, higher quantity has more priority
 *		- If both are Stoploss orders, higher price has priority for ask.
 */
func bidComparator(order1, order2 factors) bool {
	if isMarket(order1.oType) && isMarket(order2.oType) {
		return order2.placedAt < order1.placedAt
	}
	if isMarket(order1.oType) {
		return false
	}
	if isMarket(order2.oType) {
		return true
	}
	if order1.price == order2.price {
		return order2.quantity > order1.quantity
	}
	return order1.price < order2.price
}

/*
 *	askComparator performs the following actions:
 *		- If exactly one of them is Market/StopLossActive, it has priority
 *		- If both are Market/StopLossActive, the older order has priority
 *		- If both are Limit orders, lower price has priority
 *		  If prices are equal, higher quantity has more priority
 *		- If both are Stoploss orders, lower price has priority for bid.
 */
func askComparator(order1, order2 factors) bool {
	if isMarket(order1.oType) && isMarket(order2.oType) {
		return order2.placedAt < order1.placedAt
	}
	if isMarket(order1.oType) {
		return false
	}
	if isMarket(order2.oType) {
		return true
	}
	if order1.price == order2.price {
		return order2.quantity > order1.quantity
	}
	return order1.price > order2.price
}

func (pq *BidPQueue) less(i, j int) bool {
	return pq.comparator(
		factors{
			oType:    pq.items[i].value.OrderType,
			price:    pq.items[i].price,
			placedAt: pq.items[i].value.CreatedAt,
			quantity: pq.items[i].quantity,
		},
		factors{
			oType:    pq.items[j].value.OrderType,
			price:    pq.items[j].price,
			placedAt: pq.items[j].value.CreatedAt,
			quantity: pq.items[j].quantity,
		},
	)
}
func (pq *AskPQueue) less(i, j int) bool {
	return pq.comparator(
		factors{
			oType:    pq.items[i].value.OrderType,
			price:    pq.items[i].price,
			placedAt: pq.items[i].value.CreatedAt,
			quantity: pq.items[i].quantity,
		},
		factors{
			oType:    pq.items[j].value.OrderType,
			price:    pq.items[j].price,
			placedAt: pq.items[j].value.CreatedAt,
			quantity: pq.items[j].quantity,
		},
	)
}

func (pq *BidPQueue) exch(i, j int) {
	tmpItem := pq.items[i]

	pq.items[i] = pq.items[j]
	pq.items[j] = tmpItem
}

func (pq *AskPQueue) exch(i, j int) {
	tmpItem := pq.items[i]

	pq.items[i] = pq.items[j]
	pq.items[j] = tmpItem
}

func (pq *BidPQueue) swim(k int) {
	for k > 1 && pq.less(k/2, k) {
		pq.exch(k/2, k)
		k = k / 2
	}

}
func (pq *AskPQueue) swim(k int) {
	for k > 1 && pq.less(k/2, k) {
		pq.exch(k/2, k)
		k = k / 2
	}

}

func (pq *BidPQueue) sink(k int) {
	for 2*k <= pq.size() {
		j := 2 * k

		if j < pq.size() && pq.less(j, j+1) {
			j++
		}

		if !pq.less(k, j) {
			break
		}

		pq.exch(k, j)
		k = j
	}
}
func (pq *AskPQueue) sink(k int) {
	for 2*k <= pq.size() {
		j := 2 * k

		if j < pq.size() && pq.less(j, j+1) {
			j++
		}

		if !pq.less(k, j) {
			break
		}

		pq.exch(k, j)
		k = j
	}
}
