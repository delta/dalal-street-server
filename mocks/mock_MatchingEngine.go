// Code generated by MockGen. DO NOT EDIT.
// Source: ./matchingengine/MatchingEngine.go

// Package mocks is a generated GoMock package.
package mocks

import (
	models "github.com/delta/dalal-street-server/models"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockMatchingEngine is a mock of MatchingEngine interface
type MockMatchingEngine struct {
	ctrl     *gomock.Controller
	recorder *MockMatchingEngineMockRecorder
}

// MockMatchingEngineMockRecorder is the mock recorder for MockMatchingEngine
type MockMatchingEngineMockRecorder struct {
	mock *MockMatchingEngine
}

// NewMockMatchingEngine creates a new mock instance
func NewMockMatchingEngine(ctrl *gomock.Controller) *MockMatchingEngine {
	mock := &MockMatchingEngine{ctrl: ctrl}
	mock.recorder = &MockMatchingEngineMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockMatchingEngine) EXPECT() *MockMatchingEngineMockRecorder {
	return m.recorder
}

// AddAskOrder mocks base method
func (m *MockMatchingEngine) AddAskOrder(arg0 *models.Ask) {
	m.ctrl.Call(m, "AddAskOrder", arg0)
}

// AddAskOrder indicates an expected call of AddAskOrder
func (mr *MockMatchingEngineMockRecorder) AddAskOrder(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddAskOrder", reflect.TypeOf((*MockMatchingEngine)(nil).AddAskOrder), arg0)
}

// AddBidOrder mocks base method
func (m *MockMatchingEngine) AddBidOrder(arg0 *models.Bid) {
	m.ctrl.Call(m, "AddBidOrder", arg0)
}

// AddBidOrder indicates an expected call of AddBidOrder
func (mr *MockMatchingEngineMockRecorder) AddBidOrder(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddBidOrder", reflect.TypeOf((*MockMatchingEngine)(nil).AddBidOrder), arg0)
}

// CancelAskOrder mocks base method
func (m *MockMatchingEngine) CancelAskOrder(arg0 *models.Ask) {
	m.ctrl.Call(m, "CancelAskOrder", arg0)
}

// CancelAskOrder indicates an expected call of CancelAskOrder
func (mr *MockMatchingEngineMockRecorder) CancelAskOrder(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CancelAskOrder", reflect.TypeOf((*MockMatchingEngine)(nil).CancelAskOrder), arg0)
}

// CancelBidOrder mocks base method
func (m *MockMatchingEngine) CancelBidOrder(arg0 *models.Bid) {
	m.ctrl.Call(m, "CancelBidOrder", arg0)
}

// CancelBidOrder indicates an expected call of CancelBidOrder
func (mr *MockMatchingEngineMockRecorder) CancelBidOrder(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CancelBidOrder", reflect.TypeOf((*MockMatchingEngine)(nil).CancelBidOrder), arg0)
}
