import {
  Activity,
  BarChart3,
  Boxes,
  ChevronRight,
  ChevronsUpDown,
  CircleDollarSign,
  Gauge,
  KeyRound,
  LogOut,
  Settings2,
  ShieldCheck,
  Users,
  Workflow,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Fragment, useEffect, useState } from "react";
import { Link, NavLink, Outlet, useLocation } from "react-router-dom";
import type { SessionUser } from "@/App";
import { iamApi, type Organization } from "@/api";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { appName } from "@/lib/app-config";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarProvider,
  SidebarRail,
  SidebarTrigger,
} from "@/components/ui/sidebar";

type NavigationItem = {
  path: string;
  label: string;
  icon: LucideIcon;
  end?: boolean;
  children?: Array<{
    path: string;
    label: string;
    permissions?: string[];
  }>;
};

const navigation: NavigationItem[] = [
  { path: "", label: "Overview", icon: BarChart3, end: true },
  { path: "meters", label: "Meters", icon: Gauge },
  {
    path: "catalog",
    label: "Catalog",
    icon: Boxes,
    children: [
      { path: "catalog/products", label: "Products" },
      { path: "catalog/prices", label: "Prices" },
    ],
  },
  { path: "subscriptions", label: "Subscriptions", icon: Workflow },
  { path: "usage", label: "Usage", icon: Activity },
  { path: "customers", label: "Customers", icon: Users },
  { path: "invoices", label: "Invoices", icon: CircleDollarSign },
  {
    path: "developer",
    label: "Developer",
    icon: KeyRound,
    children: [
      {
        path: "developer/service-accounts",
        label: "Service accounts",
        permissions: ["billing.serviceAccounts.list"],
      },
      {
        path: "developer/api-keys",
        label: "API keys",
        permissions: ["billing.serviceAccounts.list", "billing.apiKeys.list"],
      },
      {
        path: "developer/monitor",
        label: "Monitor",
        permissions: ["billing.monitoring.get"],
      },
      {
        path: "developer/logs",
        label: "Logs",
        permissions: ["billing.logs.list"],
      },
    ],
  },
  {
    path: "iam",
    label: "IAM",
    icon: ShieldCheck,
    children: [
      { path: "iam/members", label: "Members" },
      { path: "iam/roles", label: "Roles" },
      { path: "iam/policy", label: "Policy" },
    ],
  },
  { path: "settings", label: "Settings", icon: Settings2 },
];

const breadcrumbLabels: Record<string, string> = {
  meters: "Meters",
  catalog: "Catalog",
  products: "Products",
  prices: "Prices",
  subscriptions: "Subscriptions",
  usage: "Usage",
  ingest: "Ingest usage",
  customers: "Customers",
  invoices: "Invoices",
  developer: "Developer",
  "service-accounts": "Service accounts",
  "api-keys": "API keys",
  monitor: "Monitor",
  logs: "Logs",
  iam: "IAM",
  members: "Members",
  roles: "Roles",
  policy: "Policy",
  settings: "Settings",
};

const singularLabels: Record<string, string> = {
  meters: "meter",
  products: "product",
  prices: "price",
  subscriptions: "subscription",
  customers: "customer",
  invoices: "invoice",
  "service-accounts": "service account",
  "api-keys": "API key",
  roles: "role",
  monitor: "service",
};

type BreadcrumbEntry = { label: string; to?: string };

function buildBreadcrumbs(
  base: string,
  organizationName: string,
  segments: string[],
): BreadcrumbEntry[] {
  const entries: BreadcrumbEntry[] = [{ label: organizationName, to: base }];

  if (segments.length === 0) {
    entries.push({ label: "Overview" });
    return entries;
  }

  const path: string[] = [];
  segments.forEach((segment, index) => {
    path.push(segment);
    const last = index === segments.length - 1;
    const previous = segments[index - 1] ?? "resource";

    if (segment === "new") {
      entries.push({
        label: `Create ${singularLabels[previous] ?? "resource"}`,
      });
      return;
    }

    const label = breadcrumbLabels[segment];
    if (label) {
      entries.push({
        label,
        to: last ? undefined : `${base}/${path.join("/")}`,
      });
      return;
    }

    // Resource identifiers must never leak into navigation labels. Detail
    // pages use a stable semantic title and render the resource name inside
    // the page once it has been loaded.
    entries.push({
      label: `${singularLabels[previous] ?? "Resource"} details`,
    });
  });

  return entries;
}

