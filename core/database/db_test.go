package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ismdeep/notification-gateway/core/input"
	"github.com/ismdeep/notification-gateway/core/model"
)

const testDriverName = "notification-gateway-database-test"

var (
	testDriver       = newFakeDriver()
	registerTestOnce sync.Once
)

func newTestDB(t *testing.T) *DB {
	t.Helper()

	registerTestOnce.Do(func() {
		sql.Register(testDriverName, testDriver)
	})

	testDriver.Reset()
	t.Cleanup(testDriver.Reset)

	sqlDB, err := sql.Open(testDriverName, "test")
	if err != nil {
		t.Fatalf("open test sql db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	gormDB, err := gorm.Open(
		mysql.New(mysql.Config{
			Conn:                      sqlDB,
			SkipInitializeWithVersion: true,
		}),
		&gorm.Config{
			Logger:                 logger.Discard,
			SkipDefaultTransaction: true,
			DisableAutomaticPing:   true,
		},
	)
	if err != nil {
		t.Fatalf("open gorm db: %v", err)
	}

	return &DB{db: gormDB}
}

func TestNewDB_UnsupportedDialect(t *testing.T) {
	db, err := NewDB(DBConfig{
		Dialect: "oracle",
		DSN:     "oracle://user:pass@127.0.0.1:1521/service",
	})
	assert.Nil(t, db)
	assert.EqualError(t, err, "dialect oracle not supported, supported dialects: mysql, postgres, sqlite")
}

func TestDBSupportedDialects(t *testing.T) {
	assert.Equal(t, []string{"mysql", "postgres", "sqlite"}, (&DB{}).SupportedDialects())
}

func TestNewDB_SQLite(t *testing.T) {
	db, err := NewDB(DBConfig{Dialect: "sqlite", DSN: "file::memory:?cache=shared"})
	if err != nil {
		t.Fatalf("NewDB returned error: %v", err)
	}
	if db == nil || db.db == nil {
		t.Fatal("NewDB returned an uninitialized DB")
	}
	msg := input.Message{ClientMessageID: "sqlite-client", Title: "title", Content: "content"}
	if err := db.MessageMarkSendStatus("sqlite", msg, model.MessageSendSuccess); err != nil {
		t.Fatalf("mark message: %v", err)
	}
	sent, err := db.MessageAlreadySent("sqlite", msg.ClientMessageID)
	if err != nil || !sent {
		t.Fatalf("MessageAlreadySent = %v, %v; want true, nil", sent, err)
	}
}

func TestMessageAlreadySent(t *testing.T) {
	db := newTestDB(t)
	msg := input.Message{
		ClientMessageID: "client-message-id",
		Title:           "title",
		Content:         "content",
	}

	sent, err := db.MessageAlreadySent("sender", msg.ClientMessageID)
	assert.NoError(t, err)
	assert.False(t, sent)

	assert.NoError(t, db.MessageMarkSendStatus("sender", msg, model.MessageSendSuccess))

	sent, err = db.MessageAlreadySent("sender", msg.ClientMessageID)
	assert.NoError(t, err)
	assert.True(t, sent)

	sent, err = db.MessageAlreadySent("another-sender", msg.ClientMessageID)
	assert.NoError(t, err)
	assert.False(t, sent)

	sent, err = db.MessageAlreadySent("sender", "another-client-message-id")
	assert.NoError(t, err)
	assert.False(t, sent)
}

func TestMessageAlreadySent_QueryError(t *testing.T) {
	db := newTestDB(t)
	testDriver.setQueryError(errors.New("query failed"))

	sent, err := db.MessageAlreadySent("sender", "client-message-id")
	assert.False(t, sent)
	assert.EqualError(t, err, "failed to query sender message: query failed")
}

func TestMessageMarkedAsSent(t *testing.T) {
	db := newTestDB(t)
	msg := input.Message{
		ClientMessageID: "client-message-id",
		Title:           "title",
		Content:         "content",
	}

	err := db.MessageMarkSendStatus("sender", msg, model.MessageSendSuccess)
	assert.NoError(t, err)

	stored, ok := testDriver.message("sender", msg.ClientMessageID)
	assert.True(t, ok)
	assert.Equal(t, "sender", stored.SenderName)
	assert.Equal(t, msg.ClientMessageID, stored.ClientMessageID)
	assert.Equal(t, msg.Title, stored.Title)
	assert.Equal(t, msg.Content, stored.Content)
	assert.Equal(t, model.MessageSendSuccess, stored.SendStatus)
	assert.False(t, stored.CreatedAt.IsZero())
	assert.False(t, stored.UpdatedAt.IsZero())
}

func TestMessageMarkedAsSent_CreateError(t *testing.T) {
	db := newTestDB(t)
	testDriver.setExecError(errors.New("create failed"))

	err := db.MessageMarkSendStatus("sender", input.Message{ClientMessageID: "client-message-id"}, model.MessageSendFail)
	assert.EqualError(t, err, "failed to mark message as sent: create failed")
}

type messageKey struct {
	senderName      string
	clientMessageID string
}

type fakeDriver struct {
	mu       sync.Mutex
	messages map[messageKey]model.Message
	queryErr error
	execErr  error
}

func newFakeDriver() *fakeDriver {
	return &fakeDriver{
		messages: make(map[messageKey]model.Message),
	}
}

func (d *fakeDriver) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.messages = make(map[messageKey]model.Message)
	d.queryErr = nil
	d.execErr = nil
}

