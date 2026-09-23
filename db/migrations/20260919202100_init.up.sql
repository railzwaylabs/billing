CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(100),
    email VARCHAR(320),
    display_name VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    password_change_required BOOLEAN NOT NULL DEFAULT FALSE,
    password_prompted_at TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT users_username_format CHECK (username IS NULL OR username ~ '^[a-z][a-z0-9._-]{2,99}$'),
    CONSTRAINT users_status CHECK (status IN ('active', 'disabled'))
);

CREATE UNIQUE INDEX idx_users_username_lower ON users (LOWER(username)) WHERE username IS NOT NULL;
CREATE UNIQUE INDEX idx_users_email_lower ON users (LOWER(email)) WHERE email IS NOT NULL;

CREATE TABLE user_password_credentials (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    password_hash TEXT NOT NULL,
    password_changed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT user_password_credentials_hash_not_empty CHECK (LENGTH(password_hash) > 0)
);

CREATE TABLE user_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(100) NOT NULL,
    issuer TEXT NOT NULL,
    external_subject TEXT NOT NULL,
    email TEXT,
    profile JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT user_identities_provider_not_empty CHECK (LENGTH(TRIM(provider)) > 0),
    CONSTRAINT user_identities_issuer_not_empty CHECK (LENGTH(TRIM(issuer)) > 0),
    CONSTRAINT user_identities_subject_not_empty CHECK (LENGTH(TRIM(external_subject)) > 0),
    UNIQUE (issuer, external_subject),
    UNIQUE (user_id, provider)
);

CREATE INDEX idx_user_identities_user ON user_identities (user_id);

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMPTZ,
    user_agent TEXT NOT NULL DEFAULT '',
    ip_address INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT sessions_token_hash_length CHECK (octet_length(token_hash) = 32),
    CONSTRAINT sessions_expiry CHECK (expires_at > created_at),
    UNIQUE (token_hash)
);

CREATE INDEX idx_sessions_user_active
    ON sessions (user_id, expires_at DESC) WHERE revoked_at IS NULL;
CREATE INDEX idx_sessions_expiry
    ON sessions (expires_at) WHERE revoked_at IS NULL;

CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT organizations_slug_format CHECK (slug ~ '^[a-z][a-z0-9-]{0,62}$')
);

CREATE UNIQUE INDEX idx_organizations_slug ON organizations (slug);

CREATE TABLE iam_services (
    code TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT iam_services_code_format CHECK (code ~ '^[a-z][a-z0-9]*$')
);

CREATE TABLE iam_resource_types (
    code TEXT PRIMARY KEY,
    service_code TEXT NOT NULL REFERENCES iam_services(code) ON DELETE RESTRICT,
    collection TEXT NOT NULL,
    parent_code TEXT REFERENCES iam_resource_types(code) ON DELETE RESTRICT,
    display_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT iam_resource_types_code_format CHECK (code ~ '^[a-z][a-zA-Z0-9]*$'),
    CONSTRAINT iam_resource_types_collection_format CHECK (collection ~ '^[a-z][a-zA-Z0-9]*$'),
    UNIQUE (service_code, collection)
);

CREATE TABLE iam_permissions (
    name TEXT PRIMARY KEY,
    service_code TEXT NOT NULL REFERENCES iam_services(code) ON DELETE RESTRICT,
    resource_type_code TEXT NOT NULL REFERENCES iam_resource_types(code) ON DELETE RESTRICT,
    action TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT iam_permissions_name_format
        CHECK (name ~ '^[a-z][a-z0-9]*\.[a-z][a-zA-Z0-9]*\.[a-z][a-zA-Z0-9]*$'),
    CONSTRAINT iam_permissions_action_format CHECK (action ~ '^[a-z][a-zA-Z0-9]*$'),
    UNIQUE (service_code, resource_type_code, action)
);

CREATE INDEX idx_iam_permissions_resource_type
    ON iam_permissions (resource_type_code, action);

CREATE TABLE iam_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    predefined BOOLEAN NOT NULL DEFAULT FALSE,
    etag UUID NOT NULL DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT iam_roles_scope CHECK (
        (predefined AND organization_id IS NULL AND name ~ '^roles/[a-z][a-zA-Z0-9]*$')
        OR
        (NOT predefined AND organization_id IS NOT NULL AND name ~ '^organizations/[a-z][a-z0-9-]{0,62}/roles/[a-z][a-zA-Z0-9-]{0,62}$')
    )
);

CREATE UNIQUE INDEX idx_iam_roles_predefined_name
    ON iam_roles (name) WHERE predefined;
CREATE UNIQUE INDEX idx_iam_roles_organization_name
    ON iam_roles (organization_id, name) WHERE NOT predefined;

CREATE TABLE iam_role_permissions (
    role_id UUID NOT NULL REFERENCES iam_roles(id) ON DELETE CASCADE,
    permission_name TEXT NOT NULL REFERENCES iam_permissions(name) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, permission_name)
);

CREATE INDEX idx_iam_role_permissions_permission
    ON iam_role_permissions (permission_name, role_id);

