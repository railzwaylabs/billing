import { ResourceAllocation } from "@/components/resource-allocation";

export function MonitorPage() {
  return (
    <main className="w-full flex-1 space-y-6 p-4 md:p-6 lg:p-8">
      <div className="space-y-1">
        <p className="text-sm font-medium text-muted-foreground">Developer</p>
        <h1 className="text-2xl font-semibold tracking-tight md:text-3xl">
          Monitor
        </h1>
        <p className="text-sm text-muted-foreground">
          Monitor compute allocation and network activity for the Billing
          runtime.
        </p>
      </div>

      <ResourceAllocation />
    </main>
  );
}
