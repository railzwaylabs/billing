export type ApiError = {
  error: { code: string; message: string; details?: unknown[] };
};
export type PageParams = { limit?: number; cursor?: string };
export type PageInfo = { next_cursor?: string; has_more: boolean };
export type MonitoringRange = "day" | "week" | "month";
export type MonitoringSample = { timestamp: number; value: number };
export type ResourceMetrics = {
  service: string;
  range: MonitoringRange;
  step_seconds: number;
  cpu: { used: MonitoringSample[]; allocated: MonitoringSample[] };
  memory: { used: MonitoringSample[]; allocated: MonitoringSample[] };
  disk: { used: MonitoringSample[]; allocated: MonitoringSample[] };
  network: { receive: MonitoringSample[]; transmit: MonitoringSample[] };
};
export type ServiceHealthStatus =
  "healthy" | "degraded" | "unhealthy" | "unknown";
export type MonitoredService = {
  id: string;
  name: string;
  description: string;
  status: ServiceHealthStatus;
  cpu_usage?: number;
  memory_usage_bytes?: number;
  updated_at?: string;
};
export type LogService = {
  id: string;
  name: string;
  description: string;
};
export type LogEntry = {
  timestamp: string;
  service: string;
  level?: string;
  message: string;
  fields?: Record<string, unknown>;
};
export type LogPage = {
  entries: LogEntry[];
  next_cursor?: string;
};
const backendURL = (import.meta.env.VITE_BACKEND_URL || "").replace(/\/$/, "");
export const backendPath = (path: string) => `${backendURL}${path}`;
const pageURL = (path: string, page?: PageParams) => {
  const query = new URLSearchParams();
  if (page?.limit) query.set("limit", String(page.limit));
  if (page?.cursor) query.set("cursor", page.cursor);
  const encoded = query.toString();
  return encoded ? `${path}${path.includes("?") ? "&" : "?"}${encoded}` : path;
};
export type Organization = {
  id: string;
  name: string;
  slug: string;
  created_at: string;
  updated_at: string;
};
export type Meter = {
  ID: string;
  OrganizationID: string;
  Code: string;
  Name: string;
  Aggregation: "count" | "sum";
  Unit: string;
  CreatedAt: string;
  UpdatedAt: string;
};
export type Customer = {
  ID: string;
  OrganizationID: string;
  FirstName: string;
  LastName: string;
  CreatedAt: string;
  UpdatedAt: string;
};
export type Product = {
  ID: string;
  OrganizationID: string;
  Code: string;
  Name: string;
  Description: string;
  Status: string;
  CreatedAt: string;
  UpdatedAt: string;
};
export type ChargeTier = {
  ID: string;
  StartQuantity: { Micros: number };
  UnitAmount: { Currency: string; Nanos: number };
};
export type Price = {
  ID: string;
  OrganizationID: string;
  ProductID: string;
  Currency: string;
  BillingInterval: "day" | "week" | "month" | "year";
  IntervalCount: number;
  EffectiveAt: string;
  EffectiveUntil?: string;
  Status: string;
  Charges: PriceCharge[];
  CreatedAt: string;
  UpdatedAt: string;
};
export type PriceCharge = {
  ID: string;
  MeterID: string;
  Code: string;
  Name: string;
  PricingModel: "per_unit" | "graduated";
  UnitQuantity: { Micros: number };
  Tiers: ChargeTier[];
};

export type Currency = {
  Code: string;
  Name: string;
  Symbol: string;
  MinorUnit: number;
};

export type MeasurementUnit = {
  Code: string;
  Name: string;
  Symbol: string;
  Category: string;
  Description: string;
};
export type SubscriptionItem = {
  ID: string;
  PriceID: string;
  StartAt: string;
  EndAt?: string;
};
export type Subscription = {
  ID: string;
  OrganizationID: string;
  CustomerID: string;
  Status: string;
  StartDate: string;
  EndDate?: string;
  Items: SubscriptionItem[];
};

