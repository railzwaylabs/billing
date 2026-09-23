# IAM policies

Billing IAM authorizes authenticated principals against organization-scoped
resources. PostgreSQL is the source of truth, while an in-memory Casbin
evaluator serves authorization checks. The model is default-deny: a request is
allowed only when a matching binding grants the exact permission on the target
resource or one of its parents.

## Concepts

An authorization decision combines four values:

```text
principal + organization + resource + permission -> allow or deny
```

### Principals

Supported principal types are:

| Type | Meaning |
| --- | --- |
| `user` | A console or externally authenticated user |
| `service_account` | A non-human identity that can authenticate with an API key |

A principal is identified by all three fields: `type`, `issuer`, and
`subject`. The issuer and subject are not interchangeable across identity
providers. A subject from a different issuer is a different principal.

### Resources

Canonical resource names use one of these forms:

```text
organizations/{organization_slug}
organizations/{organization_slug}/{collection}/{resource_id}
```

Supported child collections are `meters`, `usageEvents`, `products`, `prices`,
`customers`, `subscriptions`, `invoices`, `roles`, `serviceAccounts`, and
`apiKeys`.

For example:

```text
organizations/acme
organizations/acme/products/01JABC...
organizations/acme/serviceAccounts/9f0f...
```

A binding on `organizations/acme` applies to that resource and all descendants
whose names begin with `organizations/acme/`. Prefix collisions do not match;
access to `organizations/acme` does not grant access to
`organizations/acme-other`.

### Permissions

Permission names follow `service.resourceType.action`, for example:

```text
billing.products.get
billing.usageEvents.create
billing.organizations.setIamPolicy
billing.apiKeys.revoke
```

The database migration is the authoritative permission catalogue. Current
resource actions are:

| Resource type | Actions |
| --- | --- |
| `organizations` | `get`, `update`, `getIamPolicy`, `setIamPolicy`, `testIamPermissions` |
| `meters`, `products`, `prices`, `customers`, `subscriptions`, `invoices` | `create`, `get`, `list`, `update`, `delete` |
| `usageEvents` | `create`, `get`, `list` |
| `roles` | `create`, `get`, `list`, `update`, `delete` |
| `serviceAccounts` | `create`, `get`, `list`, `update`, `disable` |
| `apiKeys` | `create`, `get`, `list`, `revoke` |
| `monitoring` | `get` |
| `logs` | `list` |

## Roles

Roles group permissions. Predefined roles are shared across organizations and
cannot be modified as custom roles.

| Role | Scope |
| --- | --- |
| `roles/owner` | All billing and IAM permissions |
| `roles/admin` | All billing and IAM permissions |
| `roles/billingAdmin` | Billing-resource administration plus read-only IAM visibility |
| `roles/iamAdmin` | Role, policy, service-account, and API-key administration |
| `roles/developer` | Service-account and API-key management plus service monitoring and logs |
| `roles/viewer` | Read-only access through `get`, `list`, policy-read, and permission-test actions |

The Console evaluates the active principal's permissions when rendering the
Developer navigation. Service accounts require
`billing.serviceAccounts.list`, API keys additionally require
`billing.apiKeys.list`, Monitor requires `billing.monitoring.get`, and Logs
requires `billing.logs.list`. Hiding navigation is only a UX concern: the
monitoring and log endpoints independently enforce their permission in the
admin API.

Custom roles belong to one organization. Their canonical names are generated
as:

```text
organizations/{organization_slug}/roles/{role_name}
```

Only permissions registered for the `billing` service can be assigned to a
custom role. Custom-role updates and deletes require the role's current ETag.

## Policies and bindings

A policy belongs to one resource and contains bindings. Each binding assigns
one role to one principal:

```json
{
  "role": "roles/viewer",
  "principal": {
    "type": "user",
    "issuer": "billing-console",
    "subject": "user-123"
  }
}
```

The organization creator receives `roles/owner` on the organization resource.
An organization must retain at least one owner. Attempts to remove the final
owner fail with `IAM_LAST_OWNER_REQUIRED`.

Policy writes replace the complete binding list for the selected resource;
they are not incremental patches. Clients should always read the latest policy,
modify the returned binding set, and send the returned ETag with the update.
Duplicate principal-and-role bindings are rejected.

## Policy API

