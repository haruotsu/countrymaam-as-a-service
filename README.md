# Countrymaam as a Service (CMaaS)

> 🍪 カントリーマアムを資産として扱う、やや本気のパロディ銀行。

- コンセプト・ドメイン仕様は [`CONCEPT.md`](./CONCEPT.md) を参照
- テストリスト（TDD）は [`TEST_LIST.md`](./TEST_LIST.md) を参照
- 本番へのデプロイ手順は [`DEPLOY.md`](./DEPLOY.md) を参照

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

認証は Cookie セッション (`cmaas_session`)。**🔒** は要ログイン。

| メソッド | パス                              | 概要                     |
| -------- | --------------------------------- | ------------------------ |
| GET      | `/healthz`                        | ヘルスチェック           |
| GET      | `/flavors`                        | フレーバー一覧           |
| POST     | `/auth/register`                  | ユーザー登録（+自動ログイン） |
| POST     | `/auth/login`                     | ログイン                 |
| POST     | `/auth/logout`                    | ログアウト               |
| GET 🔒   | `/auth/me`                        | 自分のプロフィール       |
| POST 🔒  | `/accounts`                       | 口座開設（自分名義）     |
| GET 🔒   | `/accounts/me`                    | 自分の口座一覧           |
| GET 🔒   | `/accounts/search?email=&flavor=` | メールで他ユーザーの口座を検索（送金用） |
| GET 🔒   | `/accounts/{id}`                  | 口座取得（本人のみ）     |
| POST 🔒  | `/accounts/{id}/deposit`          | 預け入れ（本人のみ）     |
| POST 🔒  | `/accounts/{id}/withdraw`         | 引き出し（本人のみ）     |
| GET 🔒   | `/accounts/{id}/transactions`     | 取引履歴（本人のみ）     |
| POST 🔒  | `/transfers`                      | 送金（from は自分の口座） |
| POST 🔒  | `/exchanges`                      | 両替（自分の2口座間）    |

### 例: 登録して口座を作って両替する

```bash
# cookie jar を使ってセッションを維持する
curl -sfc /tmp/jar -X POST http://localhost:8090/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Bob","email":"bob@example.com","password":"password-bob"}'

AV=$(curl -sfb /tmp/jar -X POST http://localhost:8090/accounts \
       -H 'Content-Type: application/json' -d '{"flavor":"vanilla"}' | jq -r .id)

AC=$(curl -sfb /tmp/jar -X POST http://localhost:8090/accounts \
       -H 'Content-Type: application/json' -d '{"flavor":"chocolate"}' | jq -r .id)

curl -sfb /tmp/jar -X POST http://localhost:8090/accounts/$AV/deposit \
  -H 'Content-Type: application/json' -d '{"amount":10,"memo":"はじめての一枚"}'

curl -sfb /tmp/jar -X POST http://localhost:8090/exchanges \
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
