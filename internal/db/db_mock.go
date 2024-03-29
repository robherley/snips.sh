// Code generated by mockery v2.26.1. DO NOT EDIT.

package db

import (
	context "context"

	snips "github.com/robherley/snips.sh/internal/snips"
	mock "github.com/stretchr/testify/mock"
)

// MockDB is an autogenerated mock type for the DB type
type MockDB struct {
	mock.Mock
}

type MockDB_Expecter struct {
	mock *mock.Mock
}

func (_m *MockDB) EXPECT() *MockDB_Expecter {
	return &MockDB_Expecter{mock: &_m.Mock}
}

// CreateFile provides a mock function with given fields: ctx, file, maxFiles
func (_m *MockDB) CreateFile(ctx context.Context, file *snips.File, maxFiles uint64) error {
	ret := _m.Called(ctx, file, maxFiles)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *snips.File, uint64) error); ok {
		r0 = rf(ctx, file, maxFiles)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockDB_CreateFile_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateFile'
type MockDB_CreateFile_Call struct {
	*mock.Call
}

// CreateFile is a helper method to define mock.On call
//   - ctx context.Context
//   - file *snips.File
//   - maxFiles uint64
func (_e *MockDB_Expecter) CreateFile(ctx interface{}, file interface{}, maxFiles interface{}) *MockDB_CreateFile_Call {
	return &MockDB_CreateFile_Call{Call: _e.mock.On("CreateFile", ctx, file, maxFiles)}
}

func (_c *MockDB_CreateFile_Call) Run(run func(ctx context.Context, file *snips.File, maxFiles uint64)) *MockDB_CreateFile_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*snips.File), args[2].(uint64))
	})
	return _c
}

func (_c *MockDB_CreateFile_Call) Return(_a0 error) *MockDB_CreateFile_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDB_CreateFile_Call) RunAndReturn(run func(context.Context, *snips.File, uint64) error) *MockDB_CreateFile_Call {
	_c.Call.Return(run)
	return _c
}

// CreateUserWithPublicKey provides a mock function with given fields: ctx, publickey
func (_m *MockDB) CreateUserWithPublicKey(ctx context.Context, publickey *snips.PublicKey) (*snips.User, error) {
	ret := _m.Called(ctx, publickey)

	var r0 *snips.User
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *snips.PublicKey) (*snips.User, error)); ok {
		return rf(ctx, publickey)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *snips.PublicKey) *snips.User); ok {
		r0 = rf(ctx, publickey)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*snips.User)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *snips.PublicKey) error); ok {
		r1 = rf(ctx, publickey)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDB_CreateUserWithPublicKey_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateUserWithPublicKey'
type MockDB_CreateUserWithPublicKey_Call struct {
	*mock.Call
}

// CreateUserWithPublicKey is a helper method to define mock.On call
//   - ctx context.Context
//   - publickey *snips.PublicKey
func (_e *MockDB_Expecter) CreateUserWithPublicKey(ctx interface{}, publickey interface{}) *MockDB_CreateUserWithPublicKey_Call {
	return &MockDB_CreateUserWithPublicKey_Call{Call: _e.mock.On("CreateUserWithPublicKey", ctx, publickey)}
}

func (_c *MockDB_CreateUserWithPublicKey_Call) Run(run func(ctx context.Context, publickey *snips.PublicKey)) *MockDB_CreateUserWithPublicKey_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*snips.PublicKey))
	})
	return _c
}

