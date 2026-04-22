# デプロイ手順

このアプリは **Postgres / Goバックエンド / Next.jsフロントエンド** の3コンポーネントで構成されています。
ローカルでは `docker compose up` 1本で動きますが、本番に出すときは「どこに置くか」で段取りが変わります。

まず本番公開前の **宿題（= 必須の潰し込み）** と、現実的な **デプロイ先の3択** を載せます。

---

## ⚠️ 本番公開前の宿題

このリポジトリは MVP ＋パロディ目的のため、以下が **意図的に未実装** です。インターネットに出す前に潰してください。

| 項目                   | 現状                                                      | やること                                       |
| ---------------------- | --------------------------------------------------------- | ---------------------------------------------- |
| 認証・認可             | ✅ Cookie セッション (bcrypt + sessions) + 所有権チェック | パスワードリセット / 2FA / OIDC 連携は未        |
| CORS                   | `ALLOWED_ORIGIN` 環境変数（無ければ `http://localhost:3900`） | 本番ドメインに差し替え                        |
| レート制限             | 無し                                                      | chi の `httprate` などで /auth/login, /withdraw を絞る |
| Cookie Secure          | `SESSION_COOKIE_SECURE=1` で有効化                        | 本番では 1 を必ず立てる（HTTPS 前提）          |
| シークレット管理       | `docker-compose.yml` に `POSTGRES_PASSWORD=cmaas` が平文  | 本番では環境変数 / シークレットストアへ       |
| HTTPS                  | 生HTTPで動く                                              | リバースプロキシ (Caddy/Nginx) か PaaS のTLSで |
| マイグレーション運用   | `docker-entrypoint-initdb.d` で 001/002 を流す。既存DBには `psql` で 002 を手動適用 | `golang-migrate` / `atlas` を CI から叩く      |
| DB バックアップ        | 無し                                                      | 自動スナップショットを有効化                   |
| 監査ログ               | 取引履歴は DB にあるがアプリログは stdout のみ           | ログ集約 (Loki / CloudWatch) へ流す            |

---

## 選択肢

| 選択肢                              | コスト感     | 向いてる状況                          | TLS     | 日本語UI |
| ----------------------------------- | ------------ | ------------------------------------- | ------- | -------- |
| A. ロリポップ！マネージドクラウド   | 月¥0〜       | 個人・日本向けデモで簡単に見せたい    | 自動    | ○        |
| B. Fly.io                           | 月$0〜$5     | Docker 活かしたい / 海外から見られる  | 自動    | ×        |
| C. 自前 VPS で `docker compose`     | 月¥500〜     | 既に VPS を持っている / 全部を制御    | 要自設定 | ×       |

結論から言うと、**「とりあえず動かして人に見せたい」だけなら A が一番近道** です。
開発者視点での素直さ・docker互換性・将来性を取るなら **B が素直** です。

以下、順に手順。

---

## A. ロリポップ！マネージドクラウド (MC)

PHP/Node/Ruby など複数ランタイム対応の PaaS。本アプリの場合は **Node.js ランタイムで frontend**、**backend は別プロジェクトで Go を** という構成を取るのが無難です。DB は MC が用意する PostgreSQL を利用します。

> ℹ️ MC のランタイム対応は時期により変わるので、Go が利用できない場合は **backend だけ Fly.io に置いて frontend を MC に置く**、という分担もアリです。

### 1. 下ごしらえ

```bash
# GitHub Actions からのデプロイ用に、MC 側でアプリを 2 つ作る
#   - cmaas-api        (Go)
#   - cmaas-web        (Next.js)
# DB 追加メニューから PostgreSQL を有効化
```

### 2. 接続情報を Secrets に入れる

GitHub リポジトリ → Settings → Secrets:

- `MC_API_TOKEN` : MC の API トークン
- `DATABASE_URL` : MC Postgres の接続 URL
- `NEXT_PUBLIC_API_BASE` : `https://cmaas-api.mc.example.com`（= デプロイ後の backend URL）

### 3. GitHub Actions でビルド＆デプロイ

`.github/workflows/deploy-mc.yml` を作る（MC の CLI / Action を利用）:

```yaml
name: Deploy (MC)
on:
  push:
    branches: [main]

jobs:
  api:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker/build-push-action@v6
        with:
          context: ./backend
          push: true
          tags: ghcr.io/${{ github.repository }}/api:${{ github.sha }}
      - name: Deploy to MC
        run: mc deploy cmaas-api --image ghcr.io/${{ github.repository }}/api:${{ github.sha }}
        env:
          MC_API_TOKEN: ${{ secrets.MC_API_TOKEN }}

  web:
    runs-on: ubuntu-latest
    needs: api
    steps:
      - uses: actions/checkout@v4
      - uses: docker/build-push-action@v6
        with:
          context: ./frontend
          build-args: |
            NEXT_PUBLIC_API_BASE=${{ secrets.NEXT_PUBLIC_API_BASE }}
          push: true
          tags: ghcr.io/${{ github.repository }}/web:${{ github.sha }}
      - name: Deploy to MC
        run: mc deploy cmaas-web --image ghcr.io/${{ github.repository }}/web:${{ github.sha }}
        env:
          MC_API_TOKEN: ${{ secrets.MC_API_TOKEN }}
```

