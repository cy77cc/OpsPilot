-- Detect services likely impacted by historical hard-coded team_id usage
-- and missing default deploy target binding.
SELECT
  s.id AS service_id,
  s.name AS service_name,
  s.project_id,
  s.team_id,
  s.env,
  s.runtime_type,
  CASE WHEN t.service_id IS NULL THEN 1 ELSE 0 END AS missing_service_default_target
FROM services s
LEFT JOIN service_deploy_targets t
  ON t.service_id = s.id AND t.is_default = 1
WHERE
  s.team_id = 1
  OR t.service_id IS NULL
ORDER BY s.updated_at DESC;

-- Detect scope-mismatch records where no active deployment target exists for service scope.
SELECT
  s.id AS service_id,
  s.name AS service_name,
  s.project_id,
  s.team_id,
  s.env,
  COALESCE(s.render_target, s.runtime_type, 'k8s') AS target_type
FROM services s
WHERE NOT EXISTS (
  SELECT 1
  FROM deployment_targets d
  WHERE d.status = 'active'
    AND d.target_type = COALESCE(NULLIF(s.render_target, ''), NULLIF(s.runtime_type, ''), 'k8s')
    AND (d.project_id = s.project_id OR s.project_id = 0)
    AND (d.team_id = s.team_id OR s.team_id = 0)
    AND (d.env = s.env OR d.env = '')
);
