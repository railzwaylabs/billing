import { Card, CardContent } from "@/components/ui/card";

export function ResourcePage({
  eyebrow,
  title,
  description,
}: {
  eyebrow: string;
  title: string;
  description: string;
}) {
  return (
    <main className="content">
      <div className="page-head">
        <div>
          <p className="eyebrow">{eyebrow}</p>
          <h1>{title}</h1>
          <p className="muted">{description}</p>
        </div>
      </div>
      <Card className="empty-state">
        <CardContent>
          <h2>No data yet</h2>
          <p className="muted">
            The page route is ready. Its resource workflow will use the
            corresponding admin API.
          </p>
        </CardContent>
      </Card>
    </main>
  );
}
