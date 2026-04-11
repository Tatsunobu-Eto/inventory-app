# APIルート仕様

## ルート一覧

### パブリックルート（認証不要）

| メソッド | パス | ハンドラ | 説明 |
|---------|------|---------|------|
| GET | `/login` | `env.LoginPage` | ログイン画面を表示 |
| POST | `/login` | `env.LoginPost` | ログイン処理 |
| GET | `/logout` | `env.Logout` | ログアウト処理・/loginへリダイレクト |
| GET | `/static/*` | `http.FileServer` | 静的ファイル配信（CSS, JS） |
| GET | `/uploads/*` | `http.FileServer` | アップロード画像配信 |

---

### 認証済みルート（`mw.Auth` 必須）

#### ダッシュボード

| メソッド | パス | ハンドラ | 必要ロール | 説明 |
|---------|------|---------|----------|------|
| GET | `/` | `env.Dashboard` | user以上 | ダッシュボード表示 |

#### マーケット・アイテム（user/admin共通）

| メソッド | パス | ハンドラ | 必要ロール | 説明 |
|---------|------|---------|----------|------|
| GET | `/market` | `env.MarketList` | user以上 | マーケット一覧（ページネーション・検索） |
| GET | `/my-items` | `env.MyItems` | user以上 | 自分のアイテム一覧 |
| GET | `/items/new` | `env.CreateItemForm` | user以上 | アイテム登録フォーム表示 |
| POST | `/items/new` | `env.CreateItemPost` | user以上 | アイテム登録処理 |
| GET | `/items/{item_id}` | `env.ItemDetail` | user以上 | アイテム詳細表示 |
| POST | `/items/{item_id}` | `env.UpdateItemPost` | user以上（所有者のみ有効） | アイテム説明・画像更新 |
| POST | `/items/{item_id}/delete` | `env.DeleteItem` | user以上（所有者かつstatus=private のみ有効） | アイテム削除 |
| POST | `/items/images/{image_id}/delete` | `env.DeleteItemImage` | user以上（所有者のみ有効） | アイテム画像削除 |
| POST | `/items/put-on-market` | `env.PutOnMarket` | user以上 | アイテムをマーケットに出品 |
| POST | `/items/withdraw` | `env.WithdrawFromMarket` | user以上 | マーケットから取り下げ |
| POST | `/items/apply` | `env.ApplyForItem` | user以上 | マーケットアイテムへ申請 |

#### プロフィール・履歴（user/admin/sysadmin 共通）

| メソッド | パス | ハンドラ | 必要ロール | 説明 |
|---------|------|---------|----------|------|
| GET | `/transactions` | `env.Transactions` | user以上 | 取引履歴一覧（ページネーション） |
| GET | `/profile/password` | `env.PasswordChangePage` | user以上 | パスワード変更フォーム表示 |
| POST | `/profile/password` | `env.PasswordChangePost` | user以上 | パスワード変更処理 |

#### 部門管理者ルート（`mw.RequireRole("admin")` 必須）

| メソッド | パス | ハンドラ | 必要ロール | 説明 |
|---------|------|---------|----------|------|
| GET | `/admin/users` | `env.AdminUsers` | admin | 部門ユーザー一覧表示 |
| POST | `/admin/users` | `env.AdminCreateUser` | admin | 一般ユーザー作成 |
| GET | `/admin/items` | `env.AdminDeptItems` | admin | 部門内全アイテム一覧（ページネーション・検索） |
| GET | `/admin/items/new` | `env.AdminCreateItemForm` | admin | アイテム代理登録フォーム表示 |
| POST | `/admin/items` | `env.AdminCreateItem` | admin | アイテム代理登録処理 |
| POST | `/admin/users/{user_id}/delete` | `env.AdminDeleteUser` | admin | 自部門の一般ユーザー削除 |
| POST | `/admin/users/{user_id}/reset-password` | `env.AdminResetPassword` | admin | 自部門ユーザーのパスワードリセット |
| POST | `/admin/users/{user_id}/transfer` | `env.AdminTransferUser` | admin | 自部門ユーザーを他部門に異動 |

#### システム管理者ルート（`mw.RequireRole("sysadmin")` 必須）

