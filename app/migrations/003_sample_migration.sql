-- Write your migrate up statements here

-- Default value is 10000, which equals 100€
ALTER TABLE Settings
ADD COLUMN MaxOrderAmount integer DEFAULT 10000;

CREATE OR REPLACE FUNCTION truncate_tables(username IN VARCHAR) RETURNS void AS $$
DECLARE
    statements CURSOR FOR
        SELECT tablename FROM pg_tables
        WHERE tableowner = username AND schemaname = 'public' AND tablename != 'schema_version';
BEGIN
    FOR stmt IN statements LOOP
        EXECUTE 'TRUNCATE TABLE ' || quote_ident(stmt.tablename) || ' CASCADE;';
    END LOOP;
END;
$$ LANGUAGE plpgsql;

---- create above / drop below ----

ALTER TABLE Settings
DROP COLUMN MaxOrderAmount;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
