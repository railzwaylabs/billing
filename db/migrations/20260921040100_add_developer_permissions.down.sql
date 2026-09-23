DELETE FROM iam_policy_bindings
WHERE role_id IN (SELECT id FROM iam_roles WHERE name = 'roles/developer');

DELETE FROM iam_role_permissions
WHERE permission_name IN ('billing.monitoring.get', 'billing.logs.list')
   OR role_id IN (SELECT id FROM iam_roles WHERE name = 'roles/developer');

DELETE FROM iam_roles WHERE name = 'roles/developer';
DELETE FROM iam_permissions
WHERE name IN ('billing.monitoring.get', 'billing.logs.list');
DELETE FROM iam_resource_types WHERE code IN ('monitoring', 'logs');
