import { type FormEvent, useEffect, useState } from "react";
import { ArrowLeft, Plus } from "lucide-react";
import {
  Link,
  useNavigate,
  useOutletContext,
  useParams,
} from "react-router-dom";
import { organizationApi, type Organization, type Product } from "@/api";
import { DataTable } from "@/components/data-table";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { useCursorPagination } from "@/hooks/use-cursor-pagination";

export function ProductsPage() {
  const { organization } = useOutletContext<{ organization: Organization }>();
  const { productId } = useParams();
  const navigate = useNavigate();
  const client = organizationApi(organization.id);
  const listPath = `/organizations/${organization.id}/catalog/products`;
  const editor = location.pathname.endsWith("/new") || Boolean(productId);
  const [products, setProducts] = useState<Product[]>([]);
  const [product, setProduct] = useState<Product | null>(null);
  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState("");
  const pagination = useCursorPagination();
  useEffect(() => {
    void client
      .products(pagination.request)
      .then((result) => {
        setProducts(result.products);
        pagination.setPageInfo(result.page_info);
      })
      .catch((cause) => setError(cause.message));
  }, [organization.id, pagination.cursor]);
  useEffect(() => {
    if (productId)
      void client
        .product(productId)
        .then(({ product: value }) => {
          setProduct(value);
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
        code,
        name,
        description,
        status: product?.Status ?? "active",
        metadata: {},
      };
      if (productId) await client.updateProduct(productId, body);
      else await client.createProduct(body);
      navigate(listPath, { replace: true });
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Unable to save product",
      );
    }
  }
  if (editor)
    return (
      <main className="content editor-page">
        <Button variant="ghost" size="sm" asChild>
          <Link to={listPath}>
            <ArrowLeft />
            Products
          </Link>
        </Button>
        <div className="page-head">
          <div>
            <p className="eyebrow">CATALOG / PRODUCTS</p>
            <h1>{product ? product.Name : "Create product"}</h1>
            <p className="muted">
              A product groups one or more independently metered price charges.
            </p>
          </div>
        </div>
        <Card>
          <CardContent className="pt-6">
            <form onSubmit={submit}>
              <FieldGroup>
                <Field>
                  <FieldLabel hint="Stable machine identifier for this catalog product.">
                    Code
                  </FieldLabel>
                  <Input
                    type="text"
                    inputMode="text"
                    pattern="[A-Za-z0-9][A-Za-z0-9._\-]*"
                    spellCheck={false}
                    required
                    placeholder="compute_engine"
                    value={code}
                    onChange={(event) => setCode(event.target.value)}
                  />
                </Field>
                <Field>
                  <FieldLabel hint="Product name shown to operators and on invoice lines.">
                    Name
                  </FieldLabel>
                  <Input
                    type="text"
                    inputMode="text"
                    required
                    placeholder="Compute Engine"
                    value={name}
                    onChange={(event) => setName(event.target.value)}
                  />
                </Field>
                <Field>
                  <FieldLabel hint="Short explanation of what this product provides.">
                    Description
                  </FieldLabel>
                  <Input
                    type="text"
                    inputMode="text"
                    placeholder="Usage-based compute product"
                    value={description}
                    onChange={(event) => setDescription(event.target.value)}
                  />
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
                  <Button disabled={!code || !name}>Save product</Button>
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
            Billable capabilities that own versioned prices and usage charges.
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
            cursorPagination={{
              cursor: pagination.cursor,
              pageInfo: pagination.pageInfo,
              onCursorChange: pagination.setCursor,
            }}
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
              { accessorKey: "Status", header: "Status" },
            ]}
          />
          {error && <p className="form-error">{error}</p>}
        </CardContent>
      </Card>
    </main>
  );
}