| メソッド | パス | ハンドラ | 必要ロール | 説明 |
|---------|------|---------|----------|------|
| GET | `/sysadmin/departments` | `env.SysAdminDepartments` | sysadmin | 部門一覧・管理画面表示 |
| POST | `/sysadmin/departments` | `env.SysAdminCreateDepartment` | sysadmin | 部門作成 |
| POST | `/sysadmin/admins` | `env.SysAdminCreateAdmin` | sysadmin | 部門管理者作成 |
| GET | `/sysadmin/items` | `env.SysAdminAllItems` | sysadmin | 全部門・全アイテム一覧（検索・ページネーション） |
| POST | `/sysadmin/users/{user_id}/promote` | `env.SysAdminPromoteToAdmin` | sysadmin | 一般ユーザーを部門管理者に昇格 |
| POST | `/sysadmin/users/{user_id}/demote` | `env.SysAdminDemoteToUser` | sysadmin | 部門管理者を一般ユーザーに降格 |
| POST | `/sysadmin/users/{user_id}/delete` | `env.SysAdminDeleteUser` | sysadmin | ユーザー削除 |
| POST | `/sysadmin/users/{user_id}/reset-password` | `env.SysAdminResetPassword` | sysadmin | ユーザーのパスワードリセット |
| POST | `/sysadmin/users/{user_id}/transfer` | `env.SysAdminTransferUser` | sysadmin | ユーザーを別部門に異動 |

---

## 各エンドポイント詳細

### POST /login

**リクエスト（form-data）：**

| フィールド | 型 | 必須 | 説明 |
|-----------|----|------|------|
| username | string | ✅ | ログインID |
| password | string | ✅ | パスワード（平文） |

**レスポンス：**
- 成功: 302 → `/`
- 失敗: 200 + `login.html` 再表示 + `HX-Trigger: showToast` ヘッダー

---

### GET /market

**クエリパラメータ：**

| パラメータ | 型 | デフォルト | 説明 |
|-----------|----|-----------|------|
| q | string | "" | 検索キーワード（タイトル・説明文のILIKE検索） |
| page | int | 1 | ページ番号（1始まり） |

**動作：**
- 1ページあたり10件（perPage = 10）
- `CountItems` で総件数を取得し、`TotalPages` を算出して渡す
- `HX-Request: true` ヘッダーがある場合（HTMX部分更新）は `item_list_partial.html` のみ返す

---

### POST /items/new

**リクエスト（multipart/form-data）：**

| フィールド | 型 | 必須 | 説明 |
|-----------|----|------|------|
| title | string | ✅ | アイテム名称 |
| description | string | - | 説明文 |
| images | file（複数可） | - | 画像ファイル（最大5枚、各10MB以下） |

**レスポンス：**
- 成功: `HX-Redirect: /my-items` + `HX-Trigger: showToast`
- バリデーションエラー: 422 + `HX-Trigger: showToast`

---

### POST /items/{item_id}

**リクエスト（multipart/form-data）：**

| フィールド | 型 | 必須 | 説明 |
|-----------|----|------|------|
| description | string | - | 説明文（更新後の値） |
| images | file（複数可） | - | 追加画像（既存枚数＋追加枚数が5を超える場合はエラー） |

**権限チェック：** ログインユーザーがアイテムの `owner_id` と一致しない場合は403

---

### POST /items/images/{image_id}/delete

**権限チェック：** 画像のアイテムの `owner_id` がログインユーザーと一致しない場合は403  
**副作用：** `uploads/{item_id}/{filename}` のファイルシステム上の実ファイルも削除  
**レスポンス：** `HX-Refresh: true`（ページリロード）

---

### POST /items/put-on-market

**リクエスト（form-data）：**

| フィールド | 型 | 必須 | 説明 |
|-----------|----|------|------|
| item_id | int | ✅ | 出品するアイテムID |

**DBの更新：**
```sql
UPDATE items SET status = 'market', market_at = NOW()
WHERE id = $1 AND owner_id = $2 AND status = 'private'
```
所有者チェック・ステータスチェックはSQL側で行う（`owner_id = $2 AND status = 'private'` 条件）。

---

### POST /items/withdraw

**リクエスト（form-data）：**

| フィールド | 型 | 必須 | 説明 |
|-----------|----|------|------|
| item_id | int | ✅ | 取り下げるアイテムID |

**DBの更新：**
```sql
UPDATE items SET status = 'private', market_at = NULL
WHERE id = $1 AND owner_id = $2 AND status = 'market'
```