export type UsageEvent = {
  ID: string;
  EventID: string;
  MeterID: string;
  CustomerID: string;
  Value: { Micros: number };
  EventTime: string;
  IngestedAt: string;
};

export type UsagePoint = {
  bucket: string;
  event_count: number;
  customer_count: number;
  meter_count: number;
  value_micros: number;
};

export type UsageSummary = {
  from: string;
  to: string;
  interval: "day" | "week" | "month";
  points: UsagePoint[];
};

export type InvoiceLine = {
  ID: string;
  SubscriptionID: string;
  SubscriptionItemID: string;
  ProductID: string;
  PriceID: string;
  PriceChargeID: string;
  MeterID: string;
  Description: string;
  UsageQuantity: { Micros: number };
  Unit: string;
  PricingUnitQuantity: { Micros: number };
  UnitAmount: { Currency: string; Nanos: number };
  Amount: { Currency: string; Nanos: number };
};

export type Invoice = {
  ID: string;
  InvoiceNumber: string;
  CustomerID: string;
  Status: string;
  BillingPeriodStart: string;
  BillingPeriodEnd: string;
  Tax: { Currency: string; Nanos: number };
  Total: { Currency: string; Nanos: number };
  Lines: InvoiceLine[];
};

export type InvoiceNumberSettings = {
  number_format: string;
};

export type ServiceAccount = {
  id: string;
  resource_name: string;
  display_name: string;
  description: string;
  issuer: string;
  subject: string;
  disabled: boolean;
};

export type APIKey = {
  id: string;
  service_account_id: string;
  key_id: string;
  display_name: string;
  expires_at?: string;
  revoked_at?: string;
  created_at: string;
  key?: string;
};

export type IAMRole = {
  id: string;
  name: string;
  display_name: string;
  description: string;
  predefined: boolean;
  etag: string;
  permissions: string[];
};

export type IAMBinding = {
  role: string;
  principal: {
    type: "user" | "service_account";
    issuer: string;
    subject: string;
  };
};

export type IAMPolicy = {
  resource: string;
  version: number;
  etag: string;
  bindings: IAMBinding[];
};

export type DirectoryUser = {
  id: string;
  username: string;
  email: string;
  display_name: string;
  disabled: boolean;
  created_at: string;
};

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(backendPath(path), {
    credentials: "include",
    headers: { "Content-Type": "application/json", ...init?.headers },
    ...init,
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({
      error: { code: "REQUEST_FAILED", message: "Request failed" },
    }));
    throw new Error((body as ApiError).error?.message ?? "Request failed");
  }
  if (response.status === 204) return undefined as T;
  return response.json();
}

