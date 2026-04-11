# システム概要

## システム名

**消耗品管理システム（ItamManagement）**

## 目的・背景

組織内の消耗品（備品・物品）を部門単位で管理し、不要になった消耗品を部門内で共有・譲渡できるマーケットプレイス機能を提供するWebアプリケーション。

- **問題意識：** 部門内で余剰になった消耗品が廃棄されたり、他のメンバーが同じものを新規購入してしまう無駄を防ぐ
- **解決策：** 消耗品の登録・マーケット出品・申請による所有権移転をオンラインで完結させる

## 対象ユーザー

| ロール | 対象者 | 主な用途 |
|--------|--------|---------|
| sysadmin（システム管理者） | ITシステム担当者 | 部門作成、管理者アカウント発行 |
| admin（部門管理者） | 各部門のリーダー等 | ユーザー管理、部門内消耗品の一括管理 |
| user（一般ユーザー） | 部門メンバー全員 | 消耗品の登録・マーケット出品・申請 |

## ユースケース概要

```
[sysadmin]
  └─ 部門を作成する
  └─ 部門管理者アカウントを作成する

[admin]
  └─ 一般ユーザーアカウントを作成する
  └─ 部門内全消耗品を検索・閲覧する
  └─ 代理で消耗品を登録する

[user]
  └─ 自分の消耗品を登録する（タイトル・説明・画像）
  └─ 消耗品をマーケットに出品する
  └─ マーケットから出品を取り下げる
  └─ 他ユーザーの出品に申請して所有権を取得する
```

## システムアーキテクチャ

```
┌─────────────────────────────────────────────────────┐
│                   ブラウザ (クライアント)              │
│  Tailwind CSS │ HTMX │ Alpine.js                    │
└────────────────────────┬────────────────────────────┘
                         │ HTTP (port 8080)
┌────────────────────────▼────────────────────────────┐
│               Goアプリケーション (Chi Router)         │
│                                                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │
│  │ handlers │  │middleware│  │    models        │  │
│  │(HTTPハンドラ)│  │(認証・認可)│  │(DBアクセス層)    │  │
│  └──────────┘  └──────────┘  └──────────────────┘  │
│                                                      │
│  ┌──────────────────────────────────────────────┐   │
│  │  Go html/template (サーバーサイドレンダリング)  │   │
│  └──────────────────────────────────────────────┘   │
└────────────────────────┬────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────┐
│              PostgreSQL 17 (データベース)              │
│  departments │ users │ items │ item_images │         │
│  transactions                                        │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│              ファイルシステム (uploads/)               │
│  アップロード画像: uploads/{item_id}/{timestamp}.ext  │
└─────────────────────────────────────────────────────┘
```

## 主要機能サマリー

| 機能カテゴリ | 機能名 | 概要 |
|------------|--------|------|
| 認証 | ログイン・ログアウト | セッションベースの認証 |
| アイテム管理 | アイテム登録 | タイトル・説明・画像（最大5枚）の登録 |
| アイテム管理 | アイテム編集 | 説明文・画像の更新（所有者のみ） |
| マーケット | 出品 | privateアイテムをマーケットに公開 |
| マーケット | 出品取り下げ | マーケット出品中のアイテムをprivateに戻す |
| マーケット | 申請 | 出品中アイテムに申請し所有権を取得 |
| マーケット | 自動期限切れ | 出品後90日経過で自動削除（毎時バックグラウンド処理） |
| ユーザー管理 | ユーザー作成 | admin/sysadminによるアカウント作成 |
| 部門管理 | 部門作成 | sysadminによる部門の追加 |
| 画像管理 | 画像アップロード | 最大5枚、10MB制限、jpg/png/gif/webp対応 |

## デプロイ構成

```
Docker Compose
├── db (postgres:17-alpine)  ポート: 内部のみ
└── app (Goアプリ)            ポート: 8080:8080
```

本番・開発いずれもDocker Composeで起動する想定。

## ディレクトリ構成

