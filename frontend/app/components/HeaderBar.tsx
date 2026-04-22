"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth";

export function HeaderBar() {
  const { user, logout } = useAuth();
  const router = useRouter();

  async function onLogout() {
    await logout();
    router.push("/login");
  }

  return (
    <header className="site">
      <div className="site-inner">
        <div>
          <Link href="/" className="brand">
            🍪 Countrymaam as a Service
          </Link>
          <p className="tagline">
            カントリーマアムを資産として扱う、やや本気のパロディ銀行
          </p>
        </div>
        {user && (
          <div className="site-user">
            <span>
              {user.name} <span className="muted">({user.email})</span>
            </span>
            <button className="ghost light" onClick={onLogout}>
              ログアウト
            </button>
          </div>
        )}
      </div>
    </header>
  );
}