export const api = {
  monitoredServices: (organization: string) =>
    request<{ services: MonitoredService[] }>(
      `/admin/v1/monitoring/services?organization=${encodeURIComponent(organization)}`,
    ),
  monitoredService: (organization: string, service: string) =>
    request<{ service: MonitoredService }>(
      `/admin/v1/monitoring/services/${encodeURIComponent(service)}?organization=${encodeURIComponent(organization)}`,
    ),
  monitoringResources: (
    organization: string,
    service: string,
    range: MonitoringRange,
  ) =>
    request<ResourceMetrics>(
      `/admin/v1/monitoring/services/${encodeURIComponent(service)}/resources?organization=${encodeURIComponent(organization)}&range=${range}`,
    ),
  logServices: (organization: string) =>
    request<{ services: LogService[] }>(
      `/admin/v1/logs/services?organization=${encodeURIComponent(organization)}`,
    ),
  logs: (query: {
    organization: string;
    service: string;
    level?: string;
    search?: string;
    from: string;
    to: string;
    limit?: number;
    cursor?: string;
  }) => {
    const parameters = new URLSearchParams({
      organization: query.organization,
      service: query.service,
      from: query.from,
      to: query.to,
      limit: String(query.limit ?? 100),
    });
    if (query.level) parameters.set("level", query.level);
    if (query.search) parameters.set("search", query.search);
    if (query.cursor) parameters.set("cursor", query.cursor);
    return request<LogPage>(`/admin/v1/logs/query?${parameters.toString()}`);
  },
  providers: () =>
    request<{
      local: { enabled: boolean };
      providers: {
        id: string;
        name: string;
        enabled: boolean;
        login_url: string;
      }[];
    }>("/admin/v1/auth/providers"),
  login: (username: string, password: string) =>
    request<{
      user: { id: string; username: string };
      show_password_change: boolean;
    }>("/admin/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    }),
  session: () =>
    request<{
      authenticated: boolean;
      user: { id: string; username: string };
      show_password_change: boolean;
    }>("/admin/v1/auth/session"),
  logout: () => request<void>("/admin/v1/auth/logout", { method: "POST" }),
  skipPassword: () =>
    request<void>("/admin/v1/auth/password/skip", { method: "POST" }),
  changePassword: (currentPassword: string, newPassword: string) =>
    request<void>("/admin/v1/auth/password", {
      method: "POST",
      body: JSON.stringify({
        current_password: currentPassword,
        new_password: newPassword,
      }),
    }),
  organizations: (page?: PageParams) =>
    request<{ organizations: Organization[]; page_info: PageInfo }>(
      pageURL("/admin/v1/organizations", page),
    ),
  createOrganization: (body: { name: string; slug: string }) =>
    request<{ organization: Organization }>("/admin/v1/organizations", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  updateOrganization: (id: string, body: { name: string }) =>
    request<{ organization: Organization }>(`/admin/v1/organizations/${id}`, {
      method: "PATCH",
      body: JSON.stringify(body),
    }),
};

export function organizationApi(organizationId: string) {
  const base = `/admin/v1/organizations/${organizationId}`;
  return {
    meters: (page?: PageParams) =>
      request<{ meters: Meter[]; page_info: PageInfo }>(
        pageURL(`${base}/meters`, page),
      ),
    meter: (id: string) => request<{ meter: Meter }>(`${base}/meters/${id}`),
    createMeter: (
      body: Pick<Meter, "Code" | "Name" | "Aggregation" | "Unit">,
    ) =>
      request<{ meter: Meter }>(`${base}/meters`, {
        method: "POST",
        body: JSON.stringify({
          code: body.Code,
          name: body.Name,
          aggregation: body.Aggregation,
          unit: body.Unit,
        }),
      }),
    updateMeter: (
      id: string,
      body: Pick<Meter, "Code" | "Name" | "Aggregation" | "Unit">,
    ) =>
      request<{ meter: Meter }>(`${base}/meters/${id}`, {
        method: "PATCH",
        body: JSON.stringify({
          code: body.Code,
          name: body.Name,
          aggregation: body.Aggregation,
          unit: body.Unit,
        }),
      }),
    customers: (page?: PageParams) =>
      request<{ customers: Customer[]; page_info: PageInfo }>(
        pageURL(`${base}/customers`, page),
      ),
    customer: (id: string) =>
      request<{ customer: Customer }>(`${base}/customers/${id}`),
    createCustomer: (body: { first_name: string; last_name: string }) =>
      request<{ customer: Customer }>(`${base}/customers`, {
        method: "POST",
        body: JSON.stringify(body),
      }),
    updateCustomer: (
      id: string,
      body: { first_name: string; last_name: string },
    ) =>
      request<{ customer: Customer }>(`${base}/customers/${id}`, {
        method: "PATCH",
        body: JSON.stringify(body),
      }),
    products: (page?: PageParams) =>
      request<{ products: Product[]; page_info: PageInfo }>(
        pageURL(`${base}/products`, page),
      ),
    product: (id: string) =>
      request<{ product: Product }>(`${base}/products/${id}`),
    createProduct: (body: Record<string, unknown>) =>
      request<{ product: Product }>(`${base}/products`, {
        method: "POST",
        body: JSON.stringify(body),
      }),
    updateProduct: (id: string, body: Record<string, unknown>) =>
      request<{ product: Product }>(`${base}/products/${id}`, {
        method: "PATCH",
        body: JSON.stringify(body),
      }),
    prices: (page?: PageParams) =>
      request<{ prices: Price[]; page_info: PageInfo }>(
        pageURL(`${base}/prices`, page),
      ),
    currencies: () =>
      request<{ currencies: Currency[] }>(`${base}/reference/currencies`),
    measurementUnits: () =>
      request<{ measurement_units: MeasurementUnit[] }>(
        `${base}/reference/measurement-units`,
      ),
    price: (id: string) => request<{ price: Price }>(`${base}/prices/${id}`),
    createPrice: (body: Record<string, unknown>) =>
      request<{ price: Price }>(`${base}/prices`, {
        method: "POST",
        body: JSON.stringify(body),
      }),
    updatePrice: (id: string, body: Record<string, unknown>) =>
      request<{ price: Price }>(`${base}/prices/${id}`, {
        method: "PATCH",
        body: JSON.stringify(body),
      }),
    subscriptions: (page?: PageParams) =>
      request<{ subscriptions: Subscription[]; page_info: PageInfo }>(
        pageURL(`${base}/subscriptions`, page),
      ),
    subscription: (id: string) =>
      request<{ subscription: Subscription }>(`${base}/subscriptions/${id}`),
    createSubscription: (body: Record<string, unknown>) =>
      request<{ subscription: Subscription }>(`${base}/subscriptions`, {
        method: "POST",
        body: JSON.stringify(body),
      }),
    updateSubscription: (id: string, body: Record<string, unknown>) =>
      request<{ subscription: Subscription }>(`${base}/subscriptions/${id}`, {
        method: "PATCH",
        body: JSON.stringify(body),
      }),
    usageEvents: (page?: PageParams) =>
      request<{ events: UsageEvent[]; page_info: PageInfo }>(
        pageURL(`${base}/usage-events`, page),
      ),
    usageSummary: (range: "7d" | "30d" | "3m" | "12m" = "12m") =>
      request<UsageSummary>(`${base}/usage-events/summary?range=${range}`),
    usageEvent: (id: string) =>
      request<{ event: UsageEvent }>(`${base}/usage-events/${id}`),
    createUsageEvents: (body: Record<string, unknown>) =>
      request<{ events: UsageEvent[] }>(`${base}/usage-events`, {
        method: "POST",
        headers: { "Idempotency-Key": crypto.randomUUID() },
        body: JSON.stringify(body),
      }),
    invoices: (page?: PageParams) =>
      request<{ invoices: Invoice[]; page_info: PageInfo }>(
        pageURL(`${base}/invoices`, page),
      ),
    invoiceNumberSettings: () =>
      request<{ settings: InvoiceNumberSettings }>(
        `${base}/invoice-number-settings`,
      ),
    updateInvoiceNumberSettings: (numberFormat: string) =>
      request<{ settings: InvoiceNumberSettings }>(
        `${base}/invoice-number-settings`,
        {
          method: "PATCH",
          body: JSON.stringify({ number_format: numberFormat }),
        },
      ),
    generateInvoices: (body: { period_start: string; period_end: string }) =>
      request<{ invoices: Invoice[]; created: number; existing: number }>(
        `${base}/invoice-runs`,
        {
          method: "POST",
          body: JSON.stringify(body),
        },
      ),
    invoice: (id: string) =>
      request<{ invoice: Invoice }>(`${base}/invoices/${id}`),
    createInvoice: (body: Record<string, unknown>) =>
      request<{ invoice: Invoice }>(`${base}/invoices`, {
        method: "POST",
        body: JSON.stringify(body),
      }),
    updateInvoice: (id: string, body: Record<string, unknown>) =>
      request<{ invoice: Invoice }>(`${base}/invoices/${id}`, {
        method: "PATCH",
        body: JSON.stringify(body),
      }),
  };
}

export function iamApi(organization: Organization) {
  const query = encodeURIComponent(organization.slug);
  return {
    testPermissions: (permissions: string[]) =>
      request<{ permissions: string[] }>("/admin/v1/iam:testPermissions", {
        method: "POST",
        body: JSON.stringify({
          resource: `organizations/${organization.slug}`,
          permissions,
        }),
      }),
    users: (page?: PageParams) =>
      request<{ users: DirectoryUser[]; page_info: PageInfo }>(
        pageURL(`/admin/v1/iam/users?organization=${query}`, page),
      ),
    serviceAccounts: (page?: PageParams) =>
      request<{ service_accounts: ServiceAccount[]; page_info: PageInfo }>(
        pageURL(`/admin/v1/iam/serviceAccounts?organization=${query}`, page),
      ),
    serviceAccount: (id: string) =>
      request<ServiceAccount>(
        `/admin/v1/iam/serviceAccount?organization=${query}&id=${id}`,
      ),
    createServiceAccount: (body: {
      display_name: string;
      description: string;
      issuer: string;
      subject: string;
    }) =>
      request<ServiceAccount>(
        `/admin/v1/iam/serviceAccounts?organization=${query}`,
        { method: "POST", body: JSON.stringify(body) },
      ),
    updateServiceAccount: (body: {
      id: string;
      display_name: string;
      description: string;
    }) =>
      request<ServiceAccount>(
        `/admin/v1/iam/serviceAccount?organization=${query}`,
        { method: "PATCH", body: JSON.stringify(body) },
      ),
    disableServiceAccount: (id: string) =>
      request<void>(
        `/admin/v1/iam/serviceAccount:disable?organization=${query}&id=${id}`,
        { method: "POST" },
      ),
    apiKeys: (serviceAccountId: string, page?: PageParams) =>
      request<{ api_keys: APIKey[]; page_info: PageInfo }>(
        pageURL(
          `/admin/v1/iam/serviceAccounts/${serviceAccountId}/apiKeys?organization=${query}`,
          page,
        ),
      ),
    createAPIKey: (
      serviceAccountId: string,
      body: { display_name: string; expires_at?: string },
    ) =>
      request<APIKey>(
        `/admin/v1/iam/serviceAccounts/${serviceAccountId}/apiKeys?organization=${query}`,
        { method: "POST", body: JSON.stringify(body) },
      ),
    revokeAPIKey: (id: string) =>
      request<void>(
        `/admin/v1/iam/apiKeys/${id}/revoke?organization=${query}`,
        { method: "POST" },
      ),
    roles: (page?: PageParams) =>
      request<{ roles: IAMRole[]; page_info: PageInfo }>(
        pageURL(`/admin/v1/iam/roles?organization=${query}`, page),
      ),
    createRole: (body: {
      name: string;
      display_name: string;
      description: string;
      permissions: string[];
    }) =>
      request<IAMRole>(`/admin/v1/iam/roles?organization=${query}`, {
        method: "POST",
        body: JSON.stringify(body),
      }),
    updateRole: (body: {
      id: string;
      name: string;
      display_name: string;
      description: string;
      permissions: string[];
      etag: string;
    }) =>
      request<IAMRole>(
        `/admin/v1/iam/role?organization_id=${organization.id}`,
        { method: "PATCH", body: JSON.stringify(body) },
      ),
    policy: () =>
      request<IAMPolicy>(
        `/admin/v1/iam/policy?resource=${encodeURIComponent(`organizations/${organization.slug}`)}`,
      ),
    setPolicy: (policy: IAMPolicy) =>
      request<IAMPolicy>(
        `/admin/v1/iam/policy?resource=${encodeURIComponent(policy.resource)}`,
        {
          method: "PUT",
          body: JSON.stringify({
            etag: policy.etag,
            bindings: policy.bindings,
          }),
        },
      ),
  };
}