> 🧪 マイグレーションは **初回だけ手動** で `psql $DATABASE_URL -f backend/migrations/001_init.sql`。
> 継続的にやるなら `golang-migrate` を CI に足す。

---

## B. Fly.io

Docker コンテナがそのまま動き、Postgres も `fly postgres create` で生えるので、このリポジトリとの相性が一番良い。

### 1. CLI 準備

```bash
brew install flyctl
fly auth login
```

### 2. Postgres を立てる

```bash
fly postgres create --name cmaas-db --region nrt --initial-cluster-size 1 --vm-size shared-cpu-1x --volume-size 1
# 出力された DATABASE_URL は次で使うのでメモ
```

### 3. backend デプロイ

```bash
cd backend
fly launch --name cmaas-api --region nrt --no-deploy --dockerfile Dockerfile
# fly.toml が生成される
```

`fly.toml` に `[env]` と `[[services]].internal_port = 8080` を確認。

```bash
fly postgres attach cmaas-db --app cmaas-api   # DATABASE_URL が自動で Secrets に入る
fly deploy
```

マイグレーションは初回に一度だけ:

```bash
fly proxy 5432 -a cmaas-db &
psql postgres://postgres:$PASS@localhost:5432/cmaas_api -f backend/migrations/001_init.sql
```

### 4. frontend デプロイ

```bash
cd frontend
fly launch --name cmaas-web --region nrt --no-deploy --dockerfile Dockerfile
fly secrets set NEXT_PUBLIC_API_BASE=https://cmaas-api.fly.dev -a cmaas-web
fly deploy
```

Next.js は `next build` 時点で `NEXT_PUBLIC_*` をバンドルに焼き込むため、**API URL を変えたらビルドし直しが必要**。Fly の場合は `fly secrets set` → `fly deploy` で自動的にそうなる。

### 5. CORS を絞る

`backend/internal/httpapi/handler.go` の `cors.Handler` の `AllowedOrigins` を `["https://cmaas-web.fly.dev"]` に変えてデプロイし直す。

---

## C. 自前 VPS で `docker compose`

既に VPS（さくら / ConoHa / Lightsail 等）を持っていて、Docker が入っているなら最短。

### 1. サーバに SSH

```bash
ssh deploy@your-server
git clone git@github.com:haruotsu/countrymaam-as-a-service.git
cd countrymaam-as-a-service
```

### 2. 本番用 compose オーバーレイ

同階層に `docker-compose.prod.yml` を作る（dev 用 Dockerfile をやめて production ビルドを使う）:

```yaml
services:
  db:
    restart: always
    environment:
      POSTGRES_PASSWORD: ${DB_PASSWORD:?set in .env}

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile        # devではなく本番版
    environment:
      DATABASE_URL: postgres://cmaas:${DB_PASSWORD}@db:5432/cmaas?sslmode=disable
    restart: always
    volumes: []

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
      args:
        NEXT_PUBLIC_API_BASE: https://api.example.com
    restart: always
    volumes: []
```

`.env` は git 管理しない:

```env
DB_PASSWORD=<長いランダム文字列>
```

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

### 3. TLS 終端

前段に **Caddy** を置くのが楽:

```caddyfile
api.example.com {
  reverse_proxy localhost:8090
}
cmaas.example.com {
  reverse_proxy localhost:3900
}
```

`caddy run` するだけで Let's Encrypt から証明書を自動取得。

### 4. 更新デプロイ

```bash
git pull
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

ダウンタイムゼロにしたいなら `--scale backend=2` にして Caddy の upstream を並べる、等。

---

## 本番向けのチェックリスト

- [ ] `AllowedOrigins` を本番ドメインに限定した
- [ ] 認証を組み込んだ（最低でも「他人の口座にアクセスできない」状態に）
- [ ] `DATABASE_URL` がシークレット管理下にある
- [ ] マイグレーションの運用方法が決まっている (CI から `golang-migrate` 等)
- [ ] ログが流れ先（標準出力でよいが、集約先）がある
- [ ] DB の自動バックアップが有効
- [ ] ヘルスチェック (`/healthz`) を PaaS 側の死活監視に刺した

---

## どれで行くか迷ったら

- **「今日動くところまで見せたい」**: Fly.io で backend + frontend + pg を30分で立てる
- **「国内ホスティングがいい / 月額の上限を作りたい」**: ロリポップ！マネージドクラウド
- **「自社のサーバがすでにある」**: VPS に `docker compose` 一発

まず Fly.io か MC で一度出してみて、反響があってから認証・監視・CI を整える、で十分です。
