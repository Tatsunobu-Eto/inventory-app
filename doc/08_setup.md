# セットアップ・起動手順

## 前提条件

### 推奨（Docker Compose）
- Docker
- Docker Compose

### ローカル開発（Dockerなし）
- Go 1.23 以上
- Node.js / npm（Tailwind CSS のコンパイルに必要）
- PostgreSQL（ローカルで起動済みであること）

---

## 1. 環境変数の設定（.env）

プロジェクトルートに `.env` ファイルを作成する。

| 変数名 | 必須 | 説明 |
|--------|------|------|
| `DATABASE_URL` | ✅ | PostgreSQL 接続文字列 |
| `SESSION_KEY` | 推奨 | セッション署名キー（未設定時は `"default-dev-key-change-me!!"` が使われる） |
| `INIT_SYSADMIN_USER` | 初回のみ | 初期 sysadmin ユーザー名（sysadmin が0人のときのみ有効） |
| `INIT_SYSADMIN_PASS` | 初回のみ | 初期 sysadmin パスワード |
| `PORT` | - | リッスンポート（未設定時は `8081`） |

### .env サンプル（Docker Compose用）

```env
DATABASE_URL=postgres://postgres:postgres@db:5432/inventory?sslmode=disable
SESSION_KEY=ここに32文字以上のランダムな文字列を入れる
INIT_SYSADMIN_USER=sysadmin
INIT_SYSADMIN_PASS=changeme_at_first_login
PORT=8080
```

> **注意：** `SESSION_KEY` は本番環境では必ずランダムな強力な文字列を設定すること。デフォルト値のまま本番運用しないこと。

---

## 2. Docker Compose で起動（推奨）

```bash
docker-compose up --build
```

- `db` コンテナ（postgres:17-alpine）がヘルスチェックをパスしてから `app` コンテナが起動する
- アプリ起動時にマイグレーションが自動実行される
- sysadmin が 0 人の場合のみ `INIT_SYSADMIN_USER` / `INIT_SYSADMIN_PASS` からアカウントが作成される
- 起動後は http://localhost:8080 でアクセス可能

### バックグラウンドで起動する場合

```bash
docker-compose up --build -d
```

### 停止

```bash
docker-compose down       # コンテナ停止（データは保持）
docker-compose down -v    # コンテナ停止 + ボリューム削除（全データ消滅）
```

---

## 3. 初回ログインと初期設定

1. http://localhost:8080/login を開く
2. `.env` に設定した `INIT_SYSADMIN_USER` / `INIT_SYSADMIN_PASS` でログイン
3. `/sysadmin/departments` で部門を作成する
4. 同画面の「部門管理者を作成」フォームで admin アカウントを作成する
5. sysadmin でログアウトし、admin でログインする
6. `/admin/users` で一般ユーザー（user）を作成する

```
sysadmin ログイン
  └─ /sysadmin/departments → 部門作成
  └─ /sysadmin/departments → admin アカウント作成

admin ログイン
  └─ /admin/users → user アカウント作成

user ログイン
  └─ /items/new → アイテム登録
  └─ /market   → マーケット利用
```

> **セキュリティ注意：** 初回ログイン後に `.env` の `INIT_SYSADMIN_PASS` を変更しても再適用されない（冪等性のため）。パスワードを変更したい場合は DB を直接操作する（現バージョンにパスワード変更 UI はない）。詳細は [09_known_limitations.md](09_known_limitations.md) §2 を参照。

---

## 4. ローカル開発（Docker なし）

### .env の変更点

`DATABASE_URL` をローカル PostgreSQL に向ける：

```env
DATABASE_URL=postgres://postgres:password@localhost:5432/inventory?sslmode=disable
PORT=8080
```

### 起動手順

```bash
# 1. Tailwind CSS をコンパイル（static/style.css を生成）
make tailwind

# 2. Go アプリを起動
make run
```

または個別に実行：

```bash
npm install
npm run tailwind
go run .
```

### Tailwind CSS の開発中ウォッチ（別ターミナルで実行）

```bash
npm run tailwind:watch
```

テンプレートファイルを変更した際に自動で CSS を再コンパイルする。

---

## 5. Makefile コマンド一覧

| コマンド | 説明 |
|---------|------|
| `make tailwind` | Tailwind CSS をコンパイルして `static/style.css` を生成 |
| `make build` | Tailwind コンパイル後に Go バイナリ（`inventory.exe`）をビルド |
| `make run` | `go run .` でアプリを起動（ビルドなし） |
| `make clean` | `inventory.exe` と `static/style.css` を削除 |

---

## 6. データの永続化

| データ | 保存先 | 備考 |
|--------|--------|------|
| PostgreSQL データ | Docker ボリューム `pgdata` | `docker-compose down` では削除されない |
| アップロード画像 | ホストの `./uploads/` | `docker-compose.yml` でマウント（`./uploads:/app/uploads`） |

- `docker-compose down -v` を実行すると `pgdata` ボリュームが削除され、**全 DB データが消滅する**
- `uploads/` ディレクトリはホスト側に残るため、ボリューム削除後もファイルは残る

---

## 7. 本番環境の考慮事項

| 項目 | 対応 |
|------|------|
| `SESSION_KEY` | 必ずランダムな 32 文字以上の文字列を設定すること |
| 初期パスワード | `INIT_SYSADMIN_PASS` は初回セットアップ後に強力なパスワードへ変更（DB 直接操作） |
| HTTPS | アプリ自体は HTTP のみ。本番ではリバースプロキシ（nginx 等）で TLS を終端すること |
| `uploads/` | コンテナ外にボリュームマウントし、バックアップ対象に含めること |
| ログ | 標準出力にログが出力される。Docker の場合は `docker-compose logs -f app` で確認 |
