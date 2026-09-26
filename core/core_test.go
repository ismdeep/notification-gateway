package core

import (
	"context"
	"errors"
	"testing"

	"github.com/ismdeep/notification-gateway/core/input"
	"github.com/ismdeep/notification-gateway/core/model"
	notificationsender "github.com/ismdeep/notification-gateway/core/sender"
)

type coreTestSender struct {
	name    string
	sendErr error
	sent    []input.Message
}

func (s *coreTestSender) GetName(ctx context.Context) string {
	return s.name
}

func (s *coreTestSender) Send(ctx context.Context, msg input.Message) error {
	s.sent = append(s.sent, msg)
	return s.sendErr
}

type coreTestDatabase struct {
	alreadySent      map[string]bool
	alreadySentErrs  map[string]error
	markErr          error
	alreadySentCalls [][2]string
	marked           []coreTestMark
}

type coreTestMark struct {
	senderName string
	msg        input.Message
	status     int
}

func newCoreTestDatabase() *coreTestDatabase {
	return &coreTestDatabase{
		alreadySent:     make(map[string]bool),
		alreadySentErrs: make(map[string]error),
	}
}

func messageKey(senderName string, clientMessageID string) string {
	return senderName + "\x00" + clientMessageID
}

func (d *coreTestDatabase) MessageAlreadySent(senderName string, clientMessageID string) (bool, error) {
	d.alreadySentCalls = append(d.alreadySentCalls, [2]string{senderName, clientMessageID})
	if err := d.alreadySentErrs[senderName]; err != nil {
		return false, err
	}
	return d.alreadySent[messageKey(senderName, clientMessageID)], nil
}

func (d *coreTestDatabase) MessageMarkSendStatus(senderName string, msg input.Message, status int) error {
	d.marked = append(d.marked, coreTestMark{senderName: senderName, msg: msg, status: status})
	return d.markErr
}