IAM administration routes are served below `/admin/v1/iam` and use the
authenticated console session. The examples assume a valid session cookie in
`cookies.txt`.

### Read a policy

```bash
curl --cookie cookies.txt \
  'http://localhost:8080/admin/v1/iam/policy?resource=organizations/acme'
```

Example response:

```json
{
  "resource": "organizations/acme",
  "version": 3,
  "etag": "3",
  "bindings": [
    {
      "role": "roles/owner",
      "principal": {
        "type": "user",
        "issuer": "billing-console",
        "subject": "user-123"
      }
    }
  ]
}
```

### Replace a policy

```bash
curl --request PUT \
  --cookie cookies.txt \
  --header 'Content-Type: application/json' \
  --header 'X-Request-ID: policy-update-001' \
  --data '{
    "etag": "3",
    "bindings": [
      {
        "role": "roles/owner",
        "principal": {
          "type": "user",
          "issuer": "billing-console",
          "subject": "user-123"
        }
      },
      {
        "role": "roles/viewer",
        "principal": {
          "type": "service_account",
          "issuer": "billing",
          "subject": "reports"
        }
      }
    ]
  }' \
  'http://localhost:8080/admin/v1/iam/policy?resource=organizations/acme'
```

If another writer has changed the policy, the stale ETag is rejected with HTTP
`409` and code `IAM_POLICY_CONFLICT`. Fetch the current policy and reapply the
intended change instead of retrying the stale document unchanged.

### Test permissions

The endpoint returns only permissions currently allowed for the authenticated
principal on the requested resource:

```bash
curl --request POST \
  --cookie cookies.txt \
  --header 'Content-Type: application/json' \
  --data '{
    "resource": "organizations/acme/products/product-123",
    "permissions": [
      "billing.products.get",
      "billing.products.update"
    ]
  }' \
  http://localhost:8080/admin/v1/iam:testPermissions
```

## Service accounts and API keys

Service accounts are organization resources intended for machine access. A
service account receives permissions only through policy bindings, just like a
user. Disabling the account invalidates its API-key credentials.

API keys:

- Use the `sk_live_` format.
- Are returned in plaintext only when created.
- Are stored as hashes rather than plaintext.
- May have an RFC 3339 expiration timestamp.
- Can be revoked without deleting the service account.
- Authenticate as `Authorization: Bearer {api_key}` on API-key-protected routes.

Treat generated keys as secrets and store them in a secret manager. Listing
keys never returns the original plaintext value.

## Evaluation and synchronization

Policy and role mutations are committed to PostgreSQL and recorded with the
actor, resource, before/after state, and optional `X-Request-ID`. Bindings and
role permissions are compiled into Casbin allow rules.

After a policy write, the local evaluator reloads immediately. PostgreSQL also
publishes `billing_iam_policy_changed`; other processes listen for that event
and compare stored policy versions. A periodic 30-second version check recovers
from missed notifications. Reload builds a complete replacement evaluator and
swaps it atomically. If reload fails, the process continues serving the previous
known-good snapshot and logs the error.

## Errors and safe client behavior

| Code | Meaning | Client behavior |
| --- | --- | --- |
| `UNAUTHENTICATED` | No valid session, token, or API key | Authenticate again |
| `PERMISSION_DENIED` | Principal lacks the requested permission | Do not retry without a policy change |
| `IAM_RESOURCE_INVALID` | Canonical resource name is malformed or unsupported | Correct the resource name |
| `IAM_PERMISSION_INVALID` | Permission does not exist or has invalid syntax | Use a registered billing permission |
| `IAM_ROLE_NOT_FOUND` | Referenced role is unavailable to the organization | Refresh the role list |
| `IAM_POLICY_CONFLICT` | Policy or role ETag is stale | Read current state, merge, and retry |
| `IAM_LAST_OWNER_REQUIRED` | Update would remove the final owner | Preserve or add an owner binding |
| `API_KEY_INVALID` | Key is malformed, expired, revoked, disabled, or incorrect | Replace the credential |

## Current limitations

- Policies contain allow bindings only; explicit deny is not implemented.
- Conditional bindings are not implemented.
- Policy updates replace all bindings on one resource; there is no binding-level
  patch endpoint.
- Predefined roles are migration-defined rather than dynamically configurable.
