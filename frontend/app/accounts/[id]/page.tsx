"use client";

import { useEffect, useState, use } from "react";
import Link from "next/link";
import {
  api,
  type Account,
  type Flavor,
  type Transaction,
  type User,
} from "@/lib/api";

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
  const [account, setAccount] = useState<Account | null>(null);
  const [user, setUser] = useState<User | null>(null);
  const [txs, setTxs] = useState<Transaction[]>([]);
  const [flavors, setFlavors] = useState<Flavor[]>([]);
  const [sameFlavorAccounts, setSameFlavorAccounts] = useState<Account[]>([]);
  const [err, setErr] = useState<string | null>(null);

  async function refresh() {
    try {
      const a = await api.getAccount(id);
      setAccount(a);
      const [u, tx, fls] = await Promise.all([
        api.getUser(a.user_id),
        api.listTransactions(id),
        api.listFlavors(),
      ]);
      setUser(u);
      setTxs(tx ?? []);
      setFlavors(fls ?? []);

      // 送金先候補: 他ユーザーの同フレーバー口座一覧。
      // 今はユーザー一覧 × 口座一覧 で愚直に拾う。
      const users = await api.listUsers();
      const candidates: Account[] = [];
      for (const other of users) {
        if (other.id === a.user_id) continue;
        const accs = await api.listUserAccounts(other.id);
        accs.forEach((x) => {
          if (x.flavor === a.flavor) candidates.push(x);
        });
      }
      setSameFlavorAccounts(candidates);
    } catch (e) {
      setErr((e as Error).message);
    }
  }

  useEffect(() => {
    refresh();
  }, [id]);

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
        <Link href={`/users/${account.user_id}`}>← {user?.name ?? ""} さんの金庫へ</Link>
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

      <TransferPanel
        account={account}
        candidates={sameFlavorAccounts}
        onDone={refresh}
      />

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
  candidates,
  onDone,
}: {
  account: Account;
  candidates: Account[];
  onDone: () => void;
}) {
  const [toID, setToID] = useState(candidates[0]?.id ?? "");
  const [amount, setAmount] = useState("1");
  const [memo, setMemo] = useState("");
  const [err, setErr] = useState<string | null>(null);

  // candidates 変更時に初期値を合わせる
  useEffect(() => {
    if (!toID && candidates[0]) setToID(candidates[0].id);
  }, [candidates, toID]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    try {
      await api.transfer(account.id, toID, Number(amount), memo);
      setAmount("1");
      setMemo("");
      onDone();
    } catch (e) {
      setErr((e as Error).message);
    }
  }

  return (
    <section className="panel">
      <h3 style={{ marginTop: 0 }}>✉️ 送金（同じフレーバー間）</h3>
      {candidates.length === 0 ? (
        <p className="muted">
          送金先の候補がいません（同じ {account.flavor} 口座を持つ別ユーザーが必要）。
        </p>
      ) : (
        <form className="row" onSubmit={submit}>
          <label>
            送り先
            <select value={toID} onChange={(e) => setToID(e.target.value)}>
              {candidates.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.id.slice(0, 8)}… ({c.balance} マアム保有)
                </option>
              ))}
            </select>
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
      )}
      {err && <div className="error">{err}</div>}
    </section>
  );
}
