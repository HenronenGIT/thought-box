"use client";

import { useSearchParams } from "next/navigation";
import { Suspense } from "react";

const errorMessages: Record<string, string> = {
  not_allowed: "This app is invite-only. Ask the admin for access.",
  state_mismatch: "Sign-in expired. Please try again.",
  state_invalid: "Sign-in expired. Please try again.",
  state_missing: "Sign-in expired. Please try again.",
  email_unverified: "Your Google email is not verified.",
  exchange_failed: "Could not complete sign-in with Google. Please try again.",
  missing_code: "Sign-in was cancelled.",
};

function LoginInner() {
  const params = useSearchParams();
  const error = params.get("error");
  const message = error ? errorMessages[error] ?? "Something went wrong. Please try again." : null;

  return (
    <main className="auth-shell">
      <div className="auth-card">
        <div className="eyebrow">Thought Box</div>
        <h1 className="page-title">Sign in</h1>
        <p className="page-subtitle">
          Thought Box is invite-only. Sign in with the Google account you were
          invited under.
        </p>
        {message ? <div className="auth-error">{message}</div> : null}
        <a className="auth-button" href="/api/auth/google/login">
          Continue with Google
        </a>
      </div>
    </main>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={null}>
      <LoginInner />
    </Suspense>
  );
}
