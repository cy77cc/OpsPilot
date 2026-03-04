-- +migrate Up
SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'services' AND COLUMN_NAME = 'visibility');
SET @sql := IF(@col_exists = 0, 'ALTER TABLE services ADD COLUMN visibility VARCHAR(16) NOT NULL DEFAULT ''team''', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'services' AND COLUMN_NAME = 'granted_teams');
SET @sql := IF(@col_exists = 0, 'ALTER TABLE services ADD COLUMN granted_teams JSON', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'services' AND COLUMN_NAME = 'icon');
SET @sql := IF(@col_exists = 0, 'ALTER TABLE services ADD COLUMN icon VARCHAR(256) DEFAULT ''''', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'services' AND COLUMN_NAME = 'tags');
SET @sql := IF(@col_exists = 0, 'ALTER TABLE services ADD COLUMN tags JSON', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'services' AND COLUMN_NAME = 'deploy_count');
SET @sql := IF(@col_exists = 0, 'ALTER TABLE services ADD COLUMN deploy_count INT NOT NULL DEFAULT 0', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE services
SET service_kind = CASE
  WHEN service_kind IS NULL OR service_kind = '' OR service_kind = 'web' THEN 'business'
  ELSE service_kind
END;

UPDATE services
SET visibility = CASE
  WHEN visibility IS NULL OR visibility = '' THEN
    CASE WHEN service_kind = 'middleware' THEN 'public' ELSE 'team' END
  ELSE visibility
END;

-- migrate legacy catalog templates if table exists
SET @catalog_table_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'service_templates');
SET @sql := IF(@catalog_table_exists > 0,
'INSERT INTO services
  (project_id, team_id, owner_user_id, owner, env, runtime_type, config_mode, service_kind, visibility, render_target, custom_yaml, source_template_version, template_engine_version, status, name, type, image, replicas, service_port, container_port, yaml_content, icon, tags, deploy_count, created_at, updated_at)
 SELECT
  0,
  0,
  COALESCE(owner_id, 0),
  ''catalog'',
  ''staging'',
  CASE WHEN compose_template IS NOT NULL AND compose_template <> '''' THEN ''compose'' ELSE ''k8s'' END,
  ''custom'',
  ''middleware'',
  CASE WHEN visibility = ''public'' OR status = ''published'' THEN ''public'' ELSE ''team'' END,
  CASE WHEN compose_template IS NOT NULL AND compose_template <> '''' THEN ''compose'' ELSE ''k8s'' END,
  COALESCE(NULLIF(k8s_template, ''''), compose_template, ''''),
  COALESCE(version, ''1.0.0''),
  ''v1'',
  ''draft'',
  name,
  ''stateless'',
  ''catalog/template'',
  1,
  80,
  80,
  COALESCE(NULLIF(k8s_template, ''''), compose_template, ''''),
  COALESCE(icon, ''''),
  COALESCE(tags, JSON_ARRAY()),
  COALESCE(deploy_count, 0),
  created_at,
  updated_at
 FROM service_templates st
 WHERE NOT EXISTS (SELECT 1 FROM services s WHERE s.name = st.name)',
'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql := IF(@catalog_table_exists > 0, 'DROP TABLE IF EXISTS service_templates', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @cat_table_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'service_categories');
SET @sql := IF(@cat_table_exists > 0, 'DROP TABLE IF EXISTS service_categories', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +migrate Down
SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'services' AND COLUMN_NAME = 'deploy_count');
SET @sql := IF(@col_exists > 0, 'ALTER TABLE services DROP COLUMN deploy_count', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'services' AND COLUMN_NAME = 'tags');
SET @sql := IF(@col_exists > 0, 'ALTER TABLE services DROP COLUMN tags', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'services' AND COLUMN_NAME = 'icon');
SET @sql := IF(@col_exists > 0, 'ALTER TABLE services DROP COLUMN icon', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'services' AND COLUMN_NAME = 'granted_teams');
SET @sql := IF(@col_exists > 0, 'ALTER TABLE services DROP COLUMN granted_teams', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists := (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'services' AND COLUMN_NAME = 'visibility');
SET @sql := IF(@col_exists > 0, 'ALTER TABLE services DROP COLUMN visibility', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
