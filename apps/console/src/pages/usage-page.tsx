import { useEffect, useState } from "react";
import { ArrowLeft, Plus, Trash2 } from "lucide-react";
import { Link, useNavigate, useOutletContext } from "react-router-dom";
import {
  organizationApi,
  type Customer,
  type Meter,
  type Organization,
  type UsageEvent,
} from "@/api";
import { RelationCombobox } from "@/components/relation-combobox";
import { DataTable } from "@/components/data-table";
import { DateTimePicker } from "@/components/date-picker";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";

type Draft = {
  eventId: string;
  meterId: string;
  customerId: string;
  value: string;
  eventTime: string;
};
const empty = (): Draft => ({
  eventId: "",
  meterId: "",
  customerId: "",
  value: "1",
  eventTime: new Date().toISOString().slice(0, 16),
});
const micros = (value: string) => {
  const [whole = "0", fraction = ""] = value.split(".");
  return Number(
    BigInt(whole || "0") * 1_000_000n +
      BigInt((fraction + "000000").slice(0, 6)),
  );
};

export function UsagePage() {
  const { organization } = useOutletContext<{ organization: Organization }>();
  const navigate = useNavigate();
  const client = organizationApi(organization.id);
  const ingest = location.pathname.endsWith("/ingest");
  const [events, setEvents] = useState<UsageEvent[]>([]);
  const [meters, setMeters] = useState<Meter[]>([]);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [drafts, setDrafts] = useState<Draft[]>([empty()]);
  const [error, setError] = useState("");
  useEffect(() => {
    void Promise.all([
      client.meters(),
      client.customers(),
      ...(ingest ? [] : [client.usageEvents()]),
    ])
      .then(([m, c, e]) => {
        setMeters(m.meters);
        setCustomers(c.customers);
        if (e) setEvents(e.events);
      })
      .catch((cause) => setError(cause.message));
  }, [organization.id, ingest]);
  function change(index: number, patch: Partial<Draft>) {
    setDrafts((values) =>
      values.map((value, position) =>
        position === index ? { ...value, ...patch } : value,
      ),
    );
  }
  async function submit() {
    try {
      await client.createUsageEvents({
        events: drafts.map((draft) => ({
          event_id: draft.eventId,
          meter_id: draft.meterId,
          customer_id: draft.customerId,
          value_micros: micros(draft.value),
          event_time: new Date(draft.eventTime).toISOString(),
        })),
      });
      navigate("..");
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Unable to ingest usage",
      );
    }
  }

  if (ingest)
    return (
      <main className="content editor-page usage-editor">
        <Button variant="ghost" size="sm" asChild className="mb-6 -ml-3">
          <Link to="..">
            <ArrowLeft />
            Usage events
          </Link>
        </Button>
        <div className="page-head">
          <div>
            <p className="eyebrow">USAGE</p>
            <h1>Ingest usage</h1>
            <p className="muted">
              Send one or many idempotent events in a single batch.
            </p>
          </div>
        </div>
        <Card className="mt-8">
          <CardHeader>
            <CardTitle>Events</CardTitle>
            <CardDescription>
              Every event needs a unique ID, meter, customer, value, and
              occurrence time.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {drafts.map((draft, index) => (
              <div className="rounded-lg border bg-muted/20 p-4" key={index}>
                <div className="mb-4 flex items-center justify-between">
                  <span className="text-sm font-medium">Event {index + 1}</span>
                  <Button
                    type="button"
                    size="icon"
                    variant="ghost"
                    aria-label={`Remove event ${index + 1}`}
                    disabled={drafts.length === 1}
                    onClick={() =>
                      setDrafts(
                        drafts.filter((_, position) => position !== index),
                      )
                    }
                  >
                    <Trash2 />
                  </Button>
                </div>
                <FieldGroup className="grid gap-4 md:grid-cols-2">
                  <Field>
                    <FieldLabel htmlFor={`event-id-${index}`}>
                      Event ID
                    </FieldLabel>
                    <Input
                      type="text"
                      inputMode="text"
                      pattern="[A-Za-z0-9][A-Za-z0-9._:-]*"
                      spellCheck={false}
                      id={`event-id-${index}`}
                      required
                      placeholder="evt-001"
                      value={draft.eventId}
                      onChange={(event) =>
                        change(index, { eventId: event.target.value })
                      }
                    />
                  </Field>
                  <Field>
                    <FieldLabel>Meter</FieldLabel>
                    <RelationCombobox
                      value={draft.meterId}
                      onValueChange={(value) =>
                        change(index, { meterId: value })
                      }
                      options={meters.map((meter) => ({
                        value: meter.ID,
                        label: meter.Name,
                        description: `${meter.Code} · ${meter.Unit}`,
                      }))}
                      placeholder="Select meter"
                      searchPlaceholder="Search meters…"
                    />
                  </Field>
                  <Field>
                    <FieldLabel>Customer</FieldLabel>
                    <RelationCombobox
                      value={draft.customerId}
                      onValueChange={(value) =>
                        change(index, { customerId: value })
                      }
                      options={customers.map((customer) => ({
                        value: customer.ID,
                        label: `${customer.FirstName} ${customer.LastName}`,
                      }))}
                      placeholder="Select customer"
                      searchPlaceholder="Search customers…"
                    />
                  </Field>
                  <Field>
                    <FieldLabel htmlFor={`event-value-${index}`}>
                      Value
                    </FieldLabel>
                    <Input
                      type="text"
                      id={`event-value-${index}`}
                      inputMode="decimal"
                      pattern="-?[0-9]+(?:\.[0-9]+)?"
                      value={draft.value}
                      onChange={(event) =>
                        change(index, { value: event.target.value })
                      }
                    />
                  </Field>
                  <Field className="md:col-span-2">
                    <FieldLabel htmlFor={`event-time-${index}`}>
                      Occurred at
                    </FieldLabel>
                    <DateTimePicker
                      value={draft.eventTime}
                      onValueChange={(value) =>
                        change(index, { eventTime: value })
                      }
                    />
                  </Field>
                </FieldGroup>
              </div>
            ))}
            {error && <p className="form-error">{error}</p>}
            <div className="flex items-center justify-between border-t pt-4">
              <Button
                type="button"
                variant="outline"
                onClick={() => setDrafts([...drafts, empty()])}
              >
                <Plus />
                Add event
              </Button>
              <Button
                disabled={drafts.some(
                  (draft) =>
                    !draft.eventId || !draft.meterId || !draft.customerId,
                )}
                onClick={submit}
              >
                Ingest {drafts.length} event{drafts.length === 1 ? "" : "s"}
              </Button>
            </div>
          </CardContent>
        </Card>
      </main>
    );

  return (
    <main className="content">
      <div className="page-head">
        <div>
          <p className="eyebrow">USAGE</p>
          <h1>Usage events</h1>
          <p className="muted">
            Immutable event ledger for rating and invoicing.
          </p>
        </div>
        <Button asChild>
          <Link to="ingest">
            <Plus />
            Ingest usage
          </Link>
        </Button>
      </div>
      <Card className="mt-8">
        <CardContent>
          <DataTable
            data={events}
            searchKey="EventID"
            searchPlaceholder="Search event ID…"
            columns={[
              {
                accessorKey: "EventID",
                header: "Event ID",
                cell: ({ row }) => (
                  <span className="font-medium">{row.original.EventID}</span>
                ),
              },
              {
                id: "meter",
                header: "Meter",
                cell: ({ row }) =>
                  meters.find((meter) => meter.ID === row.original.MeterID)
                    ?.Name ?? "—",
              },
              {
                id: "customer",
                header: "Customer",
                cell: ({ row }) => {
                  const customer = customers.find(
                    (item) => item.ID === row.original.CustomerID,
                  );
                  return customer
                    ? `${customer.FirstName} ${customer.LastName}`
                    : "—";
                },
              },
              {
                id: "value",
                header: "Value",
                cell: ({ row }) => row.original.Value.Micros / 1e6,
              },
              {
                id: "time",
                header: "Time",
                cell: ({ row }) =>
                  new Date(row.original.EventTime).toLocaleString(),
              },
            ]}
          />
          {error && <p className="form-error">{error}</p>}
        </CardContent>
      </Card>
    </main>
  );
}
