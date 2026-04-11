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
| POST | `/items/images/{image_id}/delete` | `env.DeleteItemImage` | user以上（所有者のみ有効） | アイテム画像削除 |
| POST | `/items/put-on-market` | `env.PutOnMarket` | user以上 | アイテムをマーケットに出品 |
| POST | `/items/withdraw` | `env.WithdrawFromMarket` | user以上 | マーケットから取り下げ |
| POST | `/items/apply` | `env.ApplyForItem` | user以上 | マーケットアイテムへ申請 |

#### 部門管理者ルート（`mw.RequireRole("admin")` 必須）

| メソッド | パス | ハンドラ | 必要ロール | 説明 |
|---------|------|---------|----------|------|
| GET | `/admin/users` | `env.AdminUsers` | admin | 部門ユーザー一覧表示 |
| POST | `/admin/users` | `env.AdminCreateUser` | admin | 一般ユーザー作成 |
| GET | `/admin/items` | `env.AdminDeptItems` | admin | 部門内全アイテム一覧（ページネーション・検索） |
| GET | `/admin/items/new` | `env.AdminCreateItemForm` | admin | アイテム代理登録フォーム表示 |
| POST | `/admin/items` | `env.AdminCreateItem` | admin | アイテム代理登録処理 |

#### システム管理者ルート（`mw.RequireRole("sysadmin")` 必須）

| メソッド | パス | ハンドラ | 必要ロール | 説明 |
|---------|------|---------|----------|------|
| GET | `/sysadmin/departments` | `env.SysAdminDepartments` | sysadmin | 部門一覧・管理画面表示 |
| POST | `/sysadmin/departments` | `env.SysAdminCreateDepartment` | sysadmin | 部門作成 |
| POST | `/sysadmin/admins` | `env.SysAdminCreateAdmin` | sysadmin | 部門管理者作成 |

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
- 1ページあたり20件（perPage = 20）
- 次ページの有無を判定するために `perPage+1` 件取得し、21件目の有無で `HasMore` を設定
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
