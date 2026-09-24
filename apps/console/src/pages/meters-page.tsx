import { type FormEvent, useEffect, useState } from "react";
import { ArrowLeft, Plus } from "lucide-react";
import {
  Link,
  useNavigate,
  useOutletContext,
  useParams,
} from "react-router-dom";
import {
  organizationApi,
  type MeasurementUnit,
  type Meter,
  type Organization,
} from "@/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { UnitCombobox } from "@/components/unit-combobox";
import { DataTable } from "@/components/data-table";
import { useCursorPagination } from "@/hooks/use-cursor-pagination";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

export function MetersPage() {
  const { organization } = useOutletContext<{ organization: Organization }>();
  const { meterId } = useParams();
  const navigate = useNavigate();
  const client = organizationApi(organization.id);
  const listPath = `/organizations/${organization.id}/meters`;
  const isEditor = location.pathname.endsWith("/new") || Boolean(meterId);
  const [values, setValues] = useState<Meter[]>([]);
  const [units, setUnits] = useState<MeasurementUnit[]>([]);
  const [current, setCurrent] = useState<Meter | null>(null);
  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [unit, setUnit] = useState("");
  const [aggregation, setAggregation] = useState<"count" | "sum">("sum");
  const [error, setError] = useState("");
  const pagination = useCursorPagination();
  useEffect(() => {
    void client
      .measurementUnits()
      .then((result) => setUnits(result.measurement_units))
      .catch((cause) => setError(cause.message));
  }, [organization.id]);
  useEffect(() => {
    if (!isEditor)
      void client
        .meters(pagination.request)
        .then((result) => {
          setValues(result.meters);
          pagination.setPageInfo(result.page_info);
        })
        .catch((cause) => setError(cause.message));
  }, [organization.id, isEditor, pagination.cursor]);
  useEffect(() => {
    if (meterId)
      void client
        .meter(meterId)
        .then(({ meter }) => {
          setCurrent(meter);
          setCode(meter.Code);
          setName(meter.Name);
          setUnit(meter.Unit);
          setAggregation(meter.Aggregation);
        })
        .catch((cause) => setError(cause.message));
  }, [meterId]);
  async function submit(event: FormEvent) {
    event.preventDefault();
    try {
      const body = {
        Code: code,
        Name: name,
        Unit: unit,
        Aggregation: aggregation,
      };
      if (current) await client.updateMeter(current.ID, body);
      else await client.createMeter(body);
      navigate(listPath, { replace: true });
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to save meter");
    }
  }
  if (isEditor)
    return (
      <main className="content editor-page">
        <Button variant="ghost" size="sm" asChild>
          <Link to={listPath}>
            <ArrowLeft />
            Meters
          </Link>
        </Button>
        <div className="page-head">
          <div>
            <p className="eyebrow">METERING</p>
            <h1>{current ? current.Name : "Create meter"}</h1>
            <p className="muted">
              Define the event aggregation and unit used by products.
            </p>
          </div>
        </div>
        <Card>
          <CardContent className="pt-6">
            <form onSubmit={submit}>
              <FieldGroup>
                <Field>
                  <FieldLabel hint="Stable machine identifier sent by usage producers, for example api_requests.">
                    Code
                  </FieldLabel>
                  <Input
                    type="text"
                    inputMode="text"
                    pattern="[A-Za-z0-9][A-Za-z0-9._\-]*"
                    spellCheck={false}
                    required
                    placeholder="api_requests"
                    value={code}
                    onChange={(event) => setCode(event.target.value)}
                  />
                </Field>
                <Field>
                  <FieldLabel hint="Human-readable meter name shown in the catalog and reports.">
                    Name
                  </FieldLabel>
                  <Input
                    type="text"
                    inputMode="text"
                    required
                    placeholder="API requests"
                    value={name}
                    onChange={(event) => setName(event.target.value)}
                  />
                </Field>
                <Field>
                  <FieldLabel hint="Unit measured by each event value, selected from the reference catalog.">
                    Usage value unit
                  </FieldLabel>
                  <UnitCombobox
                    value={unit}
                    onValueChange={setUnit}
                    options={units.map((item) => ({
                      value: item.Code,
                      label: `${item.Name}${item.Symbol ? ` (${item.Symbol})` : ""}`,
                      description: `${item.Category} · ${item.Description}`,
                    }))}
                  />
                  <FieldDescription>
                    What does each event value measure? This does not need to
                    match the meter code.
                  </FieldDescription>
                </Field>
                {code && unit && (
                  <div className="contract-preview">
                    <strong>Usage contract</strong>
                    <span>
                      <code>{code}</code>: a value of <code>10</code> means{" "}
                      <code>10 {unit}s</code>.
                    </span>
                  </div>
                )}
                <Field>
                  <FieldLabel hint="Method used to combine event values during a billing period.">
                    Aggregation
                  </FieldLabel>
                  <Select
                    value={aggregation}
                    onValueChange={(value) =>
                      setAggregation(value as "count" | "sum")
                    }
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="sum">Sum values</SelectItem>
                      <SelectItem value="count">Count events</SelectItem>
                    </SelectContent>
                  </Select>
                </Field>
                {error && <p className="form-error">{error}</p>}
                <div className="form-actions">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => navigate(listPath)}
                  >
                    Cancel
                  </Button>
                  <Button disabled={!unit}>Save meter</Button>
                </div>
              </FieldGroup>
            </form>
          </CardContent>
        </Card>
      </main>
    );
  return (
    <main className="content">
      <div className="page-head">
        <div>
          <p className="eyebrow">METERING</p>
          <h1>Meters</h1>
          <p className="muted">
            Define how incoming usage is measured and aggregated.
          </p>
        </div>
        <Button asChild>
          <Link to="new">
            <Plus />
            Create meter
          </Link>
        </Button>
      </div>
      <Card className="resource-list">
        <CardContent>
          <DataTable
            data={values}
            cursorPagination={{
              cursor: pagination.cursor,
              pageInfo: pagination.pageInfo,
              onCursorChange: pagination.setCursor,
            }}
            searchKey="Name"
            searchPlaceholder="Search meters…"
            onRowClick={(meter) => navigate(meter.ID)}
            columns={[
              {
                accessorKey: "Name",
                header: "Name",
                cell: ({ row }) => (
                  <span className="font-medium">{row.original.Name}</span>
                ),
              },
              { accessorKey: "Code", header: "Code" },
              { accessorKey: "Aggregation", header: "Aggregation" },
              { accessorKey: "Unit", header: "Unit" },
            ]}
          />
          {error && <p className="form-error">{error}</p>}
        </CardContent>
      </Card>
    </main>
  );
}
