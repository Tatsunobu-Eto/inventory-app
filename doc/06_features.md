# 機能仕様

## 1. アイテム管理

### 1.1 アイテム登録

**対象：** user, admin

一般ユーザーが自分名義でアイテムを登録する。

**処理フロー：**
```
1. GET /items/new でフォーム表示
2. POST /items/new にmultipart/form-dataを送信
3. タイトル空チェック → 空の場合はエラートースト + 422
4. models.CreateItem でDBにアイテム挿入
   - department_id: ログインユーザーの部門
   - owner_id / created_by: ログインユーザー
   - status: 'private'（デフォルト）
5. 画像ファイルがあれば SaveUploadedImages で保存
6. 成功: HX-Redirect: /my-items
```

**部門未所属のチェック：** ログインユーザーの `department_id` が NULL の場合は 403（sysadminはアイテム登録不可）

### 1.2 アイテム代理登録（admin専用）

**対象：** admin

部門管理者が他のユーザー名義でアイテムを登録する。

**処理フロー：**
```
1. GET /admin/items/new でフォーム表示
   - 自部門のユーザー一覧をselectボックスに表示
2. POST /admin/items にmultipart/form-dataを送信
3. タイトル・owner_id チェック → いずれか空の場合はエラートースト + 422
4. models.CreateItem でDBにアイテム挿入
   - owner_id: 選択されたユーザーID
   - created_by: 実行adminのID（≠ owner_id）
5. 成功: HX-Redirect: /admin/items/new（連続登録のため同ページへ戻る）
```

### 1.3 アイテム一覧

#### マイアイテム（GET /my-items）

- ログインユーザーが `owner_id` のアイテムを全件取得
- ステータスフィルタなし（private/market/deletedすべて表示）
- ソート: 登録日時降順

#### マーケット一覧（GET /market）

- ログインユーザーの部門の `status = 'market'` のアイテムを取得
- ページネーション: 1ページ20件
- 検索: タイトル・説明文のILIKE（部分一致、大文字小文字無視）
- ソート: 登録日時降順

#### 管理者アイテム一覧（GET /admin/items）

- ログインadminの部門の全ステータスのアイテムを取得
- ページネーション: 1ページ20件
- 検索: タイトル・説明文のILIKE

### 1.4 アイテム詳細（GET /items/{item_id}）

- アイテム情報・画像一覧を表示
- `IsOwner` フラグ（ログインユーザー = owner_id）を使いテンプレートで表示を切り替え
- 所有者のみ編集フォームと画像削除ボタンを表示

### 1.5 アイテム更新（POST /items/{item_id}）

**権限チェック：** ログインユーザーが `owner_id` でなければ403

**更新可能フィールド：**
- `description`（説明文）
- `images`（画像追加）

**画像追加制限：**
```
既存画像枚数 + 新規追加枚数 > 5 の場合は 422 + エラートースト
例: 既存3枚 + 追加3枚 = 6枚 → エラー
    既存3枚 + 追加2枚 = 5枚 → OK
```

---

## 2. 画像アップロード

### 2.1 仕様

| 項目 | 仕様 |
|------|------|
| 最大ファイル数 | アイテムあたり5枚 |
| 最大ファイルサイズ | 10MB（`maxUploadSize = 10 << 20`） |
| 対応形式 | `.jpg`, `.jpeg`, `.png`, `.gif`, `.webp` |
| 保存先 | `uploads/{item_id}/{UnixNano}{拡張子}` |
| DB記録 | `item_images.file_path` に相対パスを保存（例: `42/1700000000000123456.jpg`） |

### 2.2 保存処理（`handlers/images.go`）

```
1. multipart/form-data の "images" フィールドを処理
2. 拡張子チェック（許可外の拡張子はスキップ）
3. uploads/{item_id}/ ディレクトリを os.MkdirAll で作成
4. ファイル名: time.Now().UnixNano() + 拡張子（ナノ秒で一意性確保）
5. DBパス: "item_id/filename"（スラッシュ区切り、URLセーフ）
6. ディスクへ書き込み（io.Copy）
7. models.CreateItemImage でDBに記録
```

### 2.3 画像削除（POST /items/images/{image_id}/delete）

```
1. image_id からDBのitem_images レコードを取得
2. item_id を使ってアイテムの owner_id を確認
3. ログインユーザーが所有者でなければ403
4. models.DeleteItemImage でDBレコード削除・file_pathを返却
5. os.Remove で uploads/{file_path} のファイルを削除
6. HX-Refresh: true でページリロード
```

### 2.4 画像配信

