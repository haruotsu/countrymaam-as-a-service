# テストリスト (t_wada スタイル TDD)

進行中のテストシナリオの一覧。実装中に気づきがあれば随時追加・修正する。

## 凡例

- `[ ]` 未着手 / `[~]` テスト失敗中 (Red) / `[x]` パス済 (Green)

---

## ドメイン層

### Flavor

- [ ] `Parse("vanilla")` / `Parse("chocolate")` / `Parse("matcha")` は有効
- [ ] `Parse("")` / `Parse("strawberry")` はエラー
- [ ] `String()` でフレーバー名を返す

### ExchangeRate

- [ ] `Convert(vanilla, chocolate, 10) == 8`
- [ ] `Convert(vanilla, vanilla, 10) == 10`（同一フレーバーは恒等）
- [ ] `Convert(matcha, vanilla, 6) == 9`
- [ ] `Convert(any, any, 0)` は `InvalidAmount` エラー
- [ ] `Convert(any, any, -1)` は `InvalidAmount` エラー
- [ ] 換算後が 0 になる場合（小さな額）はエラー（"too small to exchange"）

### Account

- [ ] `NewAccount(userID, flavor)` で初期残高 0
- [ ] `Deposit(5)` で残高 +5
- [ ] `Deposit(0)` はエラー
- [ ] `Deposit(-1)` はエラー
- [ ] `Withdraw(3)`（残高 10）で残高 7
- [ ] `Withdraw(11)`（残高 10）は `InsufficientBalance` エラー
- [ ] `Withdraw(0)` はエラー

---

## サービス層（ユースケース）

### TransferService

- [ ] 同フレーバー口座間で残高が正しく移動
- [ ] 違うフレーバー間は `FlavorMismatch` エラー
- [ ] 残高不足は `InsufficientBalance` エラーかつ両口座の残高は変わらない
- [ ] 同一口座への送金は `SelfTransfer` エラー

### ExchangeService

- [ ] 同一ユーザーの vanilla 口座 → chocolate 口座 へレート換算して移動
- [ ] 別ユーザー間は `ForeignExchange` エラー
- [ ] 残高不足はエラー

---

## リポジトリ層（PostgreSQL、dockertest）

- [ ] `UserRepo.Create` / `FindByID` / `FindByEmail`
- [ ] 同一 email で二重 Create は 409 相当のエラー
- [ ] `AccountRepo.Create` / `FindByID` / `FindByUserIDAndFlavor`
- [ ] 同一 user×flavor の二重口座開設はエラー
- [ ] `TransactionRepo.Create` / `ListByAccount`
- [ ] `Tx.Transfer` がトランザクション内で両残高 + 2 件の取引を書き込む
- [ ] `Tx.Transfer` が片側失敗時に両方ロールバック

---

## HTTP (httptest)

- [x] `POST /auth/register` で Cookie 発行
- [x] `POST /auth/login` / `POST /auth/logout`
- [x] `GET /auth/me` 未認証は 401
- [x] 他人の口座アクセス（GET/deposit/withdraw/transactions）は 403
- [x] `POST /accounts` 201（自分名義）
- [x] `POST /accounts/:id/deposit` で残高反映
- [x] `POST /accounts/:id/withdraw` 残高不足は 422
- [x] `POST /transfers` 成功
- [x] `POST /exchanges` 成功
- [x] `GET /accounts/:id/transactions` 履歴取得
- [x] `POST /auth/register` 弱いパスワードは 422

## 認証ドメイン / サービス

- [x] `HashPassword` + `VerifyPassword` の往復
- [x] 弱いパスワードは `ErrWeakPassword`
- [x] `NewSessionToken` が 100 回で重複しない
- [x] `Session.IsExpired`
- [x] `Register` でセッション発行
- [x] `Login` 誤りは `ErrInvalidCredentials`
- [x] `Logout` → `Authenticate` が `ErrUnauthenticated`
- [x] 非オーナーの `Deposit/Withdraw` は `ErrForbidden`
