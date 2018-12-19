// Code generated by MockGen. DO NOT EDIT.
// Source: MarketDepth.go

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockMarketDepthStream is a mock of MarketDepthStream interface
type MockMarketDepthStream struct {
	ctrl     *gomock.Controller
	recorder *MockMarketDepthStreamMockRecorder
}

// MockMarketDepthStreamMockRecorder is the mock recorder for MockMarketDepthStream
type MockMarketDepthStreamMockRecorder struct {
	mock *MockMarketDepthStream
}

// NewMockMarketDepthStream creates a new mock instance
func NewMockMarketDepthStream(ctrl *gomock.Controller) *MockMarketDepthStream {
	mock := &MockMarketDepthStream{ctrl: ctrl}
	mock.recorder = &MockMarketDepthStreamMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockMarketDepthStream) EXPECT() *MockMarketDepthStreamMockRecorder {
	return m.recorder
}

// AddListener mocks base method
func (m *MockMarketDepthStream) AddListener(done <-chan struct{}, updates chan interface{}, sessionId string) {
	m.ctrl.Call(m, "AddListener", done, updates, sessionId)
}

// AddListener indicates an expected call of AddListener
func (mr *MockMarketDepthStreamMockRecorder) AddListener(done, updates, sessionId interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddListener", reflect.TypeOf((*MockMarketDepthStream)(nil).AddListener), done, updates, sessionId)
}

// RemoveListener mocks base method
func (m *MockMarketDepthStream) RemoveListener(sessionId string) {
	m.ctrl.Call(m, "RemoveListener", sessionId)
}

// RemoveListener indicates an expected call of RemoveListener
func (mr *MockMarketDepthStreamMockRecorder) RemoveListener(sessionId interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveListener", reflect.TypeOf((*MockMarketDepthStream)(nil).RemoveListener), sessionId)
}

// AddOrder mocks base method
func (m *MockMarketDepthStream) AddOrder(isMarket, isAsk bool, price, stockQuantity uint32) {
	m.ctrl.Call(m, "AddOrder", isMarket, isAsk, price, stockQuantity)
}

// AddOrder indicates an expected call of AddOrder
func (mr *MockMarketDepthStreamMockRecorder) AddOrder(isMarket, isAsk, price, stockQuantity interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddOrder", reflect.TypeOf((*MockMarketDepthStream)(nil).AddOrder), isMarket, isAsk, price, stockQuantity)
}

// AddTrade mocks base method
func (m *MockMarketDepthStream) AddTrade(price, qty uint32, createdAt string) {
	m.ctrl.Call(m, "AddTrade", price, qty, createdAt)
}

// AddTrade indicates an expected call of AddTrade
func (mr *MockMarketDepthStreamMockRecorder) AddTrade(price, qty, createdAt interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddTrade", reflect.TypeOf((*MockMarketDepthStream)(nil).AddTrade), price, qty, createdAt)
}

// CloseOrder mocks base method
func (m *MockMarketDepthStream) CloseOrder(isMarket, isAsk bool, price, stockQuantity uint32) {
	m.ctrl.Call(m, "CloseOrder", isMarket, isAsk, price, stockQuantity)
}

// CloseOrder indicates an expected call of CloseOrder
func (mr *MockMarketDepthStreamMockRecorder) CloseOrder(isMarket, isAsk, price, stockQuantity interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseOrder", reflect.TypeOf((*MockMarketDepthStream)(nil).CloseOrder), isMarket, isAsk, price, stockQuantity)
}