CREATE TABLE iam_service_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    resource_name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    issuer TEXT NOT NULL,
    subject TEXT NOT NULL,
    disabled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT iam_service_accounts_resource_name_format
        CHECK (resource_name ~ '^organizations/[a-z][a-z0-9-]{0,62}/serviceAccounts/[0-9a-f-]{36}$'),
    CONSTRAINT iam_service_accounts_issuer_not_empty CHECK (LENGTH(TRIM(issuer)) > 0),
    CONSTRAINT iam_service_accounts_subject_not_empty CHECK (LENGTH(TRIM(subject)) > 0),
    UNIQUE (resource_name),
    UNIQUE (issuer, subject),
    UNIQUE (id, organization_id)
);

CREATE INDEX idx_iam_service_accounts_organization
    ON iam_service_accounts (organization_id, created_at DESC);

CREATE TABLE iam_api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    service_account_id UUID NOT NULL,
    key_id VARCHAR(32) NOT NULL,
    secret_hash BYTEA NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT iam_api_keys_key_id_format CHECK (key_id ~ '^[a-f0-9]{12}$'),
    CONSTRAINT iam_api_keys_secret_hash_length CHECK (octet_length(secret_hash) = 32),
    CONSTRAINT iam_api_keys_display_name_not_empty CHECK (LENGTH(TRIM(display_name)) > 0),
    CONSTRAINT iam_api_keys_expiry CHECK (expires_at IS NULL OR expires_at > created_at),
    UNIQUE (key_id),
    UNIQUE (id, organization_id),
    FOREIGN KEY (service_account_id, organization_id)
        REFERENCES iam_service_accounts(id, organization_id) ON DELETE CASCADE
);

CREATE INDEX idx_iam_api_keys_service_account
    ON iam_api_keys (organization_id, service_account_id, created_at DESC);
CREATE INDEX idx_iam_api_keys_active_expiry
    ON iam_api_keys (expires_at) WHERE revoked_at IS NULL;

CREATE TABLE iam_policy_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    resource_name TEXT NOT NULL,
    principal_type TEXT NOT NULL,
    principal_issuer TEXT NOT NULL,
    principal_subject TEXT NOT NULL,
    role_id UUID NOT NULL REFERENCES iam_roles(id) ON DELETE RESTRICT,
    created_by_issuer TEXT NOT NULL,
    created_by_subject TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT iam_policy_bindings_principal_type
        CHECK (principal_type IN ('user', 'service_account')),
    CONSTRAINT iam_policy_bindings_resource_name
        CHECK (resource_name ~ '^organizations/[a-z][a-z0-9-]{0,62}(/([a-z][a-zA-Z0-9]*)/[^/]+)*$'),
    CONSTRAINT iam_policy_bindings_principal_issuer_not_empty
        CHECK (LENGTH(TRIM(principal_issuer)) > 0),
    CONSTRAINT iam_policy_bindings_principal_subject_not_empty
        CHECK (LENGTH(TRIM(principal_subject)) > 0),
    UNIQUE (
        organization_id,
        resource_name,
        principal_type,
        principal_issuer,
        principal_subject,
        role_id
    )
);

CREATE INDEX idx_iam_policy_bindings_principal
    ON iam_policy_bindings (
        organization_id,
        principal_type,
        principal_issuer,
        principal_subject
    );
CREATE INDEX idx_iam_policy_bindings_resource
    ON iam_policy_bindings (organization_id, resource_name);
CREATE INDEX idx_iam_policy_bindings_role ON iam_policy_bindings (role_id);

CREATE TABLE iam_policy_versions (
    organization_id UUID PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    version BIGINT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT iam_policy_versions_positive CHECK (version > 0)
);

CREATE TABLE iam_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    actor_type TEXT NOT NULL,
    actor_issuer TEXT NOT NULL,
    actor_subject TEXT NOT NULL,
    action TEXT NOT NULL,
    resource_name TEXT NOT NULL,
    before_state JSONB,
    after_state JSONB,
    request_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT iam_audit_logs_actor_type CHECK (actor_type IN ('user', 'service_account'))
);

CREATE INDEX idx_iam_audit_logs_organization_time
    ON iam_audit_logs (organization_id, created_at DESC);
CREATE INDEX idx_iam_audit_logs_actor
    ON iam_audit_logs (actor_issuer, actor_subject, created_at DESC);
CREATE INDEX idx_iam_audit_logs_resource
    ON iam_audit_logs (resource_name, created_at DESC);

INSERT INTO iam_services (code, display_name, description)
VALUES ('billing', 'Billing', 'Billing service')
ON CONFLICT (code) DO NOTHING;

