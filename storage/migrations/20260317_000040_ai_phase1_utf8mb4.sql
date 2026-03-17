-- +migrate Up
ALTER TABLE ai_runs
  CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

ALTER TABLE ai_diagnosis_reports
  CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- +migrate Down
ALTER TABLE ai_diagnosis_reports
  CONVERT TO CHARACTER SET utf8 COLLATE utf8_general_ci;

ALTER TABLE ai_runs
  CONVERT TO CHARACTER SET utf8 COLLATE utf8_general_ci;
