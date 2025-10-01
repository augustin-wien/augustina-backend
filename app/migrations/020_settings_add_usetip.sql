-- Write your migrate up statements here

ALTER TABLE Settings ADD COLUMN usetipinsteadofdonation boolean NOT NULL DEFAULT false;

---- create above / drop below ----

ALTER TABLE Settings DROP COLUMN usetipinsteadofdonation;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
