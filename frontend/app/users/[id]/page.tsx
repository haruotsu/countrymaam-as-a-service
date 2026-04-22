"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { use } from "react";
import {
  api,
  type Account,
  type Flavor,
  type User,
} from "@/lib/api";

export default function UserPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const [user, setUser] = useState<User | null>(null);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [flavors, setFlavors] = useState<Flavor[]>([]);
  const [selectedFlavor, setSelectedFlavor] = useState<string>("");
  const [err, setErr] = useState<string | null>(null);

  async function refresh() {
    try {
      const [u, accs, fls] = await Promise.all([
        api.getUser(id),
        api.listUserAccounts(id),
        api.listFlavors(),
      ]);
      setUser(u);
      setAccounts(accs ?? []);
      setFlavors(fls ?? []);
      const missing = (fls ?? []).find(
        (f) => !(accs ?? []).some((a) => a.flavor === f.key),
      );
      setSelectedFlavor(missing?.key ?? "");
    } catch (e) {
      setErr((e as Error).message);
    }
  }

  useEffect(() => {
    refresh();
  }, [id]);

  async function open(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    try {
      await api.openAccount(id, selectedFlavor);
      await refresh();
    } catch (e) {
      setErr((e as Error).message);
    }
  }

  const availableFlavors = flavors.filter(
    (f) => !accounts.some((a) => a.flavor === f.key),
  );
  const totalByFlavor: Record<string, number> = {};
  accounts.forEach((a) => (totalByFlavor[a.flavor] = a.balance));

  return (
    <>
      <p>
        <Link href="/">← ユーザー一覧に戻る</Link>
      </p>
      <h2>🏦 {user?.name ?? "…"} さんの金庫</h2>
      <p className="muted">{user?.email}</p>

      {err && <div className="error">{err}</div>}

      <section className="panel">
        <h3 style={{ marginTop: 0 }}>口座一覧</h3>
        {accounts.length === 0 ? (
          <p className="muted">まだ口座がありません。下で開設してください。</p>
        ) : (
          <div className="row" style={{ alignItems: "stretch" }}>
            {accounts.map((a) => (
              <div
                key={a.id}
                className="panel"
                style={{ flex: "1 1 220px", margin: 0 }}
              >
                <div className="muted" style={{ fontSize: "0.75rem" }}>
                  {flavorLabel(a.flavor, flavors)}
                  <span className={`flavor-pill flavor-${a.flavor}`}>
                    {a.flavor}
                  </span>
                </div>
                <div className="balance">
                  {a.balance}
                  <span className="unit">マアム</span>
                </div>
                <Link href={`/accounts/${a.id}`}>
                  取引する →
                </Link>
              </div>
            ))}
          </div>
        )}
      </section>

      {availableFlavors.length > 0 && (
        <section className="panel">
          <h3 style={{ marginTop: 0 }}>新しい口座を開く</h3>
          <form className="row" onSubmit={open}>
            <label>
              フレーバー
              <select
                value={selectedFlavor}
                onChange={(e) => setSelectedFlavor(e.target.value)}
                required
              >
                <option value="">選択</option>
                {availableFlavors.map((f) => (
                  <option key={f.key} value={f.key}>
                    {f.label} ({f.key}) ・ レート {f.rate}
                  </option>
                ))}
              </select>
            </label>
            <button type="submit">口座開設</button>
          </form>
        </section>
      )}

      {accounts.length >= 2 && (
        <ExchangePanel
          accounts={accounts}
          flavors={flavors}
          onDone={refresh}
        />
      )}
    </>
  );
}

function flavorLabel(key: string, flavors: Flavor[]) {
  return flavors.find((f) => f.key === key)?.label ?? key;
}

function ExchangePanel({
  accounts,
  flavors,
  onDone,
}: {
  accounts: Account[];
  flavors: Flavor[];
  onDone: () => void;
}) {
  const [from, setFrom] = useState<string>(accounts[0]?.id ?? "");
  const [to, setTo] = useState<string>(accounts[1]?.id ?? "");
  const [amount, setAmount] = useState<string>("1");
  const [memo, setMemo] = useState<string>("");
  const [err, setErr] = useState<string | null>(null);
  const [result, setResult] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    setResult(null);
    try {
      const res = await api.exchange(from, to, Number(amount), memo);
      setResult(`${res.from_amount} マアム → ${res.to_amount} マアムに両替しました`);
      onDone();
    } catch (e) {
      setErr((e as Error).message);
    }
  }

  return (
    <section className="panel">
      <h3 style={{ marginTop: 0 }}>🔁 両替（自分の口座間）</h3>
      <form className="row" onSubmit={submit}>
        <label>
          元
          <select value={from} onChange={(e) => setFrom(e.target.value)}>
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>
                {flavorLabel(a.flavor, flavors)} ({a.balance} マアム)
              </option>
            ))}
          </select>
        </label>
        <label>
          先
          <select value={to} onChange={(e) => setTo(e.target.value)}>
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>
                {flavorLabel(a.flavor, flavors)} ({a.balance} マアム)
              </option>
            ))}
          </select>
        </label>
        <label>
          元の量
          <input
            type="number"
            min={1}
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
        </label>
        <label>
          メモ
          <input value={memo} onChange={(e) => setMemo(e.target.value)} />
        </label>
        <button type="submit">両替</button>
      </form>
      {err && <div className="error">{err}</div>}
      {result && <div className="muted">✅ {result}</div>}
    </section>
  );
}
