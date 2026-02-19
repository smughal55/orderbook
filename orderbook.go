package main

import (
	"sort"
	"sync"
)

// OrderSide holds one side (bids or asks) of the order book.
// prices is kept sorted ascending at all times.
type OrderSide struct {
	levels map[float64]float64 // price -> amount
	prices []float64           // sorted ascending
}

func newOrderSide() OrderSide {
	return OrderSide{
		levels: make(map[float64]float64),
		prices: nil,
	}
}

// set inserts or updates a price level.
func (s *OrderSide) set(price, amount float64) {
	if _, exists := s.levels[price]; !exists {
		i := sort.SearchFloat64s(s.prices, price)
		s.prices = append(s.prices, 0)
		copy(s.prices[i+1:], s.prices[i:])
		s.prices[i] = price
	}
	s.levels[price] = amount
}

// remove deletes a price level.
func (s *OrderSide) remove(price float64) {
	if _, exists := s.levels[price]; !exists {
		return
	}
	delete(s.levels, price)
	i := sort.SearchFloat64s(s.prices, price)
	if i < len(s.prices) && s.prices[i] == price {
		s.prices = append(s.prices[:i], s.prices[i+1:]...)
	}
}

// min returns the lowest price on this side.
func (s *OrderSide) min() (float64, bool) {
	if len(s.prices) == 0 {
		return 0, false
	}
	return s.prices[0], true
}

// max returns the highest price on this side.
func (s *OrderSide) max() (float64, bool) {
	if len(s.prices) == 0 {
		return 0, false
	}
	return s.prices[len(s.prices)-1], true
}

func (s *OrderSide) reset() {
	s.levels = make(map[float64]float64)
	s.prices = s.prices[:0]
}

// OrderBook is a thread-safe order book with bid and ask sides.
type OrderBook struct {
	mu   sync.RWMutex
	bids OrderSide
	asks OrderSide
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		bids: newOrderSide(),
		asks: newOrderSide(),
	}
}

// ApplySnapshot replaces the entire book with the given levels.
func (ob *OrderBook) ApplySnapshot(bids, asks []PriceLevel) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	ob.bids.reset()
	for _, pl := range bids {
		ob.bids.set(pl.Price, pl.Amount)
	}

	ob.asks.reset()
	for _, pl := range asks {
		ob.asks.set(pl.Price, pl.Amount)
	}
}

// ApplyUpdate applies a single incremental update.
// amount == 0 means remove the price level.
func (ob *OrderBook) ApplyUpdate(side string, price, amount float64) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	s := &ob.bids
	if side == "ask" {
		s = &ob.asks
	}

	if amount == 0 {
		s.remove(price)
	} else {
		s.set(price, amount)
	}
}

// BestBid returns the highest bid price.
func (ob *OrderBook) BestBid() (float64, bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.bids.max()
}

// BestAsk returns the lowest ask price.
func (ob *OrderBook) BestAsk() (float64, bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.asks.min()
}

// MedianPrice returns (highestBid + lowestAsk) / 2.
// Returns false if either side is empty.
func (ob *OrderBook) MedianPrice() (float64, bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	bestBid, okBid := ob.bids.max()
	bestAsk, okAsk := ob.asks.min()
	if !okBid || !okAsk {
		return 0, false
	}
	return (bestBid + bestAsk) / 2, true
}
