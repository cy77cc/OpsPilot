package cluster

import "gorm.io/gorm"

// Phase3Repository 是 Phase 3 领域仓储入口，后续任务中逐步补齐具体读写方法。
type Phase3Repository struct {
	db *gorm.DB
}

func NewPhase3Repository(db *gorm.DB) *Phase3Repository {
	return &Phase3Repository{db: db}
}