func (d *fakeDriver) setQueryError(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.queryErr = err
}

func (d *fakeDriver) setExecError(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.execErr = err
}

func (d *fakeDriver) message(senderName string, clientMessageID string) (model.Message, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	msg, ok := d.messages[messageKey{
		senderName:      senderName,
		clientMessageID: clientMessageID,
	}]
	return msg, ok
}

func (d *fakeDriver) store(msg model.Message) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.messages[messageKey{
		senderName:      msg.SenderName,
		clientMessageID: msg.ClientMessageID,
	}] = msg
}

func (d *fakeDriver) Open(string) (driver.Conn, error) {
	return &fakeConn{driver: d}, nil
}

type fakeConn struct {
	driver *fakeDriver
}

func (c *fakeConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (c *fakeConn) Close() error {
	return nil
}

func (c *fakeConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}

func (c *fakeConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.driver.mu.Lock()
	queryErr := c.driver.queryErr
	c.driver.mu.Unlock()
	if queryErr != nil {
		return nil, queryErr
	}

	query = strings.TrimSpace(query)
	if !strings.HasPrefix(query, "SELECT count(*) FROM `messages`") {
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
	if len(args) < 2 {
		return nil, fmt.Errorf("unexpected query args: %v", args)
	}

	senderName, ok := args[0].Value.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected sender name value type: %T", args[0].Value)
	}
	clientMessageID, ok := args[1].Value.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected client message id value type: %T", args[1].Value)
	}

	var count int64
	if _, ok := c.driver.message(senderName, clientMessageID); ok {
		count = 1
	}

	return &fakeRows{
		columns: []string{"count(*)"},
		rows:    [][]driver.Value{{count}},
	}, nil
}

func (c *fakeConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.driver.mu.Lock()
	execErr := c.driver.execErr
	c.driver.mu.Unlock()
	if execErr != nil {
		return nil, execErr
	}

	query = strings.TrimSpace(query)
	if !strings.HasPrefix(query, "INSERT INTO `messages`") {
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
	if len(args) < 7 {
		return nil, fmt.Errorf("unexpected query args: %v", args)
	}

	msg := model.Message{
		SenderName:      args[0].Value.(string),
		ClientMessageID: args[1].Value.(string),
		Title:           args[2].Value.(string),
		Content:         args[3].Value.(string),
		SendStatus:      int(args[4].Value.(int64)),
		CreatedAt:       args[5].Value.(time.Time),
		UpdatedAt:       args[6].Value.(time.Time),
	}

	c.driver.store(msg)

	return fakeResult{lastInsertID: 1, rowsAffected: 1}, nil
}

type fakeRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

func (r *fakeRows) Columns() []string {
	return r.columns
}

func (r *fakeRows) Close() error {
	return nil
}

func (r *fakeRows) Next(dest []driver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}

	row := r.rows[r.index]
	for i, value := range row {
		dest[i] = value
	}
	r.index++

	return nil
}

type fakeResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (r fakeResult) LastInsertId() (int64, error) {
	return r.lastInsertID, nil
}

func (r fakeResult) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}