---

### POST /items/apply

**リクエスト（form-data）：**

| フィールド | 型 | 必須 | 説明 |
|-----------|----|------|------|
| item_id | int | ✅ | 申請するアイテムID |

**レスポンス：**
- 成功: `HX-Redirect: /my-items` + `HX-Trigger: showToast`
- 競合（他ユーザーが先に申請）: 409 Conflict + `HX-Trigger: showToast`（「既に他のユーザーが応募済みです」）

**詳細は機能仕様（06_features.md）の「マーケット申請の排他制御」を参照。**

---

### POST /admin/users

**リクエスト（form-data）：**

| フィールド | 型 | 必須 | 説明 |
|-----------|----|------|------|
| username | string | ✅ | ログインID |
| password | string | ✅ | パスワード（平文） |
| display_name | string | - | 表示名 |

作成されるユーザーのロールは `user`、department_id は実行adminの部門と同じ。

---

### POST /admin/items

**リクエスト（multipart/form-data）：**

| フィールド | 型 | 必須 | 説明 |
|-----------|----|------|------|
| title | string | ✅ | アイテム名称 |
| description | string | - | 説明文 |
| owner_id | int | ✅ | 所有者ユーザーID（部門内ユーザーから選択） |
| images | file（複数可） | - | 画像ファイル |

`owner_id` には選択したユーザーID、`created_by` には実行adminのIDが設定される。

---

### POST /sysadmin/departments

**リクエスト（form-data）：**

| フィールド | 型 | 必須 | 説明 |
|-----------|----|------|------|
| name | string | ✅ | 部門名（一意） |

---

### POST /sysadmin/admins

**リクエスト（form-data）：**

| フィールド | 型 | 必須 | 説明 |
|-----------|----|------|------|
| department_id | int | ✅ | 所属部門ID |
| username | string | ✅ | ログインID |
| password | string | ✅ | パスワード（平文） |
| display_name | string | - | 表示名 |

作成されるユーザーのロールは `admin`。

---

### POST /items/{item_id}/delete

**権限チェック：**
1. ログインユーザーが `owner_id` でなければ 403
2. `status != 'private'` の場合は 422 + エラートースト（「非公開状態のアイテムのみ削除できます」）
3. 取引履歴（`transactions`）が存在する場合は `models.DeleteItem` がエラーを返し、422 + エラートースト

**副作用：**
- `models.DeleteItem` が画像パスを収集し、DBレコード（`item_images` もCASCADE）を削除
- `uploads/{item_id}/{filename}` のファイル群と `uploads/{item_id}/` ディレクトリを削除

**レスポンス：**
- 成功: `HX-Redirect: /my-items` + `HX-Trigger: showMessage`

---

### GET /transactions

**クエリパラメータ：**

| パラメータ | 型 | デフォルト | 説明 |
|-----------|----|-----------|------|
| page | int | 1 | ページ番号（1始まり） |

**動作：**
- ログインユーザーが `from_user_id` または `to_user_id` に含まれる取引を取得
- 1ページあたり10件（perPage = 10）
- `HX-Request: true` の場合は `transactions_partial.html` のみ返す

---

### GET /profile/password

パスワード変更フォームを表示する。全ロールが利用可能。

**レスポンス：** 200 + `password_change.html`

---

### POST /profile/password

**リクエスト（form-data）：**

| フィールド | 型 | 必須 | 説明 |
|-----------|----|------|------|
| current_password | string | ✅ | 現在のパスワード |
| new_password | string | ✅ | 新しいパスワード |
| confirm_password | string | ✅ | 新しいパスワード（確認用） |

**バリデーション：**
1. 全フィールド必須（空なら 422）
2. `new_password == confirm_password` チェック（不一致なら 422）
3. `current_password` が現在のハッシュと一致するか bcrypt で検証（不一致なら 422）

**レスポンス：**
- 成功: `HX-Redirect: /` + `HX-Trigger: showMessage`（「パスワードを変更しました」）
- 失敗: 422 + `HX-Trigger: showMessage`

---

### POST /admin/users/{user_id}/delete

**制約：** 対象ユーザーが自部門の `role = 'user'` でなければ 403  
**副作用：** 関連アイテム・画像を CASCADE 削除し、ファイルシステムの画像ファイルも削除  
**レスポンス：** `HX-Redirect: /admin/users` + `HX-Trigger: showMessage`

