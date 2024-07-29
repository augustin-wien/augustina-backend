-- Write your migrate up statements here
CREATE OR REPLACE FUNCTION prevent_delete_vendor()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'Cannot delete from table Vendor';
    -- This will prevent the delete operation
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_deleting_on_table_vendor
BEFORE DELETE ON Vendor
FOR EACH ROW
EXECUTE FUNCTION prevent_delete_vendor();

-- Same for Account, Item, PaymentOrder, OrderEntry, Payment, Settings, DBsettings, PDF, PDFDownloads

CREATE OR REPLACE FUNCTION prevent_delete_accounts()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'Cannot delete from table Account';
    -- This will prevent the delete operation
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_deleting_on_table_accounts
BEFORE DELETE ON Account
FOR EACH ROW
EXECUTE FUNCTION prevent_delete_accounts();

CREATE OR REPLACE FUNCTION prevent_delete_item()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'Cannot delete from table Item';
    -- This will prevent the delete operation
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_deleting_on_table_item
BEFORE DELETE ON Item
FOR EACH ROW
EXECUTE FUNCTION prevent_delete_item();

CREATE OR REPLACE FUNCTION prevent_delete_paymentorder()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'Cannot delete from table PaymentOrder';
    -- This will prevent the delete operation
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_deleting_on_table_paymentorder
BEFORE DELETE ON PaymentOrder
FOR EACH ROW
EXECUTE FUNCTION prevent_delete_paymentorder();

CREATE OR REPLACE FUNCTION prevent_delete_orderentry()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'Cannot delete from table OrderEntry';
    -- This will prevent the delete operation
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_deleting_on_table_orderentry
BEFORE DELETE ON OrderEntry
FOR EACH ROW
EXECUTE FUNCTION prevent_delete_orderentry();

CREATE OR REPLACE FUNCTION prevent_delete_payment()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'Cannot delete from table Payment';
    -- This will prevent the delete operation
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_deleting_on_table_payment
BEFORE DELETE ON Payment
FOR EACH ROW
EXECUTE FUNCTION prevent_delete_payment();

CREATE OR REPLACE FUNCTION prevent_delete_settings()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'Cannot delete from table Settings';
    -- This will prevent the delete operation
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_deleting_on_table_settings
BEFORE DELETE ON Settings
FOR EACH ROW
EXECUTE FUNCTION prevent_delete_settings();

CREATE OR REPLACE FUNCTION prevent_delete_dbsettings()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'Cannot delete from table DBsettings';
    -- This will prevent the delete operation
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_deleting_on_table_dbsettings
BEFORE DELETE ON DBsettings
FOR EACH ROW
EXECUTE FUNCTION prevent_delete_dbsettings();

CREATE OR REPLACE FUNCTION prevent_delete_pdf()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'Cannot delete from table PDF';
    -- This will prevent the delete operation
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_deleting_on_table_pdf
BEFORE DELETE ON PDF
FOR EACH ROW
EXECUTE FUNCTION prevent_delete_pdf();

-- CREATE OR REPLACE FUNCTION prevent_delete_pdfdownloads()
-- RETURNS trigger AS $$
-- BEGIN
--     RAISE EXCEPTION 'Cannot delete from table PDFDownloads';
--     -- This will prevent the delete operation
--     RETURN NULL;
-- END;
-- $$ LANGUAGE plpgsql;

-- CREATE TRIGGER prevent_deleting_on_table_pdfdownloads
-- BEFORE DELETE ON PDFDownloads
-- FOR EACH ROW
-- EXECUTE FUNCTION prevent_delete_pdfdownloads();

---- create above / drop below ----


DROP TRIGGER prevent_deleting_on_table_vendor ON Vendor;
DROP TRIGGER prevent_deleting_on_table_accounts ON Account;
DROP TRIGGER prevent_deleting_on_table_item ON Item;
DROP TRIGGER prevent_deleting_on_table_paymentorder ON PaymentOrder;
DROP TRIGGER prevent_deleting_on_table_orderentry ON OrderEntry;
DROP TRIGGER prevent_deleting_on_table_payment ON Payment;
DROP TRIGGER prevent_deleting_on_table_settings ON Settings;
DROP TRIGGER prevent_deleting_on_table_dbsettings ON DBsettings;
DROP TRIGGER prevent_deleting_on_table_pdf ON PDF;
DROP TRIGGER prevent_deleting_on_table_pdfdownloads ON PDFDownloads;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.