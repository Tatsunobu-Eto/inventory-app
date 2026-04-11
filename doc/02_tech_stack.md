# 技術スタック

## バックエンド

| 技術 | バージョン | 用途 |
|------|-----------|------|
| Go | 1.23.0 | アプリケーション本体 |
| github.com/go-chi/chi/v5 | v5.2.1 | HTTPルーター・ミドルウェアチェーン |
| github.com/golang-migrate/migrate/v4 | v4.18.3 | DBマイグレーション管理 |
| github.com/gorilla/sessions | v1.4.0 | セッション管理（Cookieベース） |
| github.com/gorilla/securecookie | v1.1.2 | セキュアCookie署名（gorilla/sessionsの依存） |
| github.com/joho/godotenv | v1.5.1 | `.env`ファイルの読み込み |
| github.com/lib/pq | v1.10.9 | PostgreSQL ドライバ |
| golang.org/x/crypto | v0.37.0 | bcryptによるパスワードハッシュ |
| go.uber.org/atomic | v1.7.0 | アトミック操作（migrate依存） |
| github.com/hashicorp/go-multierror | v1.1.1 | マルチエラー集約（migrate依存） |

### Go標準ライブラリ（主要利用）

| パッケージ | 用途 |
|-----------|------|
| `database/sql` | DBコネクション管理・クエリ実行 |
| `embed` | テンプレート・静的ファイル・マイグレーションをバイナリに埋め込み |
| `html/template` | サーバーサイドHTMLレンダリング |
| `net/http` | HTTPサーバー・ファイルサーバー |
| `io/fs` | 組み込みファイルシステム操作 |
| `os` | ファイルI/O（画像保存・削除） |
| `time` | タイムスタンプ・バックグラウンドジョブのTickerなど |
| `strconv` | URLパラメータの文字列→int変換 |
| `path/filepath` | OS依存ファイルパス結合 |
| `log` | サーバーログ出力 |
| `context` | リクエストコンテキストへのユーザー情報注入 |

## フロントエンド

| 技術 | バージョン | 用途 |
|------|-----------|------|
| Tailwind CSS | v4 | ユーティリティファーストCSSフレームワーク。`input.css`をソースとして`style.css`にコンパイル済み |
| HTMX | （ローカル静的ファイル） | HTMLのdata属性によるAjax・ページ部分更新・フォーム送信 |
| Alpine.js | （ローカル静的ファイル） | リアクティブなUI動作（トースト通知など） |

### フロントエンド詳細

**Tailwind CSS v4**
- `static/input.css` に `@import "tailwindcss"` を記述
- `npx @tailwindcss/cli -i ./static/input.css -o ./static/style.css` でコンパイル
- コンパイル済み `style.css` をGoバイナリに埋め込み

**HTMX**
- サーバーへのAjaxリクエストを HTML 属性のみで記述
- 主な利用パターン：
  - `hx-post` / `hx-get` でフォーム送信・ページネーション
  - `hx-confirm` で確認ダイアログ
  - `HX-Redirect` レスポンスヘッダーでリダイレクト
  - `HX-Refresh: true` でページリロード
  - `HX-Trigger` でカスタムイベント発火（トースト通知）

**Alpine.js**
- `x-data` / `x-show` / `x-on` でリアクティブなコンポーネント定義
- `HX-Trigger` ヘッダーで発火した `showToast` イベントをAlpine.jsが受信し、トースト表示を制御

## データベース

| 技術 | バージョン | 用途 |
|------|-----------|------|
| PostgreSQL | 17-alpine | メインデータベース（Docker） |

## インフラ・ビルド

| 技術 | バージョン | 用途 |
|------|-----------|------|
| Docker | - | コンテナイメージビルド・実行 |
| Docker Compose | - | アプリ＋DB のオーケストレーション |
| golang:1.23-alpine | - | ビルドステージのベースイメージ |
| alpine:latest | - | 実行ステージのベースイメージ（軽量） |
| tzdata, ca-certificates | - | タイムゾーンとTLS証明書（alpine実行イメージに追加） |

## 開発ツール（Node.js）

| ツール | 用途 |
|--------|------|
| tailwindcss | CSSコンパイル（devDependency） |
| @tailwindcss/cli | TailwindのCLIツール |

`package.json` はTailwind CSSのコンパイルのみを目的としており、JavaScriptのビルドパイプラインには使用しない。

## モジュール構成 (go.mod)

```
module inventory-app
go 1.23.0
```

モジュール名は `inventory-app`。全内部パッケージは `inventory-app/handlers`、`inventory-app/models`、`inventory-app/middleware` の形式でインポートする。
