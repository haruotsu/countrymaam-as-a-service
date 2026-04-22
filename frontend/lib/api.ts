const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE ?? "http://localhost:8080";

async function call<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
    cache: "no-store",
  });
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) {
    const msg = data?.error ?? res.statusText;
    throw new Error(`${res.status} ${msg}`);
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

export const api = {
  listFlavors: () => call<Flavor[]>("/flavors"),
  listUsers: () => call<User[]>("/users/"),
  createUser: (name: string, email: string) =>
    call<User>("/users/", {
      method: "POST",
      body: JSON.stringify({ name, email }),
    }),
  getUser: (id: string) => call<User>(`/users/${id}`),
  listUserAccounts: (userID: string) =>
    call<Account[]>(`/users/${userID}/accounts`),
  openAccount: (user_id: string, flavor: string) =>
    call<Account>("/accounts/", {
      method: "POST",
      body: JSON.stringify({ user_id, flavor }),
    }),
  getAccount: (id: string) => call<Account>(`/accounts/${id}`),
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
