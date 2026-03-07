package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cy77cc/k8s-manage/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DBCheckPointStore persists compose checkpoints in ai_checkpoints.
type DBCheckPointStore struct {
	db *gorm.DB
}

func NewDBCheckPointStore(db *gorm.DB) compose.CheckPointStore {
	return &DBCheckPointStore{db: db}
}

func (s *DBCheckPointStore) Set(ctx context.Context, key string, value []byte) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("checkpoint store db is nil")
	}

	record := &model.AICheckPoint{Key: key, Value: value}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(record).Error
	if err != nil {
		return fmt.Errorf("save checkpoint %q: %w", key, err)
	}
	return nil
}

func (s *DBCheckPointStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if s == nil || s.db == nil {
		return nil, false, fmt.Errorf("checkpoint store db is nil")
	}

	var cp model.AICheckPoint
	err := s.db.WithContext(ctx).Where("`key` = ?", key).First(&cp).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("get checkpoint %q: %w", key, err)
	}
	return cp.Value, true, nil
}
