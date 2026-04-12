-- 期限切れ機能を廃止: market_at カラムと deleted ステータスを削除する。
-- 既存の deleted アイテムは物理廃棄済みとみなして削除する。

DELETE FROM items WHERE status = 'deleted';

ALTER TABLE items DROP COLUMN IF EXISTS market_at;

ALTER TABLE items DROP CONSTRAINT IF EXISTS items_status_check;
ALTER TABLE items ADD CONSTRAINT items_status_check
    CHECK (status IN ('private', 'market', 'applying'));
