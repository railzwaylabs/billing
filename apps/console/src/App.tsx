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
import { MonitorPage } from "@/pages/monitor-page";

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

        <Route path="meters" element={<MetersPage />}>
          <Route path="new" element={<MetersPage />} />
          <Route path=":meterId" element={<MetersPage />} />
        </Route>

        <Route path="catalog" element={<Navigate to="products" replace />} />
        <Route path="catalog/products" element={<ProductsPage />}>
          <Route path="new" element={<ProductsPage />} />
          <Route path=":productId" element={<ProductsPage />} />
        </Route>
        <Route path="catalog/prices" element={<PricesPage />}>
          <Route path="new" element={<PricesPage />} />
          <Route path=":priceId" element={<PricesPage />} />
        </Route>

        <Route path="subscriptions" element={<SubscriptionsPage />}>
          <Route path="new" element={<SubscriptionsPage />} />
          <Route path=":subscriptionId" element={<SubscriptionsPage />} />
        </Route>

        <Route path="usage" element={<UsagePage />}>
          <Route path="ingest" element={<UsagePage />} />
        </Route>

        <Route path="customers" element={<CustomersPage />}>
          <Route path="new" element={<CustomersPage />} />
          <Route path=":customerId" element={<CustomersPage />} />
        </Route>

        <Route path="invoices" element={<InvoicesPage />}>
          <Route path="new" element={<InvoicesPage />} />
          <Route path=":invoiceId" element={<InvoicesPage />} />
        </Route>

        <Route
          path="developer"
          element={<Navigate to="service-accounts" replace />}
        />
        <Route
          path="developer/service-accounts"
          element={<ServiceAccountsPage />}
        >
          <Route path="new" element={<ServiceAccountsPage />} />
          <Route
            path=":serviceAccountId"
            element={<ServiceAccountsPage />}
          />
        </Route>
        <Route path="developer/api-keys" element={<APIKeysPage />}>
          <Route path="new" element={<APIKeysPage />} />
        </Route>
        <Route path="developer/monitor" element={<MonitorPage />} />

        <Route path="iam" element={<Navigate to="roles" replace />} />
        <Route path="iam/roles" element={<IAMPage />}>
          <Route path="new" element={<IAMPage />} />
          <Route path=":roleId" element={<IAMPage />} />
        </Route>
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
