ALTER TABLE ai_run_contents CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

SELECT table_name
FROM information_schema.tables
WHERE table_name = 'ai_run_contents';
