import type { ReactNode } from "react";
export function AuthBrandPanel({
  eyebrow,
  title,
  children,
  status,
}: {
  eyebrow: string;
  title: string;
  children: ReactNode;
  status: string;
}) {
  return (
    <section className="brand-panel">
      <div className="brand">
        <span className="brand-mark">R</span>
        <span>Billing</span>
      </div>
      <div className="brand-copy">
        <p className="eyebrow">{eyebrow}</p>
        <h1>{title}</h1>
        <p>{children}</p>
      </div>
      <div className="signal">
        <span className="signal-dot" />
        {status}
      </div>
    </section>
  );
}
