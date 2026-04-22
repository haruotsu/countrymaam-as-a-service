# Countrymaam as a Service (CMaaS)

> 🍪 カントリーマアムを資産として扱う、やや本気のパロディ銀行。

- コンセプト・ドメイン仕様は [`CONCEPT.md`](./CONCEPT.md) を参照
- テストリスト（TDD）は [`TEST_LIST.md`](./TEST_LIST.md) を参照

## アーキテクチャ

```
┌────────────────────┐   HTTP   ┌────────────────────┐   SQL   ┌──────────────┐
│  Next.js (App)     │ ─────→   │  Go (chi) API      │ ─────→  │  PostgreSQL  │
│  localhost:3900    │          │  localhost:8090    │         │   :5432      │
└────────────────────┘          └────────────────────┘         └──────────────┘
```

- **backend/**: Go + chi + pgx。ドメイン / サービス / リポジトリ / HTTPハンドラの 4 層
- **frontend/**: Next.js 15 App Router、TypeScript、プレーンCSS（クッキー色）
- **PostgreSQL 16**: `accounts.balance >= 0` を CHECK 制約でDB側でも守る

## 動かし方

```bash
docker compose up -d --build
```

起動後:

- 🌐 **フロント**: http://localhost:3900
- 🔧 **API**: http://localhost:8090 （`GET /healthz` で疎通確認）
- 🛢 **DB**: `postgres://cmaas:cmaas@localhost:5432/cmaas`

停止:

```bash
docker compose down           # データは残す
docker compose down -v        # ボリュームごと削除（初期化したい時）
```

## エンドポイント

| メソッド | パス                              | 概要                     |
| -------- | --------------------------------- | ------------------------ |
| GET      | `/healthz`                        | ヘルスチェック           |
| GET      | `/flavors`                        | フレーバー一覧           |
| POST     | `/users/`                         | ユーザー登録             |
| GET      | `/users/`                         | ユーザー一覧             |
| GET      | `/users/{id}`                     | ユーザー取得             |
| GET      | `/users/{id}/accounts`            | ユーザーの口座一覧       |
| POST     | `/accounts/`                      | 口座開設                 |
| GET      | `/accounts/{id}`                  | 口座取得                 |
| POST     | `/accounts/{id}/deposit`          | 預け入れ                 |
| POST     | `/accounts/{id}/withdraw`         | 引き出し                 |
| GET      | `/accounts/{id}/transactions`     | 取引履歴                 |
| POST     | `/transfers`                      | 送金（同フレーバー）     |
| POST     | `/exchanges`                      | 両替（同ユーザー内）     |

### 例: まっさらな状態から口座を作って両替する

```bash
A=$(curl -sf -X POST http://localhost:8090/users/ \
      -H 'Content-Type: application/json' \
      -d '{"name":"Bob","email":"bob@example.com"}' | jq -r .id)

AV=$(curl -sf -X POST http://localhost:8090/accounts/ \
      -H 'Content-Type: application/json' \
      -d "{\"user_id\":\"$A\",\"flavor\":\"vanilla\"}" | jq -r .id)

AC=$(curl -sf -X POST http://localhost:8090/accounts/ \
      -H 'Content-Type: application/json' \
      -d "{\"user_id\":\"$A\",\"flavor\":\"chocolate\"}" | jq -r .id)

curl -sf -X POST http://localhost:8090/accounts/$AV/deposit \
      -H 'Content-Type: application/json' \
      -d '{"amount":10,"memo":"はじめての一枚"}'

curl -sf -X POST http://localhost:8090/exchanges \
      -H 'Content-Type: application/json' \
      -d "{\"from_account_id\":\"$AV\",\"to_account_id\":\"$AC\",\"amount\":10}"
# => {"from_amount":10,"to_amount":8}
```

## テスト

ドメイン / サービス / HTTPハンドラ は DB 不要のユニットテスト。

```bash
cd backend
go test ./...
```

リポジトリ層だけ実 Postgres が必要で、`TEST_DATABASE_URL` が無ければ skip される。

```bash
docker compose up -d db
TEST_DATABASE_URL=postgres://cmaas:cmaas@localhost:5432/cmaas?sslmode=disable \
  go test ./internal/repository/ -v
```

## 開発スタイル

t_wada スタイル TDD（Red → Green → Refactor）。コミット前に必ず `go test ./...` を実行する。

## 免責

不二家および「カントリーマアム®」は株式会社不二家の登録商標・商品名です。本プロジェクトは**非公式のパロディ作品**であり、不二家とは一切関係がありません。
