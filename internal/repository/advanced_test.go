package repository_test

import (
	"context"
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"github.com/11DingKing/lab-scheduling/internal/testutil"
	"testing"
	"time"
)

func TestEquipmentPageLimitsAndOffsets(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	for i := 0; i < 7; i++ {
		if err := db.CreateEquipment(ctx, domain.Equipment{ID: string(rune('a' + i)), Name: string(rune('A' + i)), Kind: "bench", Capacity: 2, Status: domain.EquipmentReady, Version: 1}); err != nil {
			t.Fatal(err)
		}
	}
	page, err := db.EquipmentPage(ctx, "ready", 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 8 {
		t.Fatalf("total=%d", page.Total)
	}
	if len(page.Items) != 3 || page.Limit != 3 || page.Offset != 2 {
		t.Fatalf("page=%+v", page)
	}
	page, err = db.EquipmentPage(ctx, "ready", 1000, 99)
	if err != nil {
		t.Fatal(err)
	}
	if page.Limit != 100 || len(page.Items) != 0 {
		t.Fatalf("page=%+v", page)
	}
}
func TestEquipmentPageStatusFilter(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	if err := db.CreateEquipment(ctx, domain.Equipment{ID: "eq-page-off", Name: "Off", Kind: "bench", Capacity: 1, Status: domain.EquipmentOffline, Version: 1}); err != nil {
		t.Fatal(err)
	}
	page, err := db.EquipmentPage(ctx, "offline", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatal(page)
	}
}
func TestPingAndRecovery(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	n, err := db.RecoverExpiredSessions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal(n)
	}
	count, err := db.ActiveReservationCount(ctx, "eq-cnc")
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal(count)
	}
}
func TestExpiredSessionRecovery(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	_, err := db.SQL.ExecContext(ctx, "INSERT INTO sessions(id,user_id,token_hash,expires_at,created_at) VALUES('old','u-admin','old-token',datetime('now','-1 hour'),datetime('now','-2 hour'))")
	if err != nil {
		t.Fatal(err)
	}
	n, err := db.RecoverExpiredSessions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal(n)
	}
	var revoked string
	if err := db.SQL.QueryRowContext(ctx, "SELECT revoked_at FROM sessions WHERE id='old'").Scan(&revoked); err != nil {
		t.Fatal(err)
	}
	if revoked == "" {
		t.Fatal("not revoked")
	}
}
func TestPurgeExpiredIdempotency(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	_, err := db.SQL.ExecContext(ctx, "INSERT INTO idempotency_records(key,actor_id,request_hash,response_code,response_body,created_at,expires_at) VALUES('old-key','u-admin','hash',200,'{}',datetime('now','-2 hour'),datetime('now','-1 hour'))")
	if err != nil {
		t.Fatal(err)
	}
	removed, err := db.PurgeIdempotency(ctx, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatal(removed)
	}
}
