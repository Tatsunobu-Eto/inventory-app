-- 期限切れ機能を復元: market_at カラムと deleted ステータスを追加する。

ALTER TABLE items DROP CONSTRAINT IF EXISTS items_status_check;
ALTER TABLE items ADD CONSTRAINT items_status_check
    CHECK (status IN ('private', 'market', 'applying', 'deleted'));

ALTER TABLE items ADD COLUMN IF NOT EXISTS market_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_items_market_at ON items(market_at) WHERE status = 'market';
