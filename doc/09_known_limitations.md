# 既知の制限事項・将来課題

現バージョンの既知の制限と、意図的な設計上の省略をまとめる。

---

## 1. ~~取引履歴の閲覧 UI 未実装~~ → 実装済み

`GET /transactions` エンドポイントを追加し、ログインユーザーが関与した取引（受け取り・譲渡）を一覧表示できるようになった。

- `models/transaction.go` — `Transaction` 構造体と `ListTransactionsByUser()` を追加
- `handlers/items.go` — `Transactions()` ハンドラーを追加
- `templates/transactions.html` — 取引履歴テンプレートを追加
- `templates/layout.html` — サイドバーに「取引履歴」リンクを追加（`user` / `admin` ロール向け）

---

## 2. ~~パスワード変更 UI の未実装~~ → 実装済み

`GET /profile/password`（フォーム表示）と `POST /profile/password`（更新処理）を追加した。

- `models/user.go` — `UpdatePassword()` を追加
- `handlers/auth.go` — `PasswordChangePage()` / `PasswordChangePost()` を追加
- `templates/password_change.html` — パスワード変更テンプレートを追加
- `templates/layout.html` — サイドバー下部に「パスワード変更」リンクを追加

変更時は現在のパスワードの確認・新パスワードと確認フィールドの一致チェックを行う。

> **注意：** `INIT_SYSADMIN_PASS` を `.env` で変更しても、sysadmin が既に存在する場合は再適用されない（`seedSysadmin` の冪等性確保のため）。UI からパスワードを変更してください。

---

## 3. 一括操作の非対応

アイテムの一括削除・一括出品・一括取り下げは未実装。操作は 1 アイテムずつ個別に行う必要がある。

---

## 4. ~~cascade 削除時の画像ファイル残留~~ → 修正済み

以下の2経路でアイテムが削除される場合に、`uploads/{item_id}/` 配下の実ファイルも確実に削除されるよう対応した。

**① 90日期限切れ（バックグラウンドジョブ）**

- `models/item.go` — `ExpireMarketItems()` を `RETURNING id` に変更し、期限切れになったアイテムの ID リストを返すよう修正
- `main.go` — `cleanupItemFiles()` ヘルパーを追加。期限切れ ID を受け取り、画像ファイルおよび `uploads/{item_id}/` ディレクトリを削除する

**② ユーザーによる手動削除（新機能）**

- `models/item.go` — `DeleteItem()` を追加。取引履歴がある場合はエラー返却。画像パスを収集してから DB レコード（CASCADE で `item_images` も削除）を DELETE し、パスを返す
- `handlers/items.go` — `DeleteItem()` ハンドラーを追加。オーナー確認・`private` ステータス確認のうえ DB 削除・ファイル削除を実行
- `main.go` — ルート `POST /items/{item_id}/delete` を追加
- `templates/item_detail.html` — `private` ステータスかつオーナーのみ表示される「アイテムを削除」ボタンを追加（取引履歴があるアイテムは削除不可）

---

## 5. ポートのデフォルト値の不一致

- `main.go` の `PORT` 未設定時デフォルト値は `8081`
- `.env` に `PORT=8080` を設定しているため、Docker Compose 経由での起動では問題なし
- `.env` なしで `go run .` を実行すると `DATABASE_URL` 未設定でアプリが Fatal 終了するため現実には問題にならないが、`.env` に `PORT` を記載し忘れた場合は `8081` でリッスンする点に注意

---

## 6. ~~部門重複時のエラーハンドリング~~ → 修正済み

`POST /sysadmin/departments` で既存の部門名を入力した場合、PostgreSQL の UNIQUE 制約違反（エラーコード `23505`）を検出し、「その部門名はすでに存在します」というメッセージを返すよう修正した（`handlers/admin.go`）。

---

## 7. ロールの上位互換性なし

- `sysadmin` は `admin` ルート（`/admin/*`）にアクセスできない（403）
- `admin` は `sysadmin` ルート（`/sysadmin/*`）にアクセスできない（403）
- ロールは完全に独立しており、`sysadmin` がアイテムを登録するといった操作は不可（`department_id` が NULL のため 403 になる）

これは意図的な設計であり、ロール間に上位互換性は設けていない。

---

## 8. ~~Dockerfile の重複 COPY（技術的負債）~~ → 修正済み

`main.go` の `//go:embed` 指令によりビルド時に `templates/`・`static/`・`migrations/` はバイナリへ組み込まれるため、実行ステージでの以下の `COPY` は不要だった。

```dockerfile
# 削除済み
COPY templates/ templates/
COPY static/ static/
COPY migrations/ migrations/
```

これらを `Dockerfile` から削除し、イメージサイズを削減した。
