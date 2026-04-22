CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL,
    email       TEXT        NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS accounts (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    flavor      TEXT        NOT NULL CHECK (flavor IN ('vanilla', 'chocolate', 'matcha')),
    balance     BIGINT      NOT NULL DEFAULT 0 CHECK (balance >= 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, flavor)
);

CREATE INDEX IF NOT EXISTS idx_accounts_user ON accounts(user_id);

CREATE TABLE IF NOT EXISTS transactions (
    id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id               UUID        NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    counterparty_account_id  UUID        REFERENCES accounts(id) ON DELETE SET NULL,
    type                     TEXT        NOT NULL CHECK (type IN (
        'deposit','withdraw','transfer_in','transfer_out','exchange_in','exchange_out'
    )),
    amount                   BIGINT      NOT NULL CHECK (amount > 0),
    memo                     TEXT        NOT NULL DEFAULT '',
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tx_account_created ON transactions(account_id, created_at DESC);
