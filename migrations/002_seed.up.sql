-- ============================================================
-- サンプルデータ投入
-- パスワードはすべて "password" (bcrypt, cost=10)
-- ============================================================

-- departments
INSERT INTO departments (name) VALUES
    ('情報システム部'),
    ('営業部'),
    ('総務部'),
    ('開発部')
ON CONFLICT (name) DO NOTHING;

-- users
-- role: sysadmin / admin / user
-- password_hash は "password" の bcrypt ハッシュ
INSERT INTO users (department_id, username, password_hash, display_name, role) VALUES
    (NULL,                          'sysadmin',   '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'システム管理者',   'sysadmin'),
    ((SELECT id FROM departments WHERE name='情報システム部'), 'it_admin',    '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', '伊藤 健太',       'admin'),
    ((SELECT id FROM departments WHERE name='営業部'),         'sales_admin', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', '鈴木 花子',       'admin'),
    ((SELECT id FROM departments WHERE name='情報システム部'), 'tanaka',      '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', '田中 太郎',       'user'),
    ((SELECT id FROM departments WHERE name='情報システム部'), 'yamada',      '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', '山田 次郎',       'user'),
    ((SELECT id FROM departments WHERE name='営業部'),         'sato',        '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', '佐藤 三郎',       'user'),
    ((SELECT id FROM departments WHERE name='営業部'),         'watanabe',    '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', '渡辺 美咲',       'user'),
    ((SELECT id FROM departments WHERE name='総務部'),         'kobayashi',   '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', '小林 誠',         'user'),
    ((SELECT id FROM departments WHERE name='開発部'),         'nakamura',    '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', '中村 優子',       'user'),
    ((SELECT id FROM departments WHERE name='開発部'),         'kato',        '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', '加藤 翔',         'user')
ON CONFLICT (username) DO NOTHING;

-- items
INSERT INTO items (department_id, title, description, owner_id, created_by, status, market_at, created_at) VALUES
    (
        (SELECT id FROM departments WHERE name='情報システム部'),
        'ThinkPad X1 Carbon',
        '2022年購入のノートPC。バッテリー良好、傷なし。',
        (SELECT id FROM users WHERE username='tanaka'),
        (SELECT id FROM users WHERE username='tanaka'),
        'market',
        NOW() - INTERVAL '3 days',
        NOW() - INTERVAL '30 days'
    ),
    (
        (SELECT id FROM departments WHERE name='情報システム部'),
        'Dell 27インチモニター',
        '4K対応モニター。スタンド付き。',
        (SELECT id FROM users WHERE username='yamada'),
        (SELECT id FROM users WHERE username='yamada'),
        'market',
        NOW() - INTERVAL '1 day',
        NOW() - INTERVAL '20 days'
    ),
    (
        (SELECT id FROM departments WHERE name='営業部'),
        'iPhone 13 Pro',
        '会社支給スマートフォン。画面保護フィルム付き。',
        (SELECT id FROM users WHERE username='sato'),
        (SELECT id FROM users WHERE username='sato'),
        'private',
        NULL,
        NOW() - INTERVAL '15 days'
    ),
    (
        (SELECT id FROM departments WHERE name='営業部'),
        'キャノン複合機 MF644Cdw',
        'カラーレーザープリンター。トナー残量80%。',
        (SELECT id FROM users WHERE username='watanabe'),
        (SELECT id FROM users WHERE username='sales_admin'),
        'market',
        NOW() - INTERVAL '7 days',
        NOW() - INTERVAL '60 days'
    ),
    (
        (SELECT id FROM departments WHERE name='総務部'),
        'オフィスチェア（エルゴノミクス）',
        'ハーマンミラー アーロンチェア。Bサイズ。状態良好。',
        (SELECT id FROM users WHERE username='kobayashi'),
        (SELECT id FROM users WHERE username='kobayashi'),
        'market',
        NOW() - INTERVAL '2 days',
        NOW() - INTERVAL '45 days'
    ),
    (
        (SELECT id FROM departments WHERE name='開発部'),
        'iPad Pro 12.9インチ',
        'M2チップ搭載。Apple Pencil第2世代付属。',
        (SELECT id FROM users WHERE username='nakamura'),
        (SELECT id FROM users WHERE username='nakamura'),
        'private',
        NULL,
        NOW() - INTERVAL '10 days'
    ),
    (
        (SELECT id FROM departments WHERE name='開発部'),
        'Raspberry Pi 4 Model B 8GB',
        'スターターキット付き。SDカード32GB同梱。',
        (SELECT id FROM users WHERE username='kato'),
        (SELECT id FROM users WHERE username='kato'),
        'market',
        NOW() - INTERVAL '5 days',
        NOW() - INTERVAL '25 days'
    ),
    (
        (SELECT id FROM departments WHERE name='情報システム部'),
        'Logicool MX Keys キーボード',
        'Bluetooth・USBレシーバー両対応。英字配列。',
        (SELECT id FROM users WHERE username='tanaka'),
        (SELECT id FROM users WHERE username='tanaka'),
        'deleted',
        NULL,
        NOW() - INTERVAL '90 days'
    )
;

-- item_images（各アイテムにサンプル画像パスを紐付け）
INSERT INTO item_images (item_id, file_path)
SELECT id, 'uploads/sample_' || id || '_1.jpg' FROM items
UNION ALL
SELECT id, 'uploads/sample_' || id || '_2.jpg' FROM items WHERE status = 'market'
ORDER BY 1;

-- transactions（market 状態の物品に対して取引履歴を作成）
INSERT INTO transactions (item_id, from_user_id, to_user_id, created_at)
VALUES
    (
        (SELECT id FROM items WHERE title='ThinkPad X1 Carbon'),
        (SELECT id FROM users WHERE username='tanaka'),
        (SELECT id FROM users WHERE username='nakamura'),
        NOW() - INTERVAL '2 days'
    ),
    (
        (SELECT id FROM items WHERE title='Dell 27インチモニター'),
        (SELECT id FROM users WHERE username='yamada'),
        (SELECT id FROM users WHERE username='kato'),
        NOW() - INTERVAL '12 hours'
    );
