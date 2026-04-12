ALTER TABLE transactions ADD COLUMN from_user_read BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE transactions SET from_user_read = TRUE;