```
.
├── main.go                       # エントリポイント。ルーティング・起動・マイグレーション・seedSysadmin
├── go.mod / go.sum               # Go モジュール定義（モジュール名: inventory-app）
├── Dockerfile                    # マルチステージビルド（builder: golang:1.23-alpine / runner: alpine:latest）
├── docker-compose.yml            # app コンテナ + db コンテナ（postgres:17-alpine）
├── Makefile                      # tailwind / build / run / clean タスク
├── package.json                  # Tailwind CSS v4 のみ（JavaScript ビルドパイプラインではない）
│
├── handlers/                     # HTTP ハンドラ層
│   ├── auth.go                   # ログイン・ログアウト
│   ├── items.go                  # アイテム CRUD・マーケット操作（出品・取り下げ・申請）
│   ├── images.go                 # 画像アップロード・削除
│   ├── admin.go                  # admin 専用ハンドラ（ユーザー管理・代理登録）
│   └── render.go                 # テンプレートレンダリング・トースト通知ヘルパー
│
├── models/                       # DB アクセス層（database/sql を直接使用）
│   ├── item.go                   # アイテム CRUD・ListItems 動的クエリ・ApplyForItem 排他制御・ExpireMarketItems
│   ├── image.go                  # item_images テーブル操作
│   └── user.go                   # ユーザー CRUD・認証・CountSysadmins
│
├── middleware/                   # Chi ミドルウェア
│   └── auth.go                   # mw.Auth（認証）・mw.RequireRole（認可）・mw.CurrentUser
│
├── migrations/                   # golang-migrate の SQL マイグレーション（embed.FS でバイナリに埋め込み）
│   ├── 001_init.up.sql           # 全テーブル・インデックス作成
│   └── 001_init.down.sql         # 全テーブル削除（ロールバック用）
│
├── templates/                    # Go html/template ファイル（embed.FS でバイナリに埋め込み）
│   ├── layout.html               # 共通レイアウト（ナビ・トースト・HTMX/Alpine.js 読み込み）
│   ├── login.html                # スタンドアロン（layout を使わない）
│   ├── dashboard.html            # ダッシュボード
│   ├── market.html               # マーケット一覧
│   ├── item_list_partial.html    # マーケット・管理者一覧の HTMX 部分更新用パーシャル
│   ├── my_items.html             # マイアイテム一覧
│   ├── item_form.html            # アイテム登録フォーム
│   ├── item_detail.html          # アイテム詳細・編集
│   ├── admin_users.html          # ユーザー管理（admin）
│   ├── admin_item_form.html      # 代理登録フォーム（admin）
│   ├── admin_dept_items.html     # 部門アイテム一覧（admin）
│   └── sysadmin_departments.html # 部門管理（sysadmin）
│
├── static/                       # 静的ファイル（embed.FS でバイナリに埋め込み・/static/* で配信）
│   ├── input.css                 # Tailwind CSS ソース（@import "tailwindcss" のみ）
│   ├── style.css                 # コンパイル済み CSS（make tailwind で生成）
│   ├── htmx.min.js               # HTMX（ローカル配置）
│   └── alpine.min.js             # Alpine.js（ローカル配置）
│
├── uploads/                      # アップロード画像（実行時に動的生成・/uploads/* で配信）
│   └── {item_id}/                # アイテム ID ごとのサブディレクトリ
│       └── {UnixNano}.ext        # タイムスタンプをファイル名に使用（一意性確保）
│
└── doc/                          # 設計ドキュメント
    ├── 01_overview.md            # システム概要・アーキテクチャ（このファイル）
    ├── 02_tech_stack.md          # 技術スタック
    ├── 03_database.md            # DB 設計
    ├── 04_auth_roles.md          # 認証・認可
    ├── 05_api_routes.md          # API ルート仕様
    ├── 06_features.md            # 機能仕様
    ├── 07_ui_pages.md            # UI・画面仕様
    ├── 08_setup.md               # セットアップ・起動手順
    └── 09_known_limitations.md   # 既知の制限事項
```
