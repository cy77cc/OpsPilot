package logic

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	model "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	"gorm.io/gorm"
)

func TestConsumeProbe_ConcurrentOnlyOneSucceeds(t *testing.T) {
	hostSvc, db := newHostLogicTestService(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open sql db handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(4)

	const userID = uint64(9001)
	probeToken := fmt.Sprintf("task8-concurrent-probe-token-%d", time.Now().UnixNano())
	if err := db.WithContext(context.Background()).Create(&model.HostProbeSession{
		TokenHash:      hashToken(probeToken),
		Name:           "task8-node",
		IP:             "10.10.0.8",
		Port:           22,
		AuthType:       "password",
		Username:       "root",
		PasswordCipher: "task8-cipher",
		Reachable:      true,
		FactsJSON:      `{}`,
		WarningsJSON:   `[]`,
		ExpiresAt:      time.Now().Add(5 * time.Minute),
		CreatedBy:      userID,
	}).Error; err != nil {
		t.Fatalf("seed probe session: %v", err)
	}

	var queried int32
	bothQueried := make(chan struct{})
	queryCallbackName := "test:task8_consume_probe_query_barrier"
	if err := db.Callback().Query().After("gorm:query").Register(queryCallbackName, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != "host_probe_sessions" {
			return
		}
		if atomic.AddInt32(&queried, 1) == 2 {
			close(bothQueried)
		}
		select {
		case <-bothQueried:
		case <-time.After(2 * time.Second):
			tx.AddError(errors.New("timeout waiting concurrent consumeProbe queries"))
		}
	}); err != nil {
		t.Fatalf("register query barrier callback: %v", err)
	}

	var updates int32
	updateCallbackName := "test:task8_consume_probe_update_delay_second"
	if err := db.Callback().Update().Before("gorm:update").Register(updateCallbackName, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != "host_probe_sessions" {
			return
		}
		if atomic.AddInt32(&updates, 1) == 2 {
			time.Sleep(10 * time.Millisecond)
		}
	}); err != nil {
		t.Fatalf("register update ordering callback: %v", err)
	}

	start := make(chan struct{})
	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, consumeErr := hostSvc.consumeProbe(context.Background(), userID, probeToken)
			errCh <- consumeErr
		}()
	}

	close(start)
	wg.Wait()
	close(errCh)

	successes := 0
	notFound := 0
	for consumeErr := range errCh {
		switch {
		case consumeErr == nil:
			successes++
		case consumeErr.Error() == "probe_not_found":
			notFound++
		default:
			t.Fatalf("consume probe returned unexpected error: %v", consumeErr)
		}
	}

	if successes != 1 || notFound != 1 {
		t.Fatalf("expected one success and one probe_not_found, got success=%d probe_not_found=%d", successes, notFound)
	}
}
