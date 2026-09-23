INSERT INTO iam_resource_types (code, service_code, collection, parent_code, display_name)
VALUES
    ('monitoring', 'billing', 'monitoring', 'organizations', 'Service monitoring'),
    ('logs', 'billing', 'logs', 'organizations', 'Service logs')
ON CONFLICT (code) DO NOTHING;

INSERT INTO iam_permissions (
    name,
    service_code,
    resource_type_code,
    action,
    display_name,
    description
)
VALUES
    (
        'billing.monitoring.get',
        'billing',
        'monitoring',
        'get',
        'View service monitoring',
        'View service health and resource utilization in the Developer console.'
    ),
    (
        'billing.logs.list',
        'billing',
        'logs',
        'list',
        'View service logs',
        'Search and inspect service logs in the Developer console.'
    )
ON CONFLICT (name) DO NOTHING;

INSERT INTO iam_roles (name, display_name, description, predefined)
VALUES (
    'roles/developer',
    'Developer',
    'Manage developer integrations and inspect service operations',
    TRUE
)
ON CONFLICT DO NOTHING;

INSERT INTO iam_role_permissions (role_id, permission_name)
SELECT role.id, permission.name
FROM iam_roles AS role
JOIN iam_permissions AS permission
    ON permission.name IN ('billing.monitoring.get', 'billing.logs.list')
WHERE role.name IN ('roles/owner', 'roles/admin')
ON CONFLICT DO NOTHING;

INSERT INTO iam_role_permissions (role_id, permission_name)
SELECT role.id, permission.name
FROM iam_roles AS role
JOIN iam_permissions AS permission ON permission.name IN (
    'billing.organizations.get',
    'billing.organizations.testIamPermissions',
    'billing.serviceAccounts.create',
    'billing.serviceAccounts.get',
    'billing.serviceAccounts.list',
    'billing.serviceAccounts.update',
    'billing.serviceAccounts.disable',
    'billing.apiKeys.create',
    'billing.apiKeys.get',
    'billing.apiKeys.list',
    'billing.apiKeys.revoke',
    'billing.monitoring.get',
    'billing.logs.list'
)
WHERE role.name = 'roles/developer'
ON CONFLICT DO NOTHING;