func (_c *MockDB_CreateUserWithPublicKey_Call) Return(_a0 *snips.User, _a1 error) *MockDB_CreateUserWithPublicKey_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDB_CreateUserWithPublicKey_Call) RunAndReturn(run func(context.Context, *snips.PublicKey) (*snips.User, error)) *MockDB_CreateUserWithPublicKey_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteFile provides a mock function with given fields: ctx, id
func (_m *MockDB) DeleteFile(ctx context.Context, id string) error {
	ret := _m.Called(ctx, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockDB_DeleteFile_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteFile'
type MockDB_DeleteFile_Call struct {
	*mock.Call
}

// DeleteFile is a helper method to define mock.On call
//   - ctx context.Context
//   - id string
func (_e *MockDB_Expecter) DeleteFile(ctx interface{}, id interface{}) *MockDB_DeleteFile_Call {
	return &MockDB_DeleteFile_Call{Call: _e.mock.On("DeleteFile", ctx, id)}
}

func (_c *MockDB_DeleteFile_Call) Run(run func(ctx context.Context, id string)) *MockDB_DeleteFile_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockDB_DeleteFile_Call) Return(_a0 error) *MockDB_DeleteFile_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDB_DeleteFile_Call) RunAndReturn(run func(context.Context, string) error) *MockDB_DeleteFile_Call {
	_c.Call.Return(run)
	return _c
}

// FindFile provides a mock function with given fields: ctx, id
func (_m *MockDB) FindFile(ctx context.Context, id string) (*snips.File, error) {
	ret := _m.Called(ctx, id)

	var r0 *snips.File
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*snips.File, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *snips.File); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*snips.File)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDB_FindFile_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'FindFile'
type MockDB_FindFile_Call struct {
	*mock.Call
}

// FindFile is a helper method to define mock.On call
//   - ctx context.Context
//   - id string
func (_e *MockDB_Expecter) FindFile(ctx interface{}, id interface{}) *MockDB_FindFile_Call {
	return &MockDB_FindFile_Call{Call: _e.mock.On("FindFile", ctx, id)}
}

func (_c *MockDB_FindFile_Call) Run(run func(ctx context.Context, id string)) *MockDB_FindFile_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockDB_FindFile_Call) Return(_a0 *snips.File, _a1 error) *MockDB_FindFile_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDB_FindFile_Call) RunAndReturn(run func(context.Context, string) (*snips.File, error)) *MockDB_FindFile_Call {
	_c.Call.Return(run)
	return _c
}

// FindFilesByUser provides a mock function with given fields: ctx, userID
func (_m *MockDB) FindFilesByUser(ctx context.Context, userID string) ([]*snips.File, error) {
	ret := _m.Called(ctx, userID)

	var r0 []*snips.File
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]*snips.File, error)); ok {
		return rf(ctx, userID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []*snips.File); ok {
		r0 = rf(ctx, userID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*snips.File)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, userID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDB_FindFilesByUser_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'FindFilesByUser'
type MockDB_FindFilesByUser_Call struct {
	*mock.Call
}

// FindFilesByUser is a helper method to define mock.On call
//   - ctx context.Context
//   - userID string
func (_e *MockDB_Expecter) FindFilesByUser(ctx interface{}, userID interface{}) *MockDB_FindFilesByUser_Call {
	return &MockDB_FindFilesByUser_Call{Call: _e.mock.On("FindFilesByUser", ctx, userID)}
}

func (_c *MockDB_FindFilesByUser_Call) Run(run func(ctx context.Context, userID string)) *MockDB_FindFilesByUser_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockDB_FindFilesByUser_Call) Return(_a0 []*snips.File, _a1 error) *MockDB_FindFilesByUser_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDB_FindFilesByUser_Call) RunAndReturn(run func(context.Context, string) ([]*snips.File, error)) *MockDB_FindFilesByUser_Call {
	_c.Call.Return(run)
	return _c
}

// FindPublicKeyByFingerprint provides a mock function with given fields: ctx, fingerprint
func (_m *MockDB) FindPublicKeyByFingerprint(ctx context.Context, fingerprint string) (*snips.PublicKey, error) {
	ret := _m.Called(ctx, fingerprint)

	var r0 *snips.PublicKey
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*snips.PublicKey, error)); ok {
		return rf(ctx, fingerprint)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *snips.PublicKey); ok {
		r0 = rf(ctx, fingerprint)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*snips.PublicKey)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, fingerprint)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDB_FindPublicKeyByFingerprint_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'FindPublicKeyByFingerprint'
type MockDB_FindPublicKeyByFingerprint_Call struct {
	*mock.Call
}

// FindPublicKeyByFingerprint is a helper method to define mock.On call
//   - ctx context.Context
//   - fingerprint string
func (_e *MockDB_Expecter) FindPublicKeyByFingerprint(ctx interface{}, fingerprint interface{}) *MockDB_FindPublicKeyByFingerprint_Call {
	return &MockDB_FindPublicKeyByFingerprint_Call{Call: _e.mock.On("FindPublicKeyByFingerprint", ctx, fingerprint)}
}

func (_c *MockDB_FindPublicKeyByFingerprint_Call) Run(run func(ctx context.Context, fingerprint string)) *MockDB_FindPublicKeyByFingerprint_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockDB_FindPublicKeyByFingerprint_Call) Return(_a0 *snips.PublicKey, _a1 error) *MockDB_FindPublicKeyByFingerprint_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDB_FindPublicKeyByFingerprint_Call) RunAndReturn(run func(context.Context, string) (*snips.PublicKey, error)) *MockDB_FindPublicKeyByFingerprint_Call {
	_c.Call.Return(run)
	return _c
}

// FindUser provides a mock function with given fields: ctx, id
func (_m *MockDB) FindUser(ctx context.Context, id string) (*snips.User, error) {
	ret := _m.Called(ctx, id)

	var r0 *snips.User
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*snips.User, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *snips.User); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*snips.User)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDB_FindUser_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'FindUser'
type MockDB_FindUser_Call struct {
	*mock.Call
}

// FindUser is a helper method to define mock.On call
//   - ctx context.Context
//   - id string
func (_e *MockDB_Expecter) FindUser(ctx interface{}, id interface{}) *MockDB_FindUser_Call {
	return &MockDB_FindUser_Call{Call: _e.mock.On("FindUser", ctx, id)}
}

func (_c *MockDB_FindUser_Call) Run(run func(ctx context.Context, id string)) *MockDB_FindUser_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockDB_FindUser_Call) Return(_a0 *snips.User, _a1 error) *MockDB_FindUser_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDB_FindUser_Call) RunAndReturn(run func(context.Context, string) (*snips.User, error)) *MockDB_FindUser_Call {
	_c.Call.Return(run)
	return _c
}

// Migrate provides a mock function with given fields: ctx
func (_m *MockDB) Migrate(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockDB_Migrate_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Migrate'
type MockDB_Migrate_Call struct {
	*mock.Call
}

// Migrate is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockDB_Expecter) Migrate(ctx interface{}) *MockDB_Migrate_Call {
	return &MockDB_Migrate_Call{Call: _e.mock.On("Migrate", ctx)}
}

func (_c *MockDB_Migrate_Call) Run(run func(ctx context.Context)) *MockDB_Migrate_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockDB_Migrate_Call) Return(_a0 error) *MockDB_Migrate_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDB_Migrate_Call) RunAndReturn(run func(context.Context) error) *MockDB_Migrate_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateFile provides a mock function with given fields: ctx, file
func (_m *MockDB) UpdateFile(ctx context.Context, file *snips.File) error {
	ret := _m.Called(ctx, file)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *snips.File) error); ok {
		r0 = rf(ctx, file)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockDB_UpdateFile_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateFile'
type MockDB_UpdateFile_Call struct {
	*mock.Call
}

// UpdateFile is a helper method to define mock.On call
//   - ctx context.Context
//   - file *snips.File
func (_e *MockDB_Expecter) UpdateFile(ctx interface{}, file interface{}) *MockDB_UpdateFile_Call {
	return &MockDB_UpdateFile_Call{Call: _e.mock.On("UpdateFile", ctx, file)}
}

func (_c *MockDB_UpdateFile_Call) Run(run func(ctx context.Context, file *snips.File)) *MockDB_UpdateFile_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*snips.File))
	})
	return _c
}

func (_c *MockDB_UpdateFile_Call) Return(_a0 error) *MockDB_UpdateFile_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDB_UpdateFile_Call) RunAndReturn(run func(context.Context, *snips.File) error) *MockDB_UpdateFile_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewMockDB interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockDB creates a new instance of MockDB. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockDB(t mockConstructorTestingTNewMockDB) *MockDB {
	mock := &MockDB{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
