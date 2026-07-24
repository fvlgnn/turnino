ALTER TABLE queues
ADD COLUMN allow_custom_calls INTEGER NOT NULL DEFAULT 0
CHECK (allow_custom_calls IN (0, 1));
