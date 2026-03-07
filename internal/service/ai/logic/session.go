// Package logic provides business logic for AI service
package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const sessionCacheTTL = 30 * time.Minute

// DefaultAISessionTitle is the default title for AI sessions
const DefaultAISessionTitle = "AI Session"

// AISession represents an AI chat session
type AISession struct {
	ID        string           `json:"id"`
	Scene     string           `json:"scene,omitempty"`
	Title     string           `json:"title"`
	Messages  []map[string]any `json:"messages"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

// SessionStore manages AI chat sessions
type SessionStore struct {
	db  *gorm.DB
	rdb redis.UniversalClient
	ttl time.Duration
}

// NewSessionStore creates a new SessionStore
func NewSessionStore(db *gorm.DB, rdb redis.UniversalClient) *SessionStore {
	return &SessionStore{db: db, rdb: rdb, ttl: sessionCacheTTL}
}

func (s *SessionStore) dbEnabled() bool { return s != nil && s.db != nil }

// AppendMessage appends a message to a session
func (s *SessionStore) AppendMessage(userID uint64, scene, sessionID string, message map[string]any) (*AISession, error) {
	now := time.Now()
	scene = NormalizeScene(scene)
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		sid = s.getOrCreateCurrentSessionID(userID, scene)
	}
	if sid == "" {
		sid = fmt.Sprintf("sess-%d", now.UnixNano())
	}

	if !s.dbEnabled() {
		return &AISession{ID: sid, Scene: scene, Title: DefaultAISessionTitle, Messages: []map[string]any{message}, CreatedAt: now, UpdatedAt: now}, nil
	}

	var sess model.AIChatSession
	err := s.db.Where("id = ? AND user_id = ?", sid, userID).First(&sess).Error
	switch {
	case err == nil:
		if sess.Scene == "" {
			sess.Scene = scene
		}
		sess.UpdatedAt = now
		if saveErr := s.db.Save(&sess).Error; saveErr != nil {
			return nil, saveErr
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		var exists int64
		if countErr := s.db.Model(&model.AIChatSession{}).Where("id = ?", sid).Count(&exists).Error; countErr != nil {
			return nil, countErr
		}
		if exists > 0 {
			return nil, errors.New("session not found")
		}
		sess = model.AIChatSession{ID: sid, UserID: userID, Scene: scene, Title: DefaultAISessionTitle, CreatedAt: now, UpdatedAt: now}
		if createErr := s.db.Create(&sess).Error; createErr != nil {
			return nil, createErr
		}
	default:
		return nil, err
	}

	msgID := strings.TrimSpace(toStringFromMap(message["id"]))
	if msgID == "" {
		msgID = fmt.Sprintf("msg-%d", time.Now().UnixNano())
	}
	role := strings.TrimSpace(toStringFromMap(message["role"]))
	content := toStringFromMap(message["content"])
	thinking := toStringFromMap(message["thinking"])
	createdAt := now
	if t, ok := message["timestamp"].(time.Time); ok {
		createdAt = t
	}
	if err := s.db.Create(&model.AIChatMessage{
		ID:        msgID,
		SessionID: sid,
		Role:      role,
		Content:   content,
		Thinking:  thinking,
		CreatedAt: createdAt,
	}).Error; err != nil {
		return nil, err
	}

	s.invalidateSessionCaches(userID, sid, scene)
	loaded := s.mustLoadSession(userID, sid)
	if loaded == nil {
		return nil, errors.New("session not found")
	}
	s.cacheSession(userID, loaded)
	return loaded, nil
}

// BranchSession creates a branch from an existing session
func (s *SessionStore) BranchSession(userID uint64, sourceSessionID, anchorMessageID, title string) (*AISession, error) {
	if !s.dbEnabled() {
		return nil, errors.New("db unavailable")
	}
	sourceID := strings.TrimSpace(sourceSessionID)
	if sourceID == "" {
		return nil, errors.New("source session id is required")
	}
	source, ok := s.GetSession(userID, sourceID)
	if !ok || source == nil {
		return nil, gorm.ErrRecordNotFound
	}

	cutoff := len(source.Messages) - 1
	anchorID := strings.TrimSpace(anchorMessageID)
	if anchorID != "" {
		found := -1
		for i := range source.Messages {
			if strings.TrimSpace(toStringFromMap(source.Messages[i]["id"])) == anchorID {
				found = i
				break
			}
		}
		if found < 0 {
			return nil, errors.New("anchor message not found")
		}
		cutoff = found
	}
	if cutoff < 0 {
		return nil, errors.New("source session has no messages")
	}

	newID := fmt.Sprintf("sess-%d", time.Now().UnixNano())
	newTitle := normalizeSessionTitle(title)
	if newTitle == "" {
		base := normalizeSessionTitle(source.Title)
		if base == "" {
			base = DefaultAISessionTitle
		}
		newTitle = normalizeSessionTitle("分支: " + base)
		if newTitle == "" {
			newTitle = "Branch Session"
		}
	}
	now := time.Now()

	err := s.db.Transaction(func(tx *gorm.DB) error {
		sessionModel := model.AIChatSession{
			ID:        newID,
			UserID:    userID,
			Scene:     source.Scene,
			Title:     newTitle,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := tx.Create(&sessionModel).Error; err != nil {
			return err
		}
		for i := 0; i <= cutoff; i++ {
			msg := source.Messages[i]
			msgTime := now
			if t, ok := msg["timestamp"].(time.Time); ok && !t.IsZero() {
				msgTime = t
			}
			msgModel := model.AIChatMessage{
				ID:        fmt.Sprintf("msg-%d-%d", time.Now().UnixNano(), i+1),
				SessionID: newID,
				Role:      strings.TrimSpace(toStringFromMap(msg["role"])),
				Content:   toStringFromMap(msg["content"]),
				Thinking:  toStringFromMap(msg["thinking"]),
				CreatedAt: msgTime,
			}
			if err := tx.Create(&msgModel).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.invalidateListCaches(userID, source.Scene)
	branched := s.mustLoadSession(userID, newID)
	if branched == nil {
		return nil, errors.New("branch session created but failed to load")
	}
	s.cacheSession(userID, branched)
	return branched, nil
}

// CurrentSession gets the current session for a user/scene
func (s *SessionStore) CurrentSession(userID uint64, scene string) (*AISession, bool) {
	if !s.dbEnabled() {
		return nil, false
	}
	key := s.currentSessionCacheKey(userID, scene)
	if out, ok := s.loadCachedSession(key); ok {
		return out, true
	}
	var sess model.AIChatSession
	if err := s.db.Where("user_id = ? AND scene = ?", userID, NormalizeScene(scene)).Order("updated_at DESC").First(&sess).Error; err != nil {
		return nil, false
	}
	loaded := s.mustLoadSession(userID, sess.ID)
	if loaded == nil {
		return nil, false
	}
	s.cacheSessionWithKey(key, loaded)
	s.cacheSession(userID, loaded)
	return loaded, true
}

// ListSessions lists sessions for a user
func (s *SessionStore) ListSessions(userID uint64, scene string) []*AISession {
	if !s.dbEnabled() {
		return nil
	}
	key := s.sessionListCacheKey(userID, scene)
	if out, ok := s.loadCachedSessionList(key); ok {
		return out
	}
	q := s.db.Where("user_id = ?", userID)
	if trim := strings.TrimSpace(scene); trim != "" {
		q = q.Where("scene = ?", NormalizeScene(trim))
	}
	var sessions []model.AIChatSession
	if err := q.Order("updated_at DESC").Find(&sessions).Error; err != nil {
		return nil
	}
	out := make([]*AISession, 0, len(sessions))
	for i := range sessions {
		if loaded := s.mustLoadSession(userID, sessions[i].ID); loaded != nil {
			out = append(out, loaded)
			s.cacheSession(userID, loaded)
		}
	}
	s.cacheSessionList(key, out)
	return out
}

// GetSession retrieves a session by ID
func (s *SessionStore) GetSession(userID uint64, id string) (*AISession, bool) {
	if !s.dbEnabled() {
		return nil, false
	}
	key := s.sessionCacheKey(userID, id)
	if out, ok := s.loadCachedSession(key); ok {
		return out, true
	}
	var sess model.AIChatSession
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&sess).Error; err != nil {
		return nil, false
	}
	loaded := s.mustLoadSession(userID, id)
	if loaded == nil {
		return nil, false
	}
	s.cacheSession(userID, loaded)
	return loaded, true
}

// DeleteSession deletes a session
func (s *SessionStore) DeleteSession(userID uint64, id string) {
	if !s.dbEnabled() {
		return
	}
	scene := ""
	_ = s.db.Transaction(func(tx *gorm.DB) error {
		var sess model.AIChatSession
		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&sess).Error; err != nil {
			return nil
		}
		scene = sess.Scene
		if err := tx.Where("session_id = ?", sess.ID).Delete(&model.AIChatMessage{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND user_id = ?", sess.ID, userID).Delete(&model.AIChatSession{}).Error
	})
	s.invalidateSessionCaches(userID, id, scene)
}

// UpdateSessionTitle updates session title
func (s *SessionStore) UpdateSessionTitle(userID uint64, id, title string) (*AISession, error) {
	if !s.dbEnabled() {
		return nil, errors.New("db unavailable")
	}
	sid := strings.TrimSpace(id)
	if sid == "" {
		return nil, errors.New("session id is required")
	}
	nextTitle := normalizeSessionTitle(title)
	if nextTitle == "" {
		return nil, errors.New("title is required")
	}
	var sess model.AIChatSession
	if err := s.db.Where("id = ? AND user_id = ?", sid, userID).First(&sess).Error; err != nil {
		return nil, err
	}
	sess.Title = nextTitle
	if err := s.db.Save(&sess).Error; err != nil {
		return nil, err
	}
	s.invalidateSessionCaches(userID, sid, sess.Scene)
	loaded := s.mustLoadSession(userID, sid)
	if loaded == nil {
		return nil, errors.New("session not found")
	}
	s.cacheSession(userID, loaded)
	return loaded, nil
}

func (s *SessionStore) getOrCreateCurrentSessionID(userID uint64, scene string) string {
	if !s.dbEnabled() {
		return ""
	}
	var sess model.AIChatSession
	if err := s.db.Where("user_id = ? AND scene = ?", userID, scene).Order("updated_at DESC").First(&sess).Error; err == nil {
		return sess.ID
	}
	return ""
}

func (s *SessionStore) mustLoadSession(userID uint64, id string) *AISession {
	if !s.dbEnabled() {
		return nil
	}
	var sess model.AIChatSession
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&sess).Error; err != nil {
		return nil
	}
	var msgs []model.AIChatMessage
	_ = s.db.Where("session_id = ?", id).Order("created_at ASC").Find(&msgs).Error
	arr := make([]map[string]any, 0, len(msgs))
	for i := range msgs {
		m := map[string]any{
			"id":        msgs[i].ID,
			"role":      msgs[i].Role,
			"content":   msgs[i].Content,
			"timestamp": msgs[i].CreatedAt,
		}
		if strings.TrimSpace(msgs[i].Thinking) != "" {
			m["thinking"] = msgs[i].Thinking
		}
		arr = append(arr, m)
	}
	return &AISession{
		ID:        sess.ID,
		Scene:     sess.Scene,
		Title:     sess.Title,
		Messages:  arr,
		CreatedAt: sess.CreatedAt,
		UpdatedAt: sess.UpdatedAt,
	}
}

func (s *SessionStore) sessionCacheKey(userID uint64, id string) string {
	return fmt.Sprintf("ai:session:%d:%s", userID, strings.TrimSpace(id))
}

func (s *SessionStore) sessionListCacheKey(userID uint64, scene string) string {
	return fmt.Sprintf("ai:session:list:%d:%s", userID, NormalizeScene(scene))
}

func (s *SessionStore) currentSessionCacheKey(userID uint64, scene string) string {
	return fmt.Sprintf("ai:session:current:%d:%s", userID, NormalizeScene(scene))
}

func (s *SessionStore) cacheSession(userID uint64, sess *AISession) {
	if sess == nil {
		return
	}
	s.cacheSessionWithKey(s.sessionCacheKey(userID, sess.ID), sess)
	s.cacheSessionWithKey(s.currentSessionCacheKey(userID, sess.Scene), sess)
}

func (s *SessionStore) cacheSessionWithKey(key string, sess *AISession) {
	if s == nil || s.rdb == nil || sess == nil {
		return
	}
	raw, err := json.Marshal(sess)
	if err != nil {
		return
	}
	_ = s.rdb.Set(context.Background(), key, raw, s.ttl).Err()
}

func (s *SessionStore) cacheSessionList(key string, sessions []*AISession) {
	if s == nil || s.rdb == nil {
		return
	}
	raw, err := json.Marshal(sessions)
	if err != nil {
		return
	}
	_ = s.rdb.Set(context.Background(), key, raw, s.ttl).Err()
}

func (s *SessionStore) loadCachedSession(key string) (*AISession, bool) {
	if s == nil || s.rdb == nil {
		return nil, false
	}
	raw, err := s.rdb.Get(context.Background(), key).Bytes()
	if err != nil {
		return nil, false
	}
	var out AISession
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, false
	}
	return &out, true
}

func (s *SessionStore) loadCachedSessionList(key string) ([]*AISession, bool) {
	if s == nil || s.rdb == nil {
		return nil, false
	}
	raw, err := s.rdb.Get(context.Background(), key).Bytes()
	if err != nil {
		return nil, false
	}
	var out []*AISession
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, false
	}
	return out, true
}

func (s *SessionStore) invalidateSessionCaches(userID uint64, id, scene string) {
	if s == nil || s.rdb == nil {
		return
	}
	keys := []string{
		s.sessionCacheKey(userID, id),
		s.sessionListCacheKey(userID, scene),
		s.currentSessionCacheKey(userID, scene),
	}
	_ = s.rdb.Del(context.Background(), keys...).Err()
}

func (s *SessionStore) invalidateListCaches(userID uint64, scene string) {
	if s == nil || s.rdb == nil {
		return
	}
	keys := []string{
		s.sessionListCacheKey(userID, scene),
		s.currentSessionCacheKey(userID, scene),
	}
	_ = s.rdb.Del(context.Background(), keys...).Err()
}

func toStringFromMap(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	case fmt.Stringer:
		return x.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func normalizeSessionTitle(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.Map(func(r rune) rune {
		if r < 32 {
			return -1
		}
		return r
	}, trimmed)
	rs := []rune(strings.TrimSpace(trimmed))
	if len(rs) > 64 {
		rs = rs[:64]
	}
	return strings.TrimSpace(string(rs))
}
