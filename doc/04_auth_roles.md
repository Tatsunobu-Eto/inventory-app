# 認証・認可設計

## 認証方式

**セッションベース認証（Cookieセッション）**

ライブラリ：`github.com/gorilla/sessions`

### ログインフロー

```
1. ユーザーが POST /login にユーザー名・パスワードを送信
2. DB から username でユーザーレコードを取得
3. bcrypt.CompareHashAndPassword でパスワード検証
4. 検証成功 → セッションに user_id を保存（署名付きCookieに保存）
5. GET / (Dashboard) へリダイレクト

検証失敗 → エラートースト表示、ログインページ再表示
```

### セッション詳細

| 項目 | 値 |
|------|----|
| セッション名 | `"session"` |
| 保存先 | ブラウザCookie（署名済み） |
| 署名アルゴリズム | HMAC-SHA256（gorilla/securecookieによる） |
| 署名キー | 環境変数 `SESSION_KEY`（**必須**。未設定時は `log.Fatal` でアプリが起動を中断する） |
| 保存データ | `user_id`（int型） |

### ログアウトフロー

```
1. GET /logout にアクセス
2. セッションから user_id を削除・保存
3. GET /login へリダイレクト
```

## 認証ミドルウェア（`mw.Auth`）

**ファイル：** `middleware/auth.go`

```
リクエスト受信
    ↓
セッションから user_id 取得
    ↓ 取得失敗（未ログイン）
    → /login へリダイレクト（302）
    ↓ 取得成功
DB から GetUserByID でユーザー情報取得
    ↓ 取得失敗（ユーザー削除等）
    → /login へリダイレクト（302）
    ↓ 取得成功
context.WithValue に User オブジェクトを注入 (key: "user")
    ↓
次のハンドラへ処理継続
```

ハンドラ内では `mw.CurrentUser(r)` を呼び出すことで `*models.User` を取得できる。

## 認可ミドルウェア（`mw.RequireRole`）

**ファイル：** `middleware/auth.go`

```go
func RequireRole(roles ...string) func(http.Handler) http.Handler
```

- `mw.Auth` の後段で動作（前提としてユーザーがコンテキストに存在）
- 指定ロールのいずれかを持たない場合は HTTP 403 Forbidden を返す
- `mw.RequireRole("admin")` は `admin` ロールのユーザーのみ許可
- `mw.RequireRole("sysadmin")` は `sysadmin` ロールのユーザーのみ許可

**注意：** `admin` が `sysadmin` ルートにアクセスすることは403となる（sysadminがadminルートにアクセスすることも同様に403）。ロールは独立して管理される。

## ロール定義と権限マトリクス

### ロール一覧

| ロール | 値 | 説明 |
|--------|-----|------|
| システム管理者 | `sysadmin` | 全体管理者。部門・adminの作成権限を持つ。部門には所属しない（department_id = NULL） |
| 部門管理者 | `admin` | 特定部門の管理者。自部門のユーザー・アイテムを管理 |
| 一般ユーザー | `user` | 一般部門員。自分のアイテムとマーケットを利用 |

### 権限マトリクス

| 機能・操作 | sysadmin | admin | user |
|-----------|:--------:|:-----:|:----:|
| **部門管理** | | | |
| 部門一覧閲覧 | ✅ | - | - |
| 部門作成 | ✅ | - | - |
| **ユーザー管理** | | | |
| 部門管理者アカウント作成 | ✅ | - | - |
| 一般ユーザーアカウント作成 | - | ✅ | - |
| 自部門ユーザー一覧閲覧 | - | ✅ | - |
| **アイテム管理** | | | |
| 自分のアイテム一覧 | - | ✅ | ✅ |
| アイテム登録（自分名義） | - | ✅ | ✅ |
| アイテム登録（代理・他ユーザー名義） | - | ✅ | - |
| アイテム詳細閲覧 | - | ✅ | ✅ |
| アイテム説明・画像編集（所有者のみ） | - | △所有者 | △所有者 |
| 部門内全アイテム一覧（admin用） | - | ✅ | - |
| **マーケット** | | | |
| マーケット一覧閲覧 | - | ✅ | ✅ |
| マーケット検索 | - | ✅ | ✅ |
| アイテム出品（自分の所有物のみ） | - | ✅ | ✅ |
| 出品取り下げ（自分の出品のみ） | - | ✅ | ✅ |
| マーケットアイテムへの申請 | - | ✅ | ✅ |
| **ダッシュボード** | | | |
| ダッシュボード閲覧 | ✅ | ✅ | ✅ |
| **ユーザー操作（admin管理下）** | | | |
| 自部門ユーザー削除 | - | ✅（userロールのみ） | - |
| 自部門ユーザーパスワードリセット | - | ✅（userロールのみ） | - |
| 自部門ユーザー他部門異動 | - | ✅（userロールのみ） | - |
| **ユーザー操作（sysadmin管理下）** | | | |
| 任意ユーザー削除 | ✅ | - | - |
| 任意ユーザーパスワードリセット | ✅ | - | - |
| 任意ユーザー他部門異動（sysadmin除く） | ✅ | - | - |
| userをadminに昇格 | ✅ | - | - |
| adminをuserに降格 | ✅ | - | - |
| **取引・プロフィール** | | | |
| 取引履歴閲覧（自分が関与した取引） | - | ✅ | ✅ |
| パスワード変更（自分のパスワード） | ✅ | ✅ | ✅ |
| **全アイテム閲覧** | | | |
| 全部門アイテム一覧 | ✅ | - | - |

※ `-` は該当ページ自体へのアクセス権限がないことを示す（403またはリダイレクト）

## 初期 sysadmin 作成フロー

アプリ起動時に `main.go` の `seedSysadmin()` が実行される。

```
1. SELECT COUNT(*) FROM users WHERE role = 'sysadmin' を実行
2. sysadmin が1人以上存在する → スキップ（冪等性確保）
3. sysadmin が0人かつ INIT_SYSADMIN_USER / INIT_SYSADMIN_PASS が環境変数に設定済み
   → models.CreateUser でsysadminユーザーを作成
4. 環境変数未設定の場合はログに警告を出してスキップ
```

**注意：** 初回セットアップ後は `INIT_SYSADMIN_USER` / `INIT_SYSADMIN_PASS` の値を変更しても再実行されないため（`seedSysadmin` の冪等性確保のため）、sysadmin のパスワードは `GET /profile/password` の UI から変更してください。

## パスワードハッシュ

**アルゴリズム：** bcrypt  
**コスト：** `bcrypt.DefaultCost`（現在は10）

```go
// ハッシュ生成
bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

// 検証
bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
```

PasswordHashフィールドはJSONシリアライズ時に `json:"-"` タグにより除外される。
