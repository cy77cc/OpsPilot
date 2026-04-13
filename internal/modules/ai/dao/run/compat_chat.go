package run

import (
	aidaochat "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/chat"
	"gorm.io/gorm"
)

func NewAIChatDAO(db *gorm.DB) *aidaochat.AIChatDAO {
	return aidaochat.NewAIChatDAO(db)
}
