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

- [ ] `POST /users` 201 + body
- [ ] `POST /users` 同じ email は 409
- [ ] `POST /accounts` 201
- [ ] `POST /accounts/:id/deposit` で残高反映
- [ ] `POST /accounts/:id/withdraw` 残高不足は 422
- [ ] `POST /transfers` 成功
- [ ] `POST /exchanges` 成功
- [ ] `GET /accounts/:id/transactions` 履歴取得
