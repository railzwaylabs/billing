import { useEffect, useState } from "react";
import { Navigate, Route, Routes, useParams } from "react-router-dom";
import { api, type Organization } from "@/api";
import { ConsoleLayout } from "@/components/console-layout";
import { DashboardPage } from "@/pages/dashboard-page";
import { LoginPage } from "@/pages/login-page";
import { MetersPage } from "@/pages/meters-page";
import { CustomersPage } from "@/pages/customers-page";
import { ProductsPage } from "@/pages/products-page";
import { PricesPage } from "@/pages/prices-page";
import { SubscriptionsPage } from "@/pages/subscriptions-page";
import { UsagePage } from "@/pages/usage-page";
import { InvoicesPage } from "@/pages/invoices-page";
import { APIKeysPage } from "@/pages/api-keys-page";
import { ServiceAccountsPage } from "@/pages/service-accounts-page";
import { SettingsPage } from "@/pages/settings-page";
import { IAMPage } from "@/pages/iam-page";
import { MembersPage } from "@/pages/members-page";
import { OrganizationOnboardingPage } from "@/pages/organization-onboarding-page";
import { PasswordSetupPage } from "@/pages/password-setup-page";
import { MonitorPage, ServiceMonitorPage } from "@/pages/monitor-page";
import { LogsPage } from "@/pages/logs-page";

export type SessionUser = { id: string; username: string };

export function App() {
  const [user, setUser] = useState<SessionUser | null>(null);
  const [passwordSetupRequired, setPasswordSetupRequired] = useState(false);
  const [organizations, setOrganizations] = useState<Organization[] | null>(
    null,
  );
  const [sessionLoading, setSessionLoading] = useState(true);

  useEffect(() => {
    let active = true;
    async function restoreSession() {
      try {
        const session = await api.session();
        if (!active) return;
        setUser(session.user);
        setPasswordSetupRequired(session.show_password_change);
        const result = await api.organizations();
        if (active) setOrganizations(result.organizations);
      } catch {
        if (active) setUser(null);
      } finally {
        if (active) setSessionLoading(false);
      }
    }
    void restoreSession();
    return () => {
      active = false;
    };
  }, []);

  if (sessionLoading) return null;
  if (!user)
    return (
      <LoginPage
        onAuthenticated={(value, required) => {
          setUser(value);
          setPasswordSetupRequired(required);
          api
            .organizations()
            .then((result) => setOrganizations(result.organizations))
            .catch(() => setOrganizations([]));
        }}
      />
    );
  if (passwordSetupRequired)
    return (
      <PasswordSetupPage onComplete={() => setPasswordSetupRequired(false)} />
    );
  if (organizations === null) return null;
  if (organizations.length === 0)
    return (
      <OrganizationOnboardingPage
        onCreated={(organization) => setOrganizations([organization])}
      />
    );
  return (
    <Routes>
      <Route
        path="/"
        element={
          <Navigate to={`/organizations/${organizations[0].id}`} replace />
        }
      />
      <Route
        path="organizations/:organizationId"
        element={
          <OrganizationConsole
            user={user}
            organizations={organizations}
            onOrganizationUpdated={(updated) =>
              setOrganizations(
                organizations.map((item) =>
                  item.id === updated.id ? updated : item,
                ),
              )
            }
            onLogout={async () => {
              await api.logout();
              setUser(null);
            }}
          />
        }
      >
        <Route index element={<DashboardPage />} />

        <Route path="meters" element={<MetersPage />} />
        <Route path="meters/new" element={<MetersPage />} />
        <Route path="meters/:meterId" element={<MetersPage />} />

        <Route path="catalog" element={<Navigate to="products" replace />} />
        <Route path="catalog/products" element={<ProductsPage />} />
        <Route path="catalog/products/new" element={<ProductsPage />} />
        <Route path="catalog/products/:productId" element={<ProductsPage />} />
        <Route path="catalog/prices" element={<PricesPage />} />
        <Route path="catalog/prices/new" element={<PricesPage />} />
        <Route path="catalog/prices/:priceId" element={<PricesPage />} />

        <Route path="subscriptions" element={<SubscriptionsPage />} />
        <Route path="subscriptions/new" element={<SubscriptionsPage />} />
        <Route
          path="subscriptions/:subscriptionId"
          element={<SubscriptionsPage />}
        />

        <Route path="usage" element={<UsagePage />} />
        <Route path="usage/ingest" element={<UsagePage />} />

        <Route path="customers" element={<CustomersPage />} />
        <Route path="customers/new" element={<CustomersPage />} />
        <Route path="customers/:customerId" element={<CustomersPage />} />

        <Route path="invoices" element={<InvoicesPage />} />
        <Route path="invoices/new" element={<InvoicesPage />} />
        <Route path="invoices/:invoiceId" element={<InvoicesPage />} />

        <Route
          path="developer"
          element={<Navigate to="service-accounts" replace />}
        />
        <Route
          path="developer/service-accounts"
          element={<ServiceAccountsPage />}
        />
        <Route
          path="developer/service-accounts/new"
          element={<ServiceAccountsPage />}
        />
        <Route
          path="developer/service-accounts/:serviceAccountId"
          element={<ServiceAccountsPage />}
        />

        <Route path="developer/api-keys" element={<APIKeysPage />} />
        <Route path="developer/api-keys/new" element={<APIKeysPage />} />

        <Route path="developer/monitor" element={<MonitorPage />} />
        <Route
          path="developer/monitor/:serviceId"
          element={<ServiceMonitorPage />}
        />
        <Route path="developer/logs" element={<LogsPage />} />

        <Route path="iam" element={<Navigate to="roles" replace />} />
        <Route path="iam/roles" element={<IAMPage />} />
        <Route path="iam/roles/new" element={<IAMPage />} />
        <Route path="iam/roles/:roleId" element={<IAMPage />} />
        <Route path="iam/members" element={<MembersPage />} />
        <Route path="iam/policy" element={<IAMPage />} />

        <Route path="settings" element={<SettingsPage />} />
        <Route
          path="*"
          element={
            <Navigate to={`/organizations/${organizations[0].id}`} replace />
          }
        />
      </Route>
    </Routes>
  );
}

function OrganizationConsole({
  user,
  organizations,
  onLogout,
  onOrganizationUpdated,
}: {
  user: SessionUser;
  organizations: Organization[];
  onLogout: () => void;
  onOrganizationUpdated: (organization: Organization) => void;
}) {
  const { organizationId } = useParams();
  const organization = organizations.find((item) => item.id === organizationId);
  if (!organization)
    return <Navigate to={`/organizations/${organizations[0].id}`} replace />;
  return (
    <ConsoleLayout
      user={user}
      organization={organization}
      organizations={organizations}
      onLogout={onLogout}
      onOrganizationUpdated={onOrganizationUpdated}
    />
  );
}
