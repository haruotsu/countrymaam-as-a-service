"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { use, useEffect, useState } from "react";
import {
  api,
  ApiError,
  type Account,
  type Flavor,
  type Transaction,
} from "@/lib/api";
import { useAuth } from "@/lib/auth";

const typeLabels: Record<Transaction["type"], string> = {
  deposit: "預入",
  withdraw: "引出",
  transfer_in: "送金受取",
  transfer_out: "送金",
  exchange_in: "両替受取",
  exchange_out: "両替送出",
};

const signOf = (t: Transaction["type"]): 1 | -1 =>
  t === "deposit" || t === "transfer_in" || t === "exchange_in" ? 1 : -1;

export default function AccountPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const { user, loading } = useAuth();
  const router = useRouter();
  const [account, setAccount] = useState<Account | null>(null);
  const [txs, setTxs] = useState<Transaction[]>([]);
  const [flavors, setFlavors] = useState<Flavor[]>([]);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    if (!loading && !user) router.replace("/login");
  }, [loading, user, router]);

  async function refresh() {
    try {
      const [a, tx, fls] = await Promise.all([
        api.getAccount(id),
        api.listTransactions(id),
        api.listFlavors(),
      ]);
      setAccount(a);
      setTxs(tx ?? []);
      setFlavors(fls ?? []);
    } catch (e) {
      if (e instanceof ApiError && e.status === 403) {
        setErr("この口座はあなたのものではありません。");
      } else {
        setErr((e as Error).message);
      }
    }
  }

  useEffect(() => {
    if (user) refresh();
  }, [user, id]);

  if (loading || !user) return <p className="muted">読み込み中…</p>;

  if (!account) {
    return (
      <>
        <p>
          <Link href="/">← 戻る</Link>
        </p>
        {err ? <div className="error">{err}</div> : <p className="muted">読み込み中…</p>}
      </>
    );
  }

  const label =
    flavors.find((f) => f.key === account.flavor)?.label ?? account.flavor;

  return (
    <>
      <p>
        <Link href="/">← 金庫に戻る</Link>
      </p>

      <section className="panel">
        <div className="muted">
          口座ID <code>{account.id.slice(0, 8)}…</code>
          <span className={`flavor-pill flavor-${account.flavor}`}>
            {account.flavor}
          </span>
        </div>
        <h2 style={{ marginTop: 0 }}>🍪 {label} 口座</h2>
        <div className="balance">
          {account.balance}
          <span className="unit">マアム</span>
        </div>
      </section>

      {err && <div className="error">{err}</div>}

      <CashPanel account={account} onDone={refresh} />
      <TransferPanel account={account} onDone={refresh} />

      <section className="panel">
        <h3 style={{ marginTop: 0 }}>📜 取引履歴</h3>
        {txs.length === 0 ? (
          <p className="muted">まだ取引がありません。</p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>日時</th>
                <th>種別</th>
                <th>金額</th>
                <th>メモ</th>
              </tr>
            </thead>
            <tbody>
              {txs.map((t) => {
                const sign = signOf(t.type);
                return (
                  <tr key={t.id}>
                    <td className="muted">
                      {new Date(t.created_at).toLocaleString("ja-JP")}
                    </td>
                    <td>{typeLabels[t.type]}</td>
                    <td
                      className={
                        sign > 0 ? "amount-positive" : "amount-negative"
                      }
                    >
                      {sign > 0 ? "+" : "-"}
                      {t.amount}
                    </td>
                    <td>{t.memo || <span className="muted">—</span>}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </section>
    </>
  );
}

function CashPanel({
  account,
  onDone,
}: {
  account: Account;
  onDone: () => void;
}) {
  const [amount, setAmount] = useState("1");
  const [memo, setMemo] = useState("");
  const [err, setErr] = useState<string | null>(null);

  async function deposit() {
    setErr(null);
    try {
      await api.deposit(account.id, Number(amount), memo);
      setAmount("1");
      setMemo("");
      onDone();
    } catch (e) {
      setErr((e as Error).message);
    }
  }
  async function withdraw() {
    setErr(null);
    try {
      await api.withdraw(account.id, Number(amount), memo);
      setAmount("1");
      setMemo("");
      onDone();
    } catch (e) {
      setErr((e as Error).message);
    }
  }

  return (
    <section className="panel">
      <h3 style={{ marginTop: 0 }}>💰 預入・引出</h3>
      <div className="row">
        <label>
          金額（マアム）
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
        <button onClick={deposit}>預け入れ</button>
        <button className="ghost" onClick={withdraw}>
          引き出し
        </button>
      </div>
      {err && <div className="error">{err}</div>}
    </section>
  );
}

function TransferPanel({
  account,
  onDone,
}: {
  account: Account;
  onDone: () => void;
}) {
  const [email, setEmail] = useState("");
  const [amount, setAmount] = useState("1");
  const [memo, setMemo] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const [ok, setOk] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    setOk(null);
    try {
      // まず宛先口座を解決（同じフレーバー）
      const res = await api.searchAccount(email, account.flavor);
      await api.transfer(account.id, res.account.id, Number(amount), memo);
      setOk(
        `${res.user_name} さんに ${amount} マアムを送金しました（${res.user_email}）`,
      );
      setEmail("");
      setAmount("1");
      setMemo("");
      onDone();
    } catch (e) {
      setErr((e as Error).message);
    }
  }

  return (
    <section className="panel">
      <h3 style={{ marginTop: 0 }}>✉️ 送金</h3>
      <p className="muted">
        宛先のメールアドレスを入れると、同じフレーバー（{account.flavor}）の口座を自動で探して送金します。
      </p>
      <form className="row" onSubmit={submit}>
        <label>
          宛先メール
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            placeholder="friend@example.com"
          />
        </label>
        <label>
          金額（マアム）
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
        <button type="submit">送金</button>
      </form>
      {err && <div className="error">{err}</div>}
      {ok && <div className="ok-msg">✅ {ok}</div>}
    </section>
  );
}