export function ConsoleLayout({
  user,
  organization,
  organizations,
  onLogout,
  onOrganizationUpdated,
}: {
  user: SessionUser;
  organization: Organization;
  organizations: Organization[];
  onLogout: () => void;
  onOrganizationUpdated: (organization: Organization) => void;
}) {
  const base = `/organizations/${organization.id}`;
  const organizationInitial =
    organization.name.trim().charAt(0).toUpperCase() || "O";
  const routerLocation = useLocation();
  const relativePath = routerLocation.pathname
    .slice(base.length)
    .replace(/^\//, "");
  const segments = relativePath.split("/").filter(Boolean);
  const breadcrumbs = buildBreadcrumbs(base, organization.name, segments);
  const pageTitle = breadcrumbs.at(-1)?.label;
  const [allowedPermissions, setAllowedPermissions] = useState<Set<string>>(
    new Set(),
  );

  useEffect(() => {
    let active = true;
    setAllowedPermissions(new Set());
    const permissions = [
      "billing.serviceAccounts.list",
      "billing.apiKeys.list",
      "billing.monitoring.get",
      "billing.logs.list",
    ];
    iamApi(organization)
      .testPermissions(permissions)
      .then((result) => {
        if (active) setAllowedPermissions(new Set(result.permissions));
      })
      .catch(() => {
        if (active) setAllowedPermissions(new Set());
      });
    return () => {
      active = false;
    };
  }, [organization]);

  const visibleNavigation = navigation.flatMap((item) => {
    if (!item.children) return [item];
    const children = item.children.filter((child) =>
      (child.permissions ?? []).every((permission) =>
        allowedPermissions.has(permission),
      ),
    );
    return children.length > 0 ? [{ ...item, children }] : [];
  });

  useEffect(() => {
    document.title = pageTitle
      ? `${pageTitle} · ${organization.name} · ${appName}`
      : `${organization.name} · ${appName}`;

    return () => {
      document.title = appName;
    };
  }, [organization.name, pageTitle]);

  return (
    <SidebarProvider
      style={
        {
          "--sidebar-width": "calc(var(--spacing) * 72)",
          "--header-height": "calc(var(--spacing) * 12)",
        } as React.CSSProperties
      }
    >
      <Sidebar variant="inset" collapsible="offcanvas">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton size="lg" asChild>
                <NavLink to={base}>
                  <span className="flex aspect-square size-8 items-center justify-center rounded-lg bg-primary font-semibold text-primary-foreground">
                    {organizationInitial}
                  </span>
                  <span className="grid flex-1 text-left text-sm leading-tight">
                    <span className="truncate font-semibold">
                      {organization.name}
                    </span>
                    <span className="truncate text-xs text-muted-foreground">
                      Billing console
                    </span>
                  </span>
                </NavLink>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>
        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupLabel>Billing</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {visibleNavigation.map((item) => (
                  <NavigationMenuItem
                    key={item.path || "overview"}
                    item={item}
                    base={base}
                    currentPath={relativePath}
                  />
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>
        <SidebarFooter>
          <SidebarMenu>
            <SidebarMenuItem>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <SidebarMenuButton size="lg">
                    <Avatar className="size-8 rounded-lg">
                      <AvatarFallback className="rounded-lg">
                        {user.username.slice(0, 2).toUpperCase()}
                      </AvatarFallback>
                    </Avatar>
                    <span className="grid flex-1 text-left text-sm leading-tight">
                      <span className="truncate font-medium">
                        {user.username}
                      </span>
                      <span className="truncate text-xs">Administrator</span>
                    </span>
                    <ChevronsUpDown className="ml-auto size-4" />
                  </SidebarMenuButton>
                </DropdownMenuTrigger>
                <DropdownMenuContent
                  className="w-[var(--radix-dropdown-menu-trigger-width)] min-w-56 rounded-lg"
                  side="right"
                  align="end"
                  sideOffset={4}
                >
                  <DropdownMenuLabel className="font-normal">
                    <div className="flex items-center gap-2 px-1 py-1.5">
                      <Avatar className="size-8 rounded-lg">
                        <AvatarFallback className="rounded-lg">
                          {user.username.slice(0, 2).toUpperCase()}
                        </AvatarFallback>
                      </Avatar>
                      <div className="grid flex-1 text-left text-sm leading-tight">
                        <span className="truncate font-medium">
                          {user.username}
                        </span>
                        <span className="truncate text-xs text-muted-foreground">
                          Administrator
                        </span>
                      </div>
                    </div>
                  </DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onSelect={onLogout}>
                    <LogOut />
                    Log out
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </SidebarMenuItem>
          </SidebarMenu>
          <p className="px-2 pb-1 text-center text-[11px] text-muted-foreground">
            Powered by{" "}
            <span className="font-medium text-foreground/70">Railzway</span>
          </p>
        </SidebarFooter>
        <SidebarRail />
      </Sidebar>
      <SidebarInset>
        <header className="flex h-(--header-height) shrink-0 items-center justify-between gap-2 border-b px-4 transition-[width,height] ease-linear">
          <div className="flex items-center gap-2">
            <SidebarTrigger className="-ml-1" />
            <Separator
              orientation="vertical"
              className="mr-2 data-[orientation=vertical]:h-4"
            />
            <Breadcrumb>
              <BreadcrumbList>
                {breadcrumbs.map((entry, index) => {
                  const last = index === breadcrumbs.length - 1;
                  return (
                    <Fragment key={`${entry.label}-${index}`}>
                      <BreadcrumbItem
                        className={
                          index === 0 ? "hidden md:inline-flex" : undefined
                        }
                      >
                        {entry.to && !last ? (
                          <BreadcrumbLink asChild>
                            <Link to={entry.to}>{entry.label}</Link>
                          </BreadcrumbLink>
                        ) : (
                          <BreadcrumbPage className="max-w-52 truncate md:max-w-72">
                            {entry.label}
                          </BreadcrumbPage>
                        )}
                      </BreadcrumbItem>
                      {!last && (
                        <BreadcrumbSeparator
                          className={
                            index === 0 ? "hidden md:block" : undefined
                          }
                        />
                      )}
                    </Fragment>
                  );
                })}
              </BreadcrumbList>
            </Breadcrumb>
          </div>
          <Select
            value={organization.id}
            onValueChange={(value) =>
              window.location.assign(`/organizations/${value}`)
            }
          >
            <SelectTrigger
              className="w-[220px]"
              aria-label="Active organization"
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {organizations.map((item) => (
                <SelectItem key={item.id} value={item.id}>
                  {item.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </header>
        <div className="flex flex-1 flex-col">
          <div className="@container/main flex flex-1 flex-col gap-2">
            <Outlet
              key={routerLocation.pathname}
              context={{ organization, onOrganizationUpdated }}
            />
          </div>
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}

function NavigationMenuItem({
  item,
  base,
  currentPath,
}: {
  item: NavigationItem;
  base: string;
  currentPath: string;
}) {
  const Icon = item.icon;
  const active = item.path
    ? currentPath === item.path || currentPath.startsWith(`${item.path}/`)
    : !currentPath;
  const [open, setOpen] = useState(active);
  useEffect(() => {
    if (active) setOpen(true);
  }, [active]);
  if (!item.children)
    return (
      <SidebarMenuItem>
        <SidebarMenuButton asChild isActive={active} tooltip={item.label}>
          <NavLink
            to={item.path ? `${base}/${item.path}` : base}
            end={item.end}
          >
            <Icon />
            <span>{item.label}</span>
          </NavLink>
        </SidebarMenuButton>
      </SidebarMenuItem>
    );
  return (
    <Collapsible
      asChild
      open={open}
      onOpenChange={setOpen}
      className="group/collapsible"
    >
      <SidebarMenuItem>
        <CollapsibleTrigger asChild>
          <SidebarMenuButton tooltip={item.label}>
            <Icon />
            <span>{item.label}</span>
            <ChevronRight className="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
          </SidebarMenuButton>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <SidebarMenuSub>
            {item.children.map((child) => (
              <SidebarMenuSubItem key={child.path}>
                <SidebarMenuSubButton
                  asChild
                  isActive={
                    currentPath === child.path ||
                    currentPath.startsWith(`${child.path}/`)
                  }
                >
                  <NavLink to={`${base}/${child.path}`}>
                    <span>{child.label}</span>
                  </NavLink>
                </SidebarMenuSubButton>
              </SidebarMenuSubItem>
            ))}
          </SidebarMenuSub>
        </CollapsibleContent>
      </SidebarMenuItem>
    </Collapsible>
  );
}
