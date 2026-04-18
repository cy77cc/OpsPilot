package model

import "time"

type AIAlertIngestEvent struct {
	ID              string     `gorm:"column:id;type:varchar(64);primaryKey"`
	Source          string     `gorm:"column:source;type:varchar(64);not null;index:idx_ai_alert_ingest_source_fp,priority:1"`
	Protocol        string     `gorm:"column:protocol;type:varchar(32);not null"`
	Fingerprint     string     `gorm:"column:fingerprint;type:varchar(128);not null;index:idx_ai_alert_ingest_source_fp,priority:2"`
	Status          string     `gorm:"column:status;type:varchar(16);not null;index:idx_ai_alert_ingest_status"`
	DedupeKey       string     `gorm:"column:dedupe_key;type:varchar(256);not null;uniqueIndex:uk_ai_alert_ingest_dedupe"`
	Severity        string     `gorm:"column:severity;type:varchar(16);not null;default:'warning'"`
	Title           string     `gorm:"column:title;type:varchar(255);not null"`
	Target          string     `gorm:"column:target;type:varchar(255);not null;default:''"`
	LabelsJSON      string     `gorm:"column:labels_json;type:text;not null;default:'{}'"`
	AnnotationsJSON string     `gorm:"column:annotations_json;type:text;not null;default:'{}'"`
	RawPayloadJSON  string     `gorm:"column:raw_payload_json;type:text;not null"`
	StartsAt        *time.Time `gorm:"column:starts_at"`
	EndsAt          *time.Time `gorm:"column:ends_at"`
	ReceivedAt      time.Time  `gorm:"column:received_at;not null;index:idx_ai_alert_ingest_received"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (AIAlertIngestEvent) TableName() string { return "ai_alert_ingest_events" }

type AIAlertHealJob struct {
	ID          string     `gorm:"column:id;type:varchar(64);primaryKey"`
	EventID     string     `gorm:"column:event_id;type:varchar(64);not null;index:idx_ai_alert_heal_jobs_event_id"`
	Scene       string     `gorm:"column:scene;type:varchar(32);not null"`
	Status      string     `gorm:"column:status;type:varchar(32);not null;index:idx_ai_alert_heal_jobs_queue,priority:1"`
	Decision    string     `gorm:"column:decision;type:varchar(32);not null;default:''"`
	RetryCount  int        `gorm:"column:retry_count;not null;default:0"`
	MaxRetry    int        `gorm:"column:max_retry;not null;default:3"`
	NextRetryAt *time.Time `gorm:"column:next_retry_at;index:idx_ai_alert_heal_jobs_queue,priority:2"`
	LastError   string     `gorm:"column:last_error;type:text;not null;default:''"`
	LatestRunID string     `gorm:"column:latest_run_id;type:varchar(64);not null;default:''"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime;index:idx_ai_alert_heal_jobs_queue,priority:3"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (AIAlertHealJob) TableName() string { return "ai_alert_heal_jobs" }

type AIAlertHealAttempt struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	JobID        string    `gorm:"column:job_id;type:varchar(64);not null;index:idx_ai_alert_heal_attempt_job,priority:1;uniqueIndex:uk_ai_alert_heal_attempt_job_no,priority:1"`
	AttemptNo    int       `gorm:"column:attempt_no;not null;index:idx_ai_alert_heal_attempt_job,priority:2;uniqueIndex:uk_ai_alert_heal_attempt_job_no,priority:2"`
	RunID        string    `gorm:"column:run_id;type:varchar(64);not null;default:''"`
	Outcome      string    `gorm:"column:outcome;type:varchar(32);not null"`
	ErrorMessage string    `gorm:"column:error_message;type:text;not null;default:''"`
	StartedAt    time.Time `gorm:"column:started_at;not null"`
	FinishedAt   time.Time `gorm:"column:finished_at;not null"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (AIAlertHealAttempt) TableName() string { return "ai_alert_heal_attempts" }
