package worker_test

import (
	"context"
	"errors"
	"github.com/11DingKing/lab-scheduling/internal/worker"
	"testing"
)

func TestDeadLettersCopyAndRecord(t *testing.T) {
	d := &worker.DeadLetters{}
	job := func(context.Context) error { return errors.New("permanent") }
	if err := worker.RunOnce(context.Background(), job, d); err == nil {
		t.Fatal("success")
	}
	items := d.List()
	if len(items) != 1 || items[0].LastError != "permanent" {
		t.Fatalf("items=%+v", items)
	}
	items[0].LastError = "changed"
	if d.List()[0].LastError != "permanent" {
		t.Fatal("list leaked backing state")
	}
}
func TestRunOnceSuccessDoesNotDeadLetter(t *testing.T) {
	d := &worker.DeadLetters{}
	if err := worker.RunOnce(context.Background(), func(context.Context) error { return nil }, d); err != nil {
		t.Fatal(err)
	}
	if len(d.List()) != 0 {
		t.Fatal("unexpected dead letter")
	}
}
