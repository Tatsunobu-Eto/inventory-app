# UI・画面仕様

## 画面一覧

| 画面名 | URL | テンプレート | 対象ロール | 概要 |
|--------|-----|------------|----------|------|
| ログイン | `/login` | `login.html` | 全員（未認証） | ユーザー名・パスワード入力 |
| ダッシュボード | `/` | `dashboard.html` | 全ロール | ロール別のクイックリンクカード一覧 |
| マーケット | `/market` | `market.html` | user, admin | 部門内出品アイテム検索・閲覧・申請 |
| マイアイテム | `/my-items` | `my_items.html` | user, admin | 自分の所有アイテム一覧・出品操作 |
| アイテム登録 | `/items/new` | `item_form.html` | user, admin | アイテム新規登録フォーム |
| アイテム詳細 | `/items/{id}` | `item_detail.html` | user, admin | アイテム詳細・編集（所有者のみ） |
| ユーザー管理 | `/admin/users` | `admin_users.html` | admin | 部門ユーザー一覧・作成 |
| 消耗品登録（管理者） | `/admin/items/new` | `admin_item_form.html` | admin | 代理アイテム登録（所有者を選択） |
| 部門アイテム一覧 | `/admin/items` | `admin_dept_items.html` | admin | 部門内全アイテム検索・閲覧 |
| 部門管理 | `/sysadmin/departments` | `sysadmin_departments.html` | sysadmin | 部門作成・管理者アカウント作成 |

---

## 共通レイアウト（`layout.html`）

全ページ（ログインを除く）で使用するベースレイアウト。

**構成：**
```
<html lang="ja">
  <head>
    - style.css（Tailwind CSS コンパイル済み）
    - htmx.min.js
    - alpine.min.js（defer）
  </head>
  <body x-data="{ toast: '', showToast: false }">
    <aside>  ← サイドバーナビゲーション
    <main>
      {{template "content" .}}  ← 各ページのコンテンツ
    </main>
    <div x-show="showToast">   ← トースト通知
    <script>  ← HTMX→Alpine.jsブリッジ
  </body>
```

**サイドバーのロール別メニュー：**

| メニュー項目 | 表示条件 |
|------------|---------|
| ダッシュボード | 全ロール |
| マーケット | user, admin |
| マイアイテム | user, admin |
| ユーザー管理 | admin のみ |
| 部門アイテム一覧 | admin のみ |
| 消耗品登録（管理者） | admin のみ |
| 部門管理 | sysadmin のみ |

サイドバー下部にはログインユーザーの `DisplayName` とログアウトリンクを表示。

---

## テンプレート詳細

### login.html（スタンドアロン）

レイアウトを使用しない独立したページ。

**表示要素：**
- システムタイトル「消耗品管理システム」
- ユーザー名・パスワード入力フォーム
- エラーメッセージ表示エリア

**特記：** `POST /login` は通常のHTMLフォーム送信（HTMXなし）。エラー時はページごと再表示。

---

### dashboard.html

**テンプレートデータ：** `map[string]any{"User": *models.User}`

**表示内容：** ロールに応じたクイックアクションカードをグリッド表示

| ロール | 表示カード |
|--------|----------|
| user, admin | マーケット、マイアイテム、アイテム登録 |
| admin | + ユーザー管理、部門アイテム一覧、消耗品登録（管理者） |
| sysadmin | 部門管理 |

---

### market.html + item_list_partial.html

**テンプレートデータ：**
```go
map[string]any{
    "User":     *models.User,
    "Items":    []models.Item,
    "ImageMap": map[int][]models.ItemImage,
    "Query":    string,
    "Page":     int,
    "HasMore":  bool,
}
```

**検索フォーム：**
- テキスト入力（`q` パラメータ）
- 検索ボタン（`hx-get="/market"`, `hx-target="#item-list"`, `hx-push-url="true"`）

**アイテムカード（`item_list_partial.html`）：**
- アイテム画像（1枚目を表示。画像なしの場合はプレースホルダー）
- タイトル・所有者名
- 申請ボタン（`hx-post="/items/apply"`, `hx-confirm="申請しますか？"`）

**ページネーション：**
- 「前へ」「次へ」リンク
- `hx-get` + `hx-target="#item-list"` + `hx-push-url="true"` でURLを更新しながら部分更新
- `HasMore=false` の場合は「次へ」を非表示
- `Page=1` の場合は「前へ」を非表示

**HTMX部分更新：**
- `HX-Request: true` ヘッダーがある場合はパーシャル（`item_list_partial.html`）のみ返す
- 初回アクセスはフルページ（`market.html`）を返す

---

### my_items.html

**テンプレートデータ：**
```go
map[string]any{
    "User":     *models.User,
    "Items":    []models.Item,
    "ImageMap": map[int][]models.ItemImage,
}
```

**ステータスバッジ：**

| status | バッジ表示 | 色 |
|--------|-----------|-----|
| `private` | 非公開 | グレー |
| `market` | 出品中 | ブルー |
| `deleted` | 期限切れ | レッド |

**操作ボタン（statusに応じて切り替え）：**
- `private` → 「マーケットへ出品」ボタン（`hx-post="/items/put-on-market"`, `hx-confirm`付き）
- `market` → 「出品取り下げ」ボタン（`hx-post="/items/withdraw"`, `hx-confirm`付き）
- `deleted` → ボタンなし

---

### item_form.html

アイテム新規登録フォーム（ユーザー用）。

**フィールド：**
- タイトル（テキスト、必須）
- 説明（テキストエリア）
- 画像アップロード（`type="file" multiple accept=".jpg,.jpeg,.png,.gif,.webp"`）

**送信：** `hx-post="/items/new"` + `hx-encoding="multipart/form-data"`

---

