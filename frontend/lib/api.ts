const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE ?? "http://localhost:8080";

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
  }
}

async function call<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
    credentials: "include", // セッションCookieを送受信
    cache: "no-store",
  });
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) {
    const msg = data?.error ?? res.statusText;
    throw new ApiError(res.status, msg);
  }
  return data as T;
}

export type User = {
  id: string;
  name: string;
  email: string;
  created_at: string;
};

export type Account = {
  id: string;
  user_id: string;
  flavor: "vanilla" | "chocolate" | "matcha";
  balance: number;
  created_at: string;
};

export type Flavor = {
  key: "vanilla" | "chocolate" | "matcha";
  label: string;
  rate: string;
};

export type Transaction = {
  id: string;
  account_id: string;
  counterparty_account_id: string | null;
  type:
    | "deposit"
    | "withdraw"
    | "transfer_in"
    | "transfer_out"
    | "exchange_in"
    | "exchange_out";
  amount: number;
  memo: string;
  created_at: string;
};

export type AccountSearch = {
  account: Account;
  user_name: string;
  user_email: string;
};

export const api = {
  // auth
  register: (name: string, email: string, password: string) =>
    call<User>("/auth/register", {
      method: "POST",
      body: JSON.stringify({ name, email, password }),
    }),
  login: (email: string, password: string) =>
    call<User>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }),
  logout: () => call<{ status: string }>("/auth/logout", { method: "POST" }),
  me: () => call<User>("/auth/me"),

  // reference
  listFlavors: () => call<Flavor[]>("/flavors"),

  // accounts
  listMyAccounts: () => call<Account[]>("/accounts/me"),
  openAccount: (flavor: string) =>
    call<Account>("/accounts", {
      method: "POST",
      body: JSON.stringify({ flavor }),
    }),
  getAccount: (id: string) => call<Account>(`/accounts/${id}`),
  searchAccount: (email: string, flavor: string) =>
    call<AccountSearch>(
      `/accounts/search?email=${encodeURIComponent(email)}&flavor=${flavor}`,
    ),
  deposit: (id: string, amount: number, memo: string) =>
    call<Account>(`/accounts/${id}/deposit`, {
      method: "POST",
      body: JSON.stringify({ amount, memo }),
    }),
  withdraw: (id: string, amount: number, memo: string) =>
    call<Account>(`/accounts/${id}/withdraw`, {
      method: "POST",
      body: JSON.stringify({ amount, memo }),
    }),
  transfer: (
    from_account_id: string,
    to_account_id: string,
    amount: number,
    memo: string,
  ) =>
    call<{ status: string }>("/transfers", {
      method: "POST",
      body: JSON.stringify({ from_account_id, to_account_id, amount, memo }),
    }),
  exchange: (
    from_account_id: string,
    to_account_id: string,
    amount: number,
    memo: string,
  ) =>
    call<{ from_amount: number; to_amount: number }>("/exchanges", {
      method: "POST",
      body: JSON.stringify({ from_account_id, to_account_id, amount, memo }),
    }),
  listTransactions: (id: string, limit = 100) =>
    call<Transaction[]>(`/accounts/${id}/transactions?limit=${limit}`),
};
