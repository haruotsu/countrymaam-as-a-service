"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api, type User } from "@/lib/api";

export default function Home() {
  const [users, setUsers] = useState<User[]>([]);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  async function refresh() {
    try {
      const list = await api.listUsers();
      setUsers(list ?? []);
    } catch (e) {
      setErr((e as Error).message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    try {
      await api.createUser(name, email);
      setName("");
      setEmail("");
      await refresh();
    } catch (e) {
      setErr((e as Error).message);
    }
  }

  return (
    <>
      <h2>🧑 ユーザー一覧</h2>

      <section className="panel">
        <form className="row" onSubmit={create}>
          <label>
            名前
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="マーム太郎"
              required
            />
          </label>
          <label>
            メール
            <input
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              type="email"
              placeholder="maam@example.com"
              required
            />
          </label>
          <button type="submit">ユーザー開設</button>
        </form>
        {err && <div className="error">{err}</div>}
      </section>

      <section className="panel">
        {loading ? (
          <p className="muted">読み込み中…</p>
        ) : users.length === 0 ? (
          <p className="muted">
            まだユーザーがいません。上のフォームから 1 人作ってみてください。
          </p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>名前</th>
                <th>メール</th>
                <th>開設日</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.id}>
                  <td>
                    <strong>{u.name}</strong>
                  </td>
                  <td>{u.email}</td>
                  <td className="muted">
                    {new Date(u.created_at).toLocaleString("ja-JP")}
                  </td>
                  <td>
                    <Link href={`/users/${u.id}`}>金庫を見る →</Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      <section className="panel">
        <p className="muted">
          1 マアム = カントリーマアム 1 枚。このサイトは不二家とは関係のないパロディ作品です。
        </p>
      </section>
    </>
  );
}
