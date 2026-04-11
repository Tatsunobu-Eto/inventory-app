-- サンプルデータの削除（依存関係の逆順）
DELETE FROM transactions WHERE item_id IN (SELECT id FROM items WHERE title IN (
    'ThinkPad X1 Carbon', 'Dell 27インチモニター'
));

DELETE FROM item_images WHERE item_id IN (SELECT id FROM items WHERE created_by IN (
    SELECT id FROM users WHERE username IN (
        'tanaka','yamada','sato','watanabe','kobayashi','nakamura','kato','it_admin','sales_admin'
    )
));

DELETE FROM items WHERE created_by IN (
    SELECT id FROM users WHERE username IN (
        'tanaka','yamada','sato','watanabe','kobayashi','nakamura','kato','it_admin','sales_admin'
    )
);

DELETE FROM users WHERE username IN (
    'sysadmin','it_admin','sales_admin','tanaka','yamada','sato','watanabe','kobayashi','nakamura','kato'
);

DELETE FROM departments WHERE name IN ('情報システム部','営業部','総務部','開発部');
