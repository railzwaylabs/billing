import { type FormEvent, useEffect, useState } from "react";
import { ArrowRight } from "lucide-react";
import { api, backendPath } from "@/api";
import type { SessionUser } from "@/App";
import { AuthBrandPanel } from "@/components/auth-brand-panel";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";

export function LoginPage({
  onAuthenticated,
}: {
  onAuthenticated: (user: SessionUser, showPasswordSetup: boolean) => void;
}) {
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("admin");
  const [googleEnabled, setGoogleEnabled] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  useEffect(() => {
    api
      .providers()
      .then((response) =>
        setGoogleEnabled(
          response.providers.some(
            (provider) => provider.id === "google" && provider.enabled,
          ),
        ),
      )
      .catch(() => undefined);
  }, []);
  async function submit(event: FormEvent) {
    event.preventDefault();
    setLoading(true);
    setError("");
    try {
      const result = await api.login(username, password);
      await api.session();
      onAuthenticated(result.user, result.show_password_change);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to sign in");
    } finally {
      setLoading(false);
    }
  }
  return (
    <main className="login-grid">
      <AuthBrandPanel
        eyebrow="USAGE-BASED BILLING"
        title="Turn product usage into clear, defensible invoices."
        status="System ready for setup"
      >
        Configure meters, pricing, subscriptions, and IAM from one focused
        workspace.
      </AuthBrandPanel>
      <section className="login-panel">
        <div className="login-card">
          <p className="eyebrow">ADMIN CONSOLE</p>
          <h2>Welcome back</h2>
          <p className="muted">Sign in to manage your billing workspace.</p>
          <form onSubmit={submit}>
            <FieldGroup>
            <Field>
              <FieldLabel>Username</FieldLabel>
              <Input
                type="text"
                inputMode="text"
                autoComplete="username"
                spellCheck={false}
                autoFocus
                value={username}
                onChange={(event) => setUsername(event.target.value)}
              />
            </Field>
            <Field>
              <FieldLabel>Password</FieldLabel>
              <Input
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
            </Field>
            {error && <p className="form-error">{error}</p>}
            <Button className="w-full" disabled={loading}>
              {loading ? "Signing in…" : "Sign in"}
              <ArrowRight size={16} />
            </Button>
            </FieldGroup>
          </form>
          {googleEnabled && (
            <>
              <div className="separator">
                <span>or</span>
              </div>
              <Button
                variant="outline"
                className="w-full"
                onClick={() =>
                  location.assign(backendPath("/admin/v1/auth/providers/google/login"))
                }
              >
                Continue with Google
              </Button>
            </>
          )}
          <p className="login-foot">
            Local sign-in remains available as emergency access.
          </p>
        </div>
      </section>
    </main>
  );
}