func TestCore_Run(t *testing.T) {
	ctx := context.Background()

	in := make(chan input.Message, 1024)
	in <- input.Message{
		ClientMessageID: "1",
		Title:           "1",
		Content:         "1",
	}
	close(in)

	c := NewCore(in, nil, nil)
	if err := c.Run(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestNewCore(t *testing.T) {
	in := make(chan input.Message)
	sender := &coreTestSender{name: "test-sender"}
	db := newCoreTestDatabase()

	core := NewCore(in, []notificationsender.Sender{sender}, db)
	if core == nil {
		t.Fatal("NewCore returned nil")
	}
	if core.inputChan != in {
		t.Fatal("input channel was not assigned")
	}
	if len(core.senders) != 1 || core.senders[0] != sender {
		t.Fatal("sender was not assigned")
	}
	if core.database != db {
		t.Fatal("database was not assigned")
	}
}

func TestCore_ProcessInputMessage_Success(t *testing.T) {
	db := newCoreTestDatabase()
	sender := &coreTestSender{name: "test-sender"}
	core := NewCore(nil, []notificationsender.Sender{sender}, db)
	msg := input.Message{ClientMessageID: "client-1", Title: "title", Content: "content"}

	if err := core.processInputMessage(context.Background(), msg); err != nil {
		t.Fatalf("processInputMessage returned error: %v", err)
	}

	if len(sender.sent) != 1 || sender.sent[0] != msg {
		t.Fatalf("unexpected sent messages: %#v", sender.sent)
	}
	if len(db.alreadySentCalls) != 1 {
		t.Fatalf("unexpected MessageAlreadySent calls: %#v", db.alreadySentCalls)
	}
	if call := db.alreadySentCalls[0]; call != [2]string{"test-sender", "client-1"} {
		t.Fatalf("unexpected MessageAlreadySent args: %#v", call)
	}
	if len(db.marked) != 1 {
		t.Fatalf("unexpected marks: %#v", db.marked)
	}
	if mark := db.marked[0]; mark.senderName != "test-sender" || mark.msg != msg || mark.status != model.MessageSendSuccess {
		t.Fatalf("unexpected mark: %#v", mark)
	}
}

func TestCore_ProcessInputMessage_SkipsAlreadySent(t *testing.T) {
	db := newCoreTestDatabase()
	db.alreadySent[messageKey("test-sender", "client-1")] = true
	sender := &coreTestSender{name: "test-sender"}
	core := NewCore(nil, []notificationsender.Sender{sender}, db)
	msg := input.Message{ClientMessageID: "client-1", Title: "title", Content: "content"}

	if err := core.processInputMessage(context.Background(), msg); err != nil {
		t.Fatalf("processInputMessage returned error: %v", err)
	}

	if len(sender.sent) != 0 {
		t.Fatalf("already-sent message was sent again: %#v", sender.sent)
	}
	if len(db.marked) != 0 {
		t.Fatalf("already-sent message was marked again: %#v", db.marked)
	}
}

func TestCore_ProcessInputMessage_AlreadySentError(t *testing.T) {
	queryErr := errors.New("query failed")
	db := newCoreTestDatabase()
	db.alreadySentErrs["test-sender"] = queryErr
	sender := &coreTestSender{name: "test-sender"}
	core := NewCore(nil, []notificationsender.Sender{sender}, db)

	err := core.processInputMessage(context.Background(), input.Message{ClientMessageID: "client-1"})
	if !errors.Is(err, queryErr) {
		t.Fatalf("expected query error, got %v", err)
	}
	if len(sender.sent) != 0 {
		t.Fatalf("message was sent after query error: %#v", sender.sent)
	}
	if len(db.marked) != 0 {
		t.Fatalf("message was marked after query error: %#v", db.marked)
	}
}

func TestCore_ProcessInputMessage_SendErrorStillMarksSent(t *testing.T) {
	sendErr := errors.New("send failed")
	db := newCoreTestDatabase()
	sender := &coreTestSender{name: "test-sender", sendErr: sendErr}
	core := NewCore(nil, []notificationsender.Sender{sender}, db)
	msg := input.Message{ClientMessageID: "client-1"}

	err := core.processInputMessage(context.Background(), msg)
	if !errors.Is(err, sendErr) {
		t.Fatalf("expected send error, got %v", err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("expected one send attempt, got %#v", sender.sent)
	}
	if len(db.marked) != 1 || db.marked[0].msg != msg || db.marked[0].status != model.MessageSendFail {
		t.Fatalf("expected message to be marked after send error, got %#v", db.marked)
	}
}

func TestCore_ProcessInputMessage_MarkError(t *testing.T) {
	markErr := errors.New("mark failed")
	db := newCoreTestDatabase()
	db.markErr = markErr
	sender := &coreTestSender{name: "test-sender"}
	core := NewCore(nil, []notificationsender.Sender{sender}, db)

	err := core.processInputMessage(context.Background(), input.Message{ClientMessageID: "client-1"})
	if !errors.Is(err, markErr) {
		t.Fatalf("expected mark error, got %v", err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("expected message to be sent, got %#v", sender.sent)
	}
}

func TestCore_ProcessInputMessage_AggregatesErrorsAcrossSenders(t *testing.T) {
	firstErr := errors.New("first sender failed")
	secondErr := errors.New("second sender failed")
	db := newCoreTestDatabase()
	first := &coreTestSender{name: "first", sendErr: firstErr}
	second := &coreTestSender{name: "second", sendErr: secondErr}
	core := NewCore(nil, []notificationsender.Sender{first, second}, db)

	err := core.processInputMessage(context.Background(), input.Message{ClientMessageID: "client-1"})
	if !errors.Is(err, firstErr) || !errors.Is(err, secondErr) {
		t.Fatalf("expected both sender errors, got %v", err)
	}
	if len(first.sent) != 1 || len(second.sent) != 1 {
		t.Fatalf("expected both senders to be attempted, got %#v and %#v", first.sent, second.sent)
	}
	if len(db.marked) != 2 {
		t.Fatalf("expected both senders to be marked, got %#v", db.marked)
	}
}

func TestCore_ProcessInputMessage_ContinuesAfterQueryError(t *testing.T) {
	queryErr := errors.New("query failed")
	db := newCoreTestDatabase()
	first := &coreTestSender{name: "first"}
	second := &coreTestSender{name: "second"}
	core := NewCore(nil, []notificationsender.Sender{first, second}, db)

	db.alreadySentErrs["first"] = queryErr
	err := core.processInputMessage(context.Background(), input.Message{ClientMessageID: "client-1"})
	if !errors.Is(err, queryErr) {
		t.Fatalf("expected query error, got %v", err)
	}
	if len(first.sent) != 0 {
		t.Fatalf("first sender should not be called after query error: %#v", first.sent)
	}
	if len(second.sent) != 1 {
		t.Fatalf("second sender should still be attempted, got %#v", second.sent)
	}
	if len(db.marked) != 1 || db.marked[0].senderName != "second" {
		t.Fatalf("only second sender should be marked, got %#v", db.marked)
	}
}

func TestCore_Run_ProcessesMessagesAndIgnoresProcessErrors(t *testing.T) {
	ctx := context.Background()
	db := newCoreTestDatabase()
	sender := &coreTestSender{name: "test-sender", sendErr: errors.New("send failed")}
	core := NewCore(nil, []notificationsender.Sender{sender}, db)

	in := make(chan input.Message, 2)
	in <- input.Message{ClientMessageID: "client-1"}
	in <- input.Message{ClientMessageID: "client-2"}
	close(in)
	core.inputChan = in

	if err := core.Run(ctx); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(sender.sent) != 2 {
		t.Fatalf("expected 2 sent messages, got %#v", sender.sent)
	}
	if len(db.marked) != 2 {
		t.Fatalf("expected 2 marked messages, got %#v", db.marked)
	}
}
