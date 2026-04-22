"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

export default function SignupPage() {
  const router = useRouter();
  const { refresh } = useAuth();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    setSubmitting(true);
    try {
      await api.register(name, email, password);
      await refresh();
      router.push("/");
    } catch (e) {
      setErr((e as Error).message);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="auth-card panel">
      <h2 style={{ marginTop: 0 }}>🍪 新規口座開設</h2>
      <form className="row-stack" onSubmit={onSubmit}>
        <label>
          お名前
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            placeholder="マーム太郎"
            autoComplete="name"
          />
        </label>
        <label>
          メール
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            autoComplete="email"
          />
        </label>
        <label>
          パスワード <span className="muted">（8文字以上）</span>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            minLength={8}
            required
            autoComplete="new-password"
          />
        </label>
        {err && <div className="error">{err}</div>}
        <button type="submit" disabled={submitting}>
          {submitting ? "登録中…" : "登録"}
        </button>
      </form>
      <p className="muted" style={{ marginTop: "1rem" }}>
        すでにアカウントがある？ <Link href="/login">ログイン</Link>
      </p>
    </div>
  );
}
