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
  type Meter,
  type Organization,
  type Product,
} from "@/api";
import { RelationCombobox } from "@/components/relation-combobox";
import { DataTable } from "@/components/data-table";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";

export function ProductsPage() {
  const { organization } = useOutletContext<{ organization: Organization }>();
  const { productId } = useParams();
  const navigate = useNavigate();
  const client = organizationApi(organization.id);
  const editor = location.pathname.endsWith("/new") || Boolean(productId);
  const [meters, setMeters] = useState<Meter[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [product, setProduct] = useState<Product | null>(null);
  const [meterID, setMeterID] = useState("");
  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState("");
  useEffect(() => {
    void Promise.all([client.meters(), client.products()])
      .then(([m, p]) => {
        setMeters(m.meters);
        setProducts(p.products);
      })
      .catch((cause) => setError(cause.message));
  }, [organization.id]);
  useEffect(() => {
    if (productId)
      void client
        .product(productId)
        .then(({ product: value }) => {
          setProduct(value);
          setMeterID(value.MeterID);
          setCode(value.Code);
          setName(value.Name);
          setDescription(value.Description);
        })
        .catch((cause) => setError(cause.message));
  }, [productId]);
  async function submit(event: FormEvent) {
    event.preventDefault();
    try {
      const body = {
        meter_id: meterID,
        code,
        name,
        description,
        status: product?.Status ?? "active",
        metadata: {},
      };
      if (productId) await client.updateProduct(productId, body);
      else await client.createProduct(body);
      navigate("..");
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Unable to save product",
      );
    }
  }
  if (editor)
    return (
      <main className="content editor-page">
        <Button variant="ghost" size="sm" asChild><Link to=".."><ArrowLeft />Products</Link></Button>
        <div className="page-head">
          <div>
            <p className="eyebrow">CATALOG / PRODUCTS</p>
            <h1>{product ? product.Name : "Create product"}</h1>
            <p className="muted">
              Connect a billable product to its usage meter.
            </p>
          </div>
        </div>
        <Card>
          <CardContent className="pt-6">
            <form onSubmit={submit}>
              <FieldGroup>
              <Field>
                <FieldLabel>Meter</FieldLabel>
                <RelationCombobox
                  value={meterID}
                  onValueChange={setMeterID}
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
                <FieldLabel>Code</FieldLabel>
                <Input
                  type="text"
                  inputMode="text"
                  pattern="[A-Za-z0-9][A-Za-z0-9._-]*"
                  spellCheck={false}
                  required
                  value={code}
                  onChange={(event) => setCode(event.target.value)}
                />
              </Field>
              <Field>
                <FieldLabel>Name</FieldLabel>
                <Input
                  type="text"
                  inputMode="text"
                  required
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                />
              </Field>
              <Field>
                <FieldLabel>Description</FieldLabel>
                <Input
                  type="text"
                  inputMode="text"
                  value={description}
                  onChange={(event) => setDescription(event.target.value)}
                />
              </Field>
              {error && <p className="form-error">{error}</p>}
              <div className="form-actions">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => navigate("..")}
                >
                  Cancel
                </Button>
                <Button disabled={!meterID}>Save product</Button>
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
          <p className="eyebrow">CATALOG</p>
          <h1>Products</h1>
          <p className="muted">
            Billable capabilities connected to usage meters.
          </p>
        </div>
        <Button asChild>
          <Link to="new">
            <Plus />
            Create product
          </Link>
        </Button>
      </div>
      <Card className="resource-list">
        <CardContent>
          <DataTable
            data={products}
            searchKey="Name"
            searchPlaceholder="Search products…"
            onRowClick={(product) => navigate(product.ID)}
            columns={[
              {
                accessorKey: "Name",
                header: "Name",
                cell: ({ row }) => (
                  <div>
                    <div className="font-medium">{row.original.Name}</div>
                    <div className="text-xs text-muted-foreground">
                      {row.original.Description}
                    </div>
                  </div>
                ),
              },
              { accessorKey: "Code", header: "Code" },
              {
                id: "meter",
                header: "Meter",
                cell: ({ row }) =>
                  meters.find((meter) => meter.ID === row.original.MeterID)
                    ?.Name ?? "—",
              },
              { accessorKey: "Status", header: "Status" },
            ]}
          />
          {error && <p className="form-error">{error}</p>}
        </CardContent>
      </Card>
    </main>
  );
}
