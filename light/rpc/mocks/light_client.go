// Code generated by mockery v2.50.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	time "time"

	types "github.com/cometbft/cometbft/v2/types"
)

// LightClient is an autogenerated mock type for the LightClient type
type LightClient struct {
	mock.Mock
}

// ChainID provides a mock function with no fields
func (_m *LightClient) ChainID() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ChainID")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// TrustedLightBlock provides a mock function with given fields: height
func (_m *LightClient) TrustedLightBlock(height int64) (*types.LightBlock, error) {
	ret := _m.Called(height)

	if len(ret) == 0 {
		panic("no return value specified for TrustedLightBlock")
	}

	var r0 *types.LightBlock
	var r1 error
	if rf, ok := ret.Get(0).(func(int64) (*types.LightBlock, error)); ok {
		return rf(height)
	}
	if rf, ok := ret.Get(0).(func(int64) *types.LightBlock); ok {
		r0 = rf(height)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.LightBlock)
		}
	}

	if rf, ok := ret.Get(1).(func(int64) error); ok {
		r1 = rf(height)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Update provides a mock function with given fields: ctx, now
func (_m *LightClient) Update(ctx context.Context, now time.Time) (*types.LightBlock, error) {
	ret := _m.Called(ctx, now)

	if len(ret) == 0 {
		panic("no return value specified for Update")
	}

	var r0 *types.LightBlock
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, time.Time) (*types.LightBlock, error)); ok {
		return rf(ctx, now)
	}
	if rf, ok := ret.Get(0).(func(context.Context, time.Time) *types.LightBlock); ok {
		r0 = rf(ctx, now)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.LightBlock)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, time.Time) error); ok {
		r1 = rf(ctx, now)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// VerifyLightBlockAtHeight provides a mock function with given fields: ctx, height, now
func (_m *LightClient) VerifyLightBlockAtHeight(ctx context.Context, height int64, now time.Time) (*types.LightBlock, error) {
	ret := _m.Called(ctx, height, now)

	if len(ret) == 0 {
		panic("no return value specified for VerifyLightBlockAtHeight")
	}

	var r0 *types.LightBlock
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, int64, time.Time) (*types.LightBlock, error)); ok {
		return rf(ctx, height, now)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int64, time.Time) *types.LightBlock); ok {
		r0 = rf(ctx, height, now)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.LightBlock)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, int64, time.Time) error); ok {
		r1 = rf(ctx, height, now)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewLightClient creates a new instance of LightClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewLightClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *LightClient {
	mock := &LightClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