INSERT INTO iam_resource_types (code, service_code, collection, parent_code, display_name)
VALUES
    ('organizations', 'billing', 'organizations', NULL, 'Organizations'),
    ('meters', 'billing', 'meters', 'organizations', 'Meters'),
    ('usageEvents', 'billing', 'usageEvents', 'organizations', 'Usage events'),
    ('apiKeys', 'billing', 'apiKeys', 'organizations', 'API keys'),
    ('products', 'billing', 'products', 'organizations', 'Products'),
    ('prices', 'billing', 'prices', 'organizations', 'Prices'),
    ('customers', 'billing', 'customers', 'organizations', 'Customers'),
    ('subscriptions', 'billing', 'subscriptions', 'organizations', 'Subscriptions'),
    ('invoices', 'billing', 'invoices', 'organizations', 'Invoices'),
    ('roles', 'billing', 'roles', 'organizations', 'IAM roles'),
    ('serviceAccounts', 'billing', 'serviceAccounts', 'organizations', 'Service accounts')
ON CONFLICT (code) DO NOTHING;

INSERT INTO iam_permissions (name, service_code, resource_type_code, action, display_name)
SELECT
    'billing.' || resource_type || '.' || action,
    'billing',
    resource_type,
    action,
    resource_type || ' ' || action
FROM (VALUES
    ('organizations', 'get'),
    ('organizations', 'update'),
    ('organizations', 'getIamPolicy'),
    ('organizations', 'setIamPolicy'),
    ('organizations', 'testIamPermissions'),
    ('meters', 'create'), ('meters', 'get'), ('meters', 'list'), ('meters', 'update'), ('meters', 'delete'),
    ('usageEvents', 'create'), ('usageEvents', 'get'), ('usageEvents', 'list'),
    ('apiKeys', 'create'), ('apiKeys', 'get'), ('apiKeys', 'list'), ('apiKeys', 'revoke'),
    ('products', 'create'), ('products', 'get'), ('products', 'list'), ('products', 'update'), ('products', 'delete'),
    ('prices', 'create'), ('prices', 'get'), ('prices', 'list'), ('prices', 'update'), ('prices', 'delete'),
    ('customers', 'create'), ('customers', 'get'), ('customers', 'list'), ('customers', 'update'), ('customers', 'delete'),
    ('subscriptions', 'create'), ('subscriptions', 'get'), ('subscriptions', 'list'), ('subscriptions', 'update'), ('subscriptions', 'delete'),
    ('invoices', 'create'), ('invoices', 'get'), ('invoices', 'list'), ('invoices', 'update'), ('invoices', 'delete'),
    ('roles', 'create'), ('roles', 'get'), ('roles', 'list'), ('roles', 'update'), ('roles', 'delete'),
    ('serviceAccounts', 'create'), ('serviceAccounts', 'get'), ('serviceAccounts', 'list'), ('serviceAccounts', 'update'), ('serviceAccounts', 'disable')
) AS permissions(resource_type, action)
ON CONFLICT (name) DO NOTHING;

INSERT INTO iam_roles (name, display_name, description, predefined)
VALUES
    ('roles/owner', 'Owner', 'Full billing and IAM access', TRUE),
    ('roles/admin', 'Administrator', 'Billing and IAM administration', TRUE),
    ('roles/billingAdmin', 'Billing administrator', 'Manage billing resources and view IAM', TRUE),
    ('roles/iamAdmin', 'IAM administrator', 'Manage roles, policies, and service accounts', TRUE),
    ('roles/viewer', 'Viewer', 'Read-only billing access', TRUE)
ON CONFLICT DO NOTHING;

INSERT INTO iam_role_permissions (role_id, permission_name)
SELECT role.id, permission.name
FROM iam_roles AS role
CROSS JOIN iam_permissions AS permission
WHERE role.name IN ('roles/owner', 'roles/admin')
ON CONFLICT DO NOTHING;

INSERT INTO iam_role_permissions (role_id, permission_name)
SELECT role.id, permission.name
FROM iam_roles AS role
JOIN iam_permissions AS permission ON (
    permission.resource_type_code IN ('meters', 'usageEvents', 'products', 'prices', 'customers', 'subscriptions', 'invoices')
    OR permission.name IN (
        'billing.organizations.get',
        'billing.organizations.getIamPolicy',
        'billing.organizations.testIamPermissions',
        'billing.roles.get',
        'billing.roles.list',
        'billing.serviceAccounts.get',
        'billing.serviceAccounts.list'
    )
)
WHERE role.name = 'roles/billingAdmin'
ON CONFLICT DO NOTHING;

INSERT INTO iam_role_permissions (role_id, permission_name)
SELECT role.id, permission.name
FROM iam_roles AS role
JOIN iam_permissions AS permission ON (
    permission.resource_type_code IN ('roles', 'serviceAccounts', 'apiKeys')
    OR permission.name IN (
        'billing.organizations.get',
        'billing.organizations.getIamPolicy',
        'billing.organizations.setIamPolicy',
        'billing.organizations.testIamPermissions'
    )
)
WHERE role.name = 'roles/iamAdmin'
ON CONFLICT DO NOTHING;

INSERT INTO iam_role_permissions (role_id, permission_name)
SELECT role.id, permission.name
FROM iam_roles AS role
JOIN iam_permissions AS permission
    ON permission.action IN ('get', 'list', 'getIamPolicy', 'testIamPermissions')
WHERE role.name = 'roles/viewer'
ON CONFLICT DO NOTHING;
