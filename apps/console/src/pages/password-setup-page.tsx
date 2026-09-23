import { type FormEvent, useState } from "react";
import { ArrowRight, KeyRound } from "lucide-react";
import { api } from "@/api";
import { AuthBrandPanel } from "@/components/auth-brand-panel";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";

export function PasswordSetupPage({ onComplete }: { onComplete: () => void }) {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState("");
  async function change(event: FormEvent) {
    event.preventDefault();
    if (next !== confirm) {
      setError("New passwords do not match");
      return;
    }
    try {
      await api.changePassword(current, next);
      onComplete();
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Unable to change password",
      );
    }
  }
  async function skip() {
    try {
      await api.skipPassword();
      onComplete();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to continue");
    }
  }
  return (
    <main className="login-grid">
      <AuthBrandPanel
        eyebrow="ACCOUNT SECURITY"
        title="One last step before your workspace."
        status="Secure session established"
      >
        Replace the initial administrator password, or skip this once and
        continue setting up billing.
      </AuthBrandPanel>
      <section className="login-panel">
        <div className="login-card">
          <div className="modal-icon">
            <KeyRound size={20} />
          </div>
          <p className="eyebrow password-eyebrow">INITIAL PASSWORD</p>
          <h2>Secure your admin account</h2>
          <p className="muted">
            Change the initial password before inviting your team.
          </p>
          <form onSubmit={change}>
            <FieldGroup>
              <Field>
                <FieldLabel hint="Enter the password used for this login session.">
                  Current password
                </FieldLabel>
                <Input
                  autoFocus
                  type="password"
                  autoComplete="current-password"
                  placeholder="Current password"
                  value={current}
                  onChange={(event) => setCurrent(event.target.value)}
                />
              </Field>
              <Field>
                <FieldLabel hint="Use at least 12 characters and avoid reused passwords.">
                  New password
                </FieldLabel>
                <Input
                  type="password"
                  autoComplete="new-password"
                  value={next}
                  onChange={(event) => setNext(event.target.value)}
                  placeholder="At least 12 characters"
                />
              </Field>
              <Field>
                <FieldLabel hint="Repeat the new password exactly.">
                  Confirm password
                </FieldLabel>
                <Input
                  type="password"
                  autoComplete="new-password"
                  placeholder="Repeat the new password"
                  value={confirm}
                  onChange={(event) => setConfirm(event.target.value)}
                />
              </Field>
              {error && <p className="form-error">{error}</p>}
              <Button className="w-full" disabled={next.length < 12}>
                Save password
                <ArrowRight size={16} />
              </Button>
            </FieldGroup>
          </form>
          <Button className="w-full skip-button" variant="ghost" onClick={skip}>
            Skip for now
          </Button>
        </div>
      </section>
    </main>
  );
}