### item_detail.html

**テンプレートデータ：**
```go
map[string]any{
    "User":    *models.User,
    "Item":    models.Item,
    "Images":  []models.ItemImage,
    "IsOwner": bool,
}
```

**表示内容：**
- アイテムタイトル・説明文
- 画像ギャラリー（`/uploads/{file_path}` でサーブ）
- 所有者名・作成日時・ステータス

**所有者のみ表示（`IsOwner = true`）：**
- 説明文編集フォーム（`hx-post="/items/{id}"`）
- 画像追加アップロード
- 各画像の削除ボタン（`hx-post="/items/images/{id}/delete"`, `hx-confirm`付き）

---

### admin_users.html

**テンプレートデータ：**
```go
map[string]any{
    "User":       *models.User,
    "Users":      []models.User,
    "Department": models.Department,
}
```

**表示内容：**
- 部門名ヘッダー
- ユーザー一覧テーブル（表示名、ユーザー名、ロール）
- ユーザー作成フォーム（ユーザー名・パスワード・表示名）

---

### admin_item_form.html

管理者による代理アイテム登録フォーム。

**テンプレートデータ：**
```go
map[string]any{
    "User":  *models.User,
    "Users": []models.User,  // 部門内ユーザー一覧（selectボックス用）
}
```

**フィールド：**
- 所有者選択（`<select name="owner_id">`、部門内ユーザー一覧）
- タイトル（テキスト、必須）
- 説明（テキストエリア）
- 画像アップロード

---

### admin_dept_items.html

**テンプレートデータ：**
```go
map[string]any{
    "User":       *models.User,
    "Items":      []models.Item,
    "ImageMap":   map[int][]models.ItemImage,
    "Department": models.Department,
    "Query":      string,
    "Page":       int,
    "HasMore":    bool,
}
```

マーケット一覧と同様の検索・ページネーション構成。全ステータスのアイテムを表示。

---

### sysadmin_departments.html

**テンプレートデータ：**
```go
map[string]any{
    "User":        *models.User,
    "Departments": []models.Department,
}
```

**表示内容：**
- 部門一覧テーブル
- 部門作成フォーム（部門名のみ）
- 部門管理者作成フォーム（所属部門選択・ユーザー名・パスワード・表示名）

---

## テンプレートエンジン仕様

### テンプレート読み込み（`handlers/render.go: NewEnv`）

**起動時に全テンプレートをコンパイル済みとしてメモリにキャッシュ：**

```go
// layout使用ページ: layout.html + item_list_partial.html + 対象ページ の3ファイルをパース
template.Must(template.New("").Funcs(funcs).ParseFS(tmplFS, "layout.html", "item_list_partial.html", page))

// ログイン（スタンドアロン）
template.Must(template.New("").Funcs(funcs).ParseFS(tmplFS, "login.html"))

// パーシャル（HTMX用）
template.Must(template.New("").Funcs(funcs).ParseFS(tmplFS, "item_list_partial.html"))
```

**レンダリング：**
- layout使用ページ: `t.ExecuteTemplate(w, "layout.html", data)` → `{{template "content" .}}` 経由でページ固有コンテンツを呼び出す
- ログイン: `t.ExecuteTemplate(w, "login.html", data)`
- パーシャル: `t.ExecuteTemplate(w, "item_list_partial.html", data)`

### カスタムテンプレート関数（FuncMap）

| 関数名 | シグネチャ | 説明 |
|--------|-----------|------|
| `dict` | `dict(key, val, ...)` → `map[string]any` | テンプレート内でmapを生成（テンプレートへのデータ受け渡し用） |
| `add` | `add(a, b int)` → `int` | テンプレート内の整数加算（ページネーションの次ページ番号計算） |
| `images` | `images(imageMap, itemID)` → `[]ItemImage` | `ImageMap[itemID]` を取得する糖衣構文 |

---

## UIコンポーネント

### トースト通知

**実装：** Alpine.js + HTMX HX-Triggerブリッジ

```
[サーバー]
  HX-Trigger: {"showMessage": "メッセージ"}

[ブラウザ: layout.htmlのscriptタグ]
  htmx:afterRequest イベントを監視
  → HX-Triggerヘッダーをパース
  → showMessage があれば window.dispatchEvent('show-message', ...) を発火

[Alpine.js: <body x-data>]
  @show-message.window でイベントを受信
  → toast にメッセージをセット
  → showToast = true
  → 3000ms後に showToast = false（自動消去）
```

表示スタイル: 画面右下に固定表示（`fixed bottom-6 right-6`）、インディゴ背景、フェードトランジション

### ページネーション

```html
<div hx-get="/market?page={{add .Page 1}}&q={{.Query}}"
     hx-target="#item-list"
     hx-push-url="true">次へ</div>
```

- `hx-target="#item-list"` で一覧部分のみ更新
- `hx-push-url="true"` でブラウザのURLバーを更新（ブックマーク・戻るボタン対応）

### 確認ダイアログ

```html
<button hx-post="/items/put-on-market"
        hx-confirm="マーケットに出品しますか？">出品</button>
```

HTMXの `hx-confirm` 属性でブラウザネイティブの確認ダイアログを表示。

---

## 静的ファイル管理

| ファイル | 説明 |
|---------|------|
| `static/style.css` | Tailwind CSSのコンパイル済みCSS（ミニファイ）。Goバイナリに埋め込み |
| `static/input.css` | Tailwindのソースファイル（`@import "tailwindcss"`のみ記述） |
| `static/htmx.min.js` | HTMXライブラリ本体（ローカル配置） |
| `static/alpine.min.js` | Alpine.jsライブラリ本体（ローカル配置） |

`//go:embed static/*` によりGoバイナリに埋め込まれる。`/static/*` パスで配信。
