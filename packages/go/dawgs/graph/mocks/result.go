// Copyright 2023 Specter Ops, Inc.
//
// Licensed under the Apache License, Version 2.0
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

// Code generated by MockGen. DO NOT EDIT.
// Source: result.go

// Package graph_mocks is a generated GoMock package.
package graph_mocks

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockCursor is a mock of Cursor interface.
type MockCursor[T any] struct {
	ctrl     *gomock.Controller
	recorder *MockCursorMockRecorder[T]
}

// MockCursorMockRecorder is the mock recorder for MockCursor.
type MockCursorMockRecorder[T any] struct {
	mock *MockCursor[T]
}

// NewMockCursor creates a new mock instance.
func NewMockCursor[T any](ctrl *gomock.Controller) *MockCursor[T] {
	mock := &MockCursor[T]{ctrl: ctrl}
	mock.recorder = &MockCursorMockRecorder[T]{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCursor[T]) EXPECT() *MockCursorMockRecorder[T] {
	return m.recorder
}

// Chan mocks base method.
func (m *MockCursor[T]) Chan() chan T {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Chan")
	ret0, _ := ret[0].(chan T)
	return ret0
}

// Chan indicates an expected call of Chan.
func (mr *MockCursorMockRecorder[T]) Chan() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Chan", reflect.TypeOf((*MockCursor[T])(nil).Chan))
}

// Close mocks base method.
func (m *MockCursor[T]) Close() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Close")
}

// Close indicates an expected call of Close.
func (mr *MockCursorMockRecorder[T]) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockCursor[T])(nil).Close))
}

// Error mocks base method.
func (m *MockCursor[T]) Error() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Error")
	ret0, _ := ret[0].(error)
	return ret0
}

// Error indicates an expected call of Error.
func (mr *MockCursorMockRecorder[T]) Error() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Error", reflect.TypeOf((*MockCursor[T])(nil).Error))
}