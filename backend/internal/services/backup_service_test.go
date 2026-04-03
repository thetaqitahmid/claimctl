package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/thetaqitahmid/claimctl/internal/testutils"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDBPool satisfies the DBPool interface
type MockDBPool struct {
	mock.Mock
}

func (m *MockDBPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	calledArgs := m.Called(ctx, sql, args)
	return calledArgs.Get(0).(pgx.Rows), calledArgs.Error(1)
}

func (m *MockDBPool) Begin(ctx context.Context) (pgx.Tx, error) {
	calledArgs := m.Called(ctx)
	return calledArgs.Get(0).(pgx.Tx), calledArgs.Error(1)
}

// MockTx satisfies the pgx.Tx interface (subset needed)
type MockTx struct {
	mock.Mock
}

func (m *MockTx) Begin(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	return args.Get(0).(pgx.Tx), args.Error(1)
}

func (m *MockTx) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTx) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	args := m.Called(ctx, tableName, columnNames, rowSrc)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	args := m.Called(ctx, b)
	return args.Get(0).(pgx.BatchResults)
}

func (m *MockTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (m *MockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	args := m.Called(ctx, name, sql)
	return args.Get(0).(*pgconn.StatementDescription), args.Error(1)
}

func (m *MockTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	// Flexible argument matching for dynamic queries
	args := m.Called(ctx, sql, arguments)
	return args.Get(0).(pgconn.CommandTag), args.Error(1)
}

func (m *MockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	calledArgs := m.Called(ctx, sql, args)
	return calledArgs.Get(0).(pgx.Rows), calledArgs.Error(1)
}

func (m *MockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	calledArgs := m.Called(ctx, sql, args)
	return calledArgs.Get(0).(pgx.Row)
}

func (m *MockTx) Conn() *pgx.Conn {
	return nil
}

// Custom Mock for Rows because Scan/Values logic is complex to mock generically
type TestRows struct {
	data    []map[string]interface{}
	columns []string // Ordered column names
	idx     int
	closed  bool
	err     error
}

func NewTestRows(data []map[string]interface{}, columns []string) *TestRows {
	return &TestRows{
		data:    data,
		columns: columns,
	}
}

func (r *TestRows) Close()                        { r.closed = true }
func (r *TestRows) Err() error                    { return r.err }
func (r *TestRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }
func (r *TestRows) FieldDescriptions() []pgconn.FieldDescription {
	fds := make([]pgconn.FieldDescription, len(r.columns))
	for i, name := range r.columns {
		fds[i] = pgconn.FieldDescription{Name: name}
	}
	return fds
}
func (r *TestRows) Next() bool {
	if r.idx < len(r.data) {
		r.idx++
		return true
	}
	return false
}
func (r *TestRows) Scan(dest ...any) error { return nil } // Not used by service
func (r *TestRows) Values() ([]any, error) {
	if r.idx > 0 && r.idx <= len(r.data) {
		rowMap := r.data[r.idx-1]
		values := make([]any, len(r.columns))
		for i, col := range r.columns {
			values[i] = rowMap[col]
		}
		return values, nil
	}
	return nil, errors.New("no data")
}
func (r *TestRows) RawValues() [][]byte { return nil }
func (r *TestRows) Conn() *pgx.Conn     { return nil }

func TestBackupService_CreateBackup(t *testing.T) {
	mockPool := new(MockDBPool)
	service := NewBackupService(mockPool)

	ctx := context.Background()

	// 1. Setup Mock Data
	// For tables other than api_tokens, verify simple empty return
	// For api_tokens, return data with int32 ID to verify new schema handling
	apiTokenData := []map[string]interface{}{
		{
			"id":         testutils.TestUUID(1),
			"user_id":    int32(5),
			"name":       "Test Token",
			"token_hash": "hash123",
			"created_at": time.Now(),
		},
	}

	// 2. Expect Queries
	// We loop through tableQueries to setup expectations
	for _, tq := range tableQueries {
		if tq.Name == "api_tokens" {
			// Create SPECIFIC rows instance for this call
			rows := NewTestRows(apiTokenData, []string{"id", "user_id", "name", "token_hash", "created_at"})
			mockPool.On("Query", ctx, tq.Query, mock.Anything).Return(rows, nil).Once()
		} else {
			// Create NEW rows instance for each call
			rows := NewTestRows([]map[string]interface{}{}, []string{"id"})
			mockPool.On("Query", ctx, tq.Query, mock.Anything).Return(rows, nil).Once()
		}
	}

	// 3. Execute
	backup, err := service.CreateBackup(ctx)

	// 4. Assert
	assert.NoError(t, err)
	assert.NotNil(t, backup)

	// Verify API Tokens were backed up correctly
	assert.Len(t, backup.APITokens, 1)
	assert.Equal(t, testutils.TestUUID(1), backup.APITokens[0]["id"])
	assert.Equal(t, int32(5), backup.APITokens[0]["user_id"])
	assert.Equal(t, "Test Token", backup.APITokens[0]["name"])

	mockPool.AssertExpectations(t)
}

func TestBackupService_RestoreBackup(t *testing.T) {
	mockPool := new(MockDBPool)
	service := NewBackupService(mockPool)
	mockTx := new(MockTx)

	ctx := context.Background()
	backupJSON := []byte(`{
		"metadata": {
			"version": "1",
			"timestamp": 12345,
			"note": "test"
		},
		"api_tokens": [
			{"id": 1, "name": "Restored Token", "user_id": 5, "token_hash": "abc"}
		],
		"spaces": []
	}`)

	// 1. Transaction Begin
	mockPool.On("Begin", ctx).Return(mockTx, nil).Once()

	// 2. Truncate Tables
	for _, table := range truncateOrder {
		mockTx.On("Exec", ctx, "TRUNCATE TABLE "+table+" CASCADE", mock.Anything).Return(pgconn.CommandTag{}, nil).Once()
	}

	// 3. Insert Data
	// Match INSERT INTO claimctl.api_tokens
	mockTx.On("Exec", ctx, mock.MatchedBy(func(sql string) bool {
		return len(sql) >= 31 && sql[:31] == "INSERT INTO claimctl.api_tokens"
	}), mock.Anything).Return(pgconn.CommandTag{}, nil).Once()

	// 4. Sequence Reset
	// Match "SELECT setval"
	// We expect 13 sequence resets
	mockTx.On("Exec", ctx, mock.MatchedBy(func(sql string) bool {
		return len(sql) >= 13 && sql[:13] == "SELECT setval"
	}), mock.Anything).Return(pgconn.CommandTag{}, nil).Times(13)

	// 5. Commit
	mockTx.On("Commit", ctx).Return(nil).Once()

	// Expect Rollback (defer)
	mockTx.On("Rollback", ctx).Return(nil).Once()

	// Execute
	err := service.RestoreBackup(ctx, backupJSON)

	// Assert
	assert.NoError(t, err)
	mockPool.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}