- `/uploads/*` パスで `http.FileServer(http.Dir("uploads"))` が静的ファイルを配信
- URL例: `http://localhost:8080/uploads/42/1700000000000123456.jpg`

---

## 3. マーケットプレイス

### 3.1 出品（POST /items/put-on-market）

**前提：** `status = 'private'` かつ `owner_id` がログインユーザー

```sql
UPDATE items SET status = 'market', market_at = NOW()
WHERE id = $1 AND owner_id = $2 AND status = 'private'
```

- `market_at` に出品日時を記録（90日期限の起算点）
- 所有者チェック・ステータスチェックはSQL条件で行う（アプリ層では別途チェックしない）

### 3.2 取り下げ（POST /items/withdraw）

**前提：** `status = 'market'` かつ `owner_id` がログインユーザー

```sql
UPDATE items SET status = 'private', market_at = NULL
WHERE id = $1 AND owner_id = $2 AND status = 'market'
```

- `market_at` を NULL にリセット

### 3.3 申請（POST /items/apply）

**最も重要な排他制御を伴う処理。**

```
処理フロー（models.ApplyForItem）:

1. db.Begin() でトランザクション開始
2. SELECT status, owner_id FROM items WHERE id = $1 FOR UPDATE
   → 行レベルロック取得（他のトランザクションの同アイテムへのアクセスをブロック）
3. status が 'market' でなければ（取り下げ・期限切れ・既応募）
   → tx.Rollback, return (false, nil)
4. UPDATE items SET status = 'private', owner_id = $1（申請者）, market_at = NULL
   → 所有権移転・マーケットから除外
5. INSERT INTO transactions (item_id, from_user_id, to_user_id)
   → 取引履歴記録
6. tx.Commit() でコミット
7. return (true, nil)
```

**競合ケース：**
- 複数ユーザーが同時に申請した場合、最初にFOR UPDATEロックを取得したトランザクションのみが成功
- 後続のトランザクションはstepの2でロック待ち → ロック解放後にstep 3でstatus != 'market' を検知 → false を返す
- ハンドラ側でfalseの場合は409 Conflictを返す

### 3.4 90日自動期限切れ（バックグラウンドジョブ）

**実行タイミング：** アプリ起動時にgoroutineで開始、1時間ごとに実行

```go
// main.go
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()
    for {
        n, err := models.ExpireMarketItems(db)
        // ...ログ出力
        <-ticker.C
    }
}()
```

**SQL：**
```sql
UPDATE items SET status = 'deleted'
WHERE status = 'market' AND market_at < NOW() - INTERVAL '90 days'
```

- `status` を `'deleted'` に変更（物理削除ではない）
- 起動直後にも1回実行される（Tickerの前にループ処理を記述しているため）

---

## 4. ユーザー管理

### 4.1 一般ユーザー作成（admin専用）

- `POST /admin/users`
- ロール: `user`、department_id: 実行adminの部門
- バリデーション: username・password 両方必須

### 4.2 部門管理者作成（sysadmin専用）

- `POST /sysadmin/admins`
- ロール: `admin`、department_id: フォームで選択
- バリデーション: username・password・department_id すべて必須

### 4.3 sysadmin初期作成

- アプリ起動時に自動実行（`main.go: seedSysadmin()`）
- sysadminが0人の場合のみ作成（冪等）
- department_id: NULL（部門なし）

---

## 5. 部門管理

### 5.1 部門作成（sysadmin専用）

- `POST /sysadmin/departments`
- バリデーション: name 必須（DB制約: UNIQUE）
- DB制約違反（重複）時は500エラー（現在はエラーメッセージのみ）

---

## 6. データアクセス層（models パッケージ）

### ItemFilter 構造体

```go
type ItemFilter struct {
    DepartmentID int    // 必須：部門による絞り込み
    Status       string // オプション：ステータス絞り込み（空文字で全ステータス）
    OwnerID      int    // オプション：所有者による絞り込み（0で絞り込みなし）
    Query        string // オプション：検索キーワード
    Limit        int    // オプション：取得件数上限（0で制限なし）
    Offset       int    // オプション：オフセット（ページネーション用）
}
```

### 動的クエリビルダー（models/item.go: ListItems）

条件に応じてWHERE句を動的に追加するパターン。パラメータインデックス（$1, $2...）を `itoa(n)` で管理。

```go
q := "SELECT ... FROM items i JOIN users u ON u.id = i.owner_id WHERE i.department_id = $1"
args := []any{f.DepartmentID}
n := 2

if f.Status != "" {
    q += " AND i.status = $" + itoa(n)
    args = append(args, f.Status)
    n++
}
// ... 以下同様
```
