-- Write your migrate up statements here

ALTER TABLE Item ADD COLUMN IsPDFItem varchar(255);
ALTER TABLE Item ADD COLUMN PDF integer REFERENCES PDF(ID);


---- create above / drop below ----

ALTER TABLE Item DROP COLUMN IsPDFItem;
ALTER TABLE Item DROP COLUMN PDF;


-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
