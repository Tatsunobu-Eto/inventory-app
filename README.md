# 消耗品管理システム (Inventory App)

社内の備品・消耗品を管理し、部署間などでの不要品の譲渡（マーケット機能）を行うことができるシステムです。Goと言語、PostgreSQL、HTMX/Alpine.jsを用いて構築されています。

## 主な機能

*   **ユーザー権限管理:** 
    *   一般ユーザー (`user`): 自身のアイテム管理、マーケットへの出品や応募。
    *   部門管理者 (`admin`): 所属部門のユーザー管理や消耗品の登録。
    *   システム管理者 (`sysadmin`): 部門の作成、部門管理者の任命、全アイテムや全ユーザーの管理。
*   **ダッシュボード:** 自身の所有アイテムや状態の一覧表示。
*   **マーケット機能:** 不要になった社内備品を出品し、他のユーザーが応募して受け取れる社内フリマ機能。
*   **取引履歴・承認フロー:** マーケットでの応募・譲渡を管理・承認するフロー。

## 技術スタック

*   **バックエンド:** Go 1.23.0
    *   **ルーティング:** [go-chi/chi](https://github.com/go-chi/chi)
    *   **セッション管理:** [gorilla/sessions](https://github.com/gorilla/sessions)
    *   **DBマイグレーション:** [golang-migrate](https://github.com/golang-migrate/migrate)
*   **データベース:** PostgreSQL 17
*   **フロントエンド:** HTMLテンプレート (Go `html/template`)
    *   [HTMX](https://htmx.org/) (非同期UI更新)
    *   [Alpine.js](https://alpinejs.dev/) (UI状態管理)
*   **インフラ/環境:** Docker, Docker Compose

## 開発環境のセットアップと起動方法

本システムはDockerを利用して素早く開発環境を構築できます。

1.  **環境変数の設定**
    `.env.example` をコピーして `.env` ファイルを作成します。
    ```bash
    cp .env.example .env
    ```
    ※`.env`の `SESSION_KEY` や初期システム管理者のパスワード(`INIT_SYSADMIN_PASS`)等を適宜変更してください。

2.  **Docker Composeによる起動**
    ```bash
    docker compose up -d
    ```
    これにより、PostgreSQLのデータベース(`shomohin_db`)とアプリケーション(`shomohin_app`)のコンテナが立ち上がります。

3.  **アクセス**
    ブラウザで `http://localhost:8080` にアクセスしてください。
    初回起動時、`.env`に設定した `INIT_SYSADMIN_USER` と `INIT_SYSADMIN_PASS` の情報で初期システム管理者アカウントが自動的に作成されますので、そのアカウントでログインが可能です。

## プロジェクト構成

```text
inventory-app/
├── main.go             # アプリケーションのエントリポイント・ルーティング
├── handlers/           # HTTPハンドラー (コントローラー)
├── middleware/         # 認証・権限管理等のミドルウェア
├── models/             # データベース連携・ビジネスロジック
├── migrations/         # PostgreSQLマイグレーションSQLファイル
├── templates/          # Go HTMLテンプレート (.html)
├── static/             # 静的ファイル (CSS, JS, アイコン等)
├── docker-compose.yml  # コンテナ環境の起動構成
└── Dockerfile          # アプリケーションのコンテナビルド定義
```

## その他特記事項

*   **静的アセットのバイナリ埋め込み:** `templates` フォルダおよび `static` フォルダ内のファイルは `go:embed` を用いてバイナリに埋め込まれており、配布・デプロイが容易な設計となっています。
*   **画像の保存:** アップロードされたアイテム画像は `uploads/` フォルダに保存され、ホストOSとのVolumeマウントにより永続化されます。