---

### POST /admin/users/{user_id}/reset-password

**リクエスト（form-data）：**

| フィールド | 型 | 必須 | 説明 |
|-----------|----|------|------|
| new_password | string | ✅ | 新しいパスワード |

**制約：** 対象ユーザーが自部門の `role = 'user'` でなければ 403  
**レスポンス：** `HX-Redirect: /admin/users` + `HX-Trigger: showMessage`

---

### POST /admin/users/{user_id}/transfer

**リクエスト（form-data）：**

| フィールド | 型 | 必須 | 説明 |
|-----------|----|------|------|
| department_id | int | ✅ | 異動先部門ID |

**制約：**
- 対象ユーザーが自部門の `role = 'user'` でなければ 403
- `department_id` が現在の部門と同じなら 422

**レスポンス：** `HX-Redirect: /admin/users` + `HX-Trigger: showMessage`

---

### POST /sysadmin/users/{user_id}/promote

**制約：** 対象ユーザーの `role = 'user'` でなければ 422  
**副作用：** ロールを `'admin'` に変更  
**レスポンス：** `HX-Redirect: /sysadmin/departments` + `HX-Trigger: showMessage`

---

### POST /sysadmin/users/{user_id}/demote

**制約：** 対象ユーザーの `role = 'admin'` でなければ 422  
**副作用：** ロールを `'user'` に変更  
**レスポンス：** `HX-Redirect: /sysadmin/departments` + `HX-Trigger: showMessage`

---

### POST /sysadmin/users/{user_id}/delete

**副作用：** 関連アイテム・画像を CASCADE 削除し、ファイルシステムの画像ファイルも削除  
**レスポンス：** `HX-Redirect: /sysadmin/departments` + `HX-Trigger: showMessage`

---

### POST /sysadmin/users/{user_id}/reset-password

**リクエスト（form-data）：**

| フィールド | 型 | 必須 | 説明 |
|-----------|----|------|------|
| new_password | string | ✅ | 新しいパスワード |

**制約：** sysadmin 権限があれば対象ユーザーのロール問わずリセット可  
**レスポンス：** `HX-Redirect: /sysadmin/departments` + `HX-Trigger: showMessage`

---

### POST /sysadmin/users/{user_id}/transfer

**リクエスト（form-data）：**

| フィールド | 型 | 必須 | 説明 |
|-----------|----|------|------|
| department_id | int | ✅ | 異動先部門ID |

**制約：**
- 対象ユーザーの `role = 'sysadmin'` なら 403
- 現在の部門と同じ `department_id` なら 422

**レスポンス：** `HX-Redirect: /sysadmin/departments` + `HX-Trigger: showMessage`

---

### GET /sysadmin/items

**クエリパラメータ：**

| パラメータ | 型 | デフォルト | 説明 |
|-----------|----|-----------|------|
| q | string | "" | 検索キーワード（タイトル・説明文の ILIKE 検索） |
| status | string | "" | ステータス絞り込み（`private`/`market`/`deleted`/空=全件） |
| owner_id | int | 0 | 所有者絞り込み（0=全員） |
| page | int | 1 | ページ番号 |

**動作：**
- 全部門のアイテムを取得（DepartmentID フィルタなし）
- 1ページあたり10件
- テンプレートデータに `AllUsers`（全ユーザー一覧、owner フィルタ用 select ボックス）を渡す

---

## HTMX連携パターン

### トースト通知

サーバー側でHTTPレスポンスヘッダーに以下を設定することで、Alpine.jsが通知を表示する。

```go
// handlers/render.go
func triggerToast(w http.ResponseWriter, message string) {
    w.Header().Set("HX-Trigger", `{"showToast":"`+message+`"}`)
}
```

### ページネーション

マーケット一覧・管理者アイテム一覧では `hx-get` によるHTMXリクエストでページ切り替え。
`HX-Request: true` ヘッダーを検知してハンドラが部分テンプレートのみ返す。

### リダイレクト

フォーム送信成功後は `HX-Redirect` ヘッダーを返すことでHTMX経由でページ遷移する。

### ページリフレッシュ

画像削除後は `HX-Refresh: true` ヘッダーを返すことでHTMXがページ全体をリロードする。
