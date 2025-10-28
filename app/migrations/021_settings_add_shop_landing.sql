-- Write your migrate up statements here

ALTER TABLE Settings ADD COLUMN shoplanding boolean NOT NULL DEFAULT false;

---- create above / drop below ----

ALTER TABLE Settings DROP COLUMN shoplanding;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
