import type { Page, Route } from "@playwright/test";

export const organization = { id: "2e3b8554-f14e-4568-84ce-523704e14a02", name: "Acme", slug: "acme", created_at: "2026-09-21T00:00:00Z", updated_at: "2026-09-21T00:00:00Z" };
export const base = `/organizations/${organization.id}`;

type Failure = { method: string; path: RegExp; status: number; code: string; message: string };
type MockOptions = { authenticated?: boolean; organizations?: typeof organization[]; failure?: Failure };

const json = (route: Route, body: unknown, status = 200) => route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });

export async function mockAPI(page: Page, options: MockOptions = {}) {
  const authenticated = options.authenticated ?? true;
  const organizations = options.organizations ?? [organization];
  await page.route("**/prometheus/api/v1/query**", (route) => {
    const timestamp = Math.floor(Date.now() / 1000);
    return json(route, {
      status: "success",
      data: {
        resultType: "matrix",
        result: [{
          values: Array.from({ length: 24 }, (_, index) => [
            timestamp - (23 - index) * 3600,
            String(1 + index / 10),
          ]),
        }],
      },
    });
  });
  await page.route("**/admin/v1/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    const method = request.method();
    const failure = options.failure;
    if (failure && failure.method === method && failure.path.test(path))
      return json(route, { error: { code: failure.code, message: failure.message } }, failure.status);

    if (path === "/admin/v1/auth/providers") return json(route, { local: { enabled: true }, providers: [] });
    if (path === "/admin/v1/auth/session") return authenticated ? json(route, { authenticated: true, user: { id: "user-1", username: "admin" }, show_password_change: false }) : json(route, { error: { code: "UNAUTHENTICATED", message: "Authentication required" } }, 401);
    if (path === "/admin/v1/auth/login") return json(route, { user: { id: "user-1", username: "admin" }, show_password_change: false });
    if (path === "/admin/v1/auth/logout" || path.includes("/password/")) return route.fulfill({ status: 204 });
    if (path === "/admin/v1/organizations" && method === "GET") return json(route, { organizations });
    if (path === "/admin/v1/organizations" && method === "POST") return json(route, { organization });
    if (path.endsWith("/usage-events/summary") && method === "GET") return json(route, {
      from: "2025-10-01T00:00:00Z", to: "2026-10-01T00:00:00Z", interval: "month",
      points: Array.from({ length: 12 }, (_, index) => ({ bucket: new Date(Date.UTC(2025, 9 + index, 1)).toISOString().slice(0, 10), event_count: 0, customer_count: 0, meter_count: 0, value_micros: 0 })),
    });
    if (path.endsWith("/invoice-runs") && method === "POST") return json(route, { invoices: [], created: 0, existing: 0 });

    const emptyLists: Array<[RegExp, unknown]> = [
      [/\/meters$/, { meters: [] }], [/\/customers$/, { customers: [] }], [/\/products$/, { products: [] }], [/\/prices$/, { prices: [] }],
      [/\/subscriptions$/, { subscriptions: [] }], [/\/usage-events$/, { events: [] }], [/\/invoices$/, { invoices: [] }],
      [/\/iam\/serviceAccounts$/, { service_accounts: [] }], [/\/iam\/roles$/, { roles: [{ id: "owner", name: "roles/owner", display_name: "Owner", description: "Owner", predefined: true, etag: "1", permissions: ["billing.organizations.get"] }] }],
    ];
    if (method === "GET") {
      for (const [pattern, body] of emptyLists) if (pattern.test(path)) return json(route, body);
      if (path === "/admin/v1/iam/policy") return json(route, { resource: "organizations/acme", version: 1, etag: "1", bindings: [] });
    }
    return json(route, {});
  });
}

export async function openAuthenticated(page: Page, path = base) {
  await mockAPI(page);
  await page.goto(path);
}
