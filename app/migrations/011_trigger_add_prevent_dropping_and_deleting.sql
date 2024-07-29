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



-- CREATE TRIGGER prevent_deleting_on_table_accounttype ON AccountType INSTEAD OF DELETE AS RAISERROR ('Cannot delete from table AccountType', 16, 2);
-- CREATE TRIGGER prevent_deleting_on_table_accounts ON Account INSTEAD OF DELETE AS RAISERROR ('Cannot delete from table Account', 16, 3);
-- CREATE TRIGGER prevent_deleting_on_table_item ON Item INSTEAD OF DELETE AS RAISERROR ('Cannot delete from table Item', 16, 4);
-- CREATE TRIGGER prevent_deleting_on_table_paymentorder ON PaymentOrder INSTEAD OF DELETE AS RAISERROR ('Cannot delete from table PaymentOrder', 16, 5);
-- CREATE TRIGGER prevent_deleting_on_table_orderentry ON OrderEntry INSTEAD OF DELETE AS RAISERROR ('Cannot delete from table OrderEntry', 16, 6);
-- CREATE TRIGGER prevent_deleting_on_table_payment ON Payment INSTEAD OF DELETE AS RAISERROR ('Cannot delete from table Payment', 16, 7);
-- CREATE TRIGGER prevent_deleting_on_table_settings ON Settings INSTEAD OF DELETE AS RAISERROR ('Cannot delete from table Settings', 16, 8);
-- CREATE TRIGGER prevent_deleting_on_table_dbsettings ON DBsettings INSTEAD OF DELETE AS RAISERROR ('Cannot drop table DBsettings', 16, 19);
-- CREATE TRIGGER prevent_deleting_on_table_pdf ON PDF INSTEAD OF DELETE AS RAISERROR ('Cannot delete from table PDF', 16, 9);
-- CREATE TRIGGER prevent_deleting_on_table_pdfdownloads ON PDFDownloads INSTEAD OF DELETE AS RAISERROR ('Cannot delete from table PDFDownloads', 16, 10);

-- CREATE TRIGGER prevent_dropping_on_table_vendor ON Vendor INSTEAD OF DROP AS RAISERROR ('Cannot drop table Vendor', 16, 11);
-- CREATE TRIGGER prevent_dropping_on_table_accounttype ON AccountType INSTEAD OF DROP AS RAISERROR ('Cannot drop table AccountType', 16, 12);
-- CREATE TRIGGER prevent_dropping_on_table_accounts ON Account INSTEAD OF DROP AS RAISERROR ('Cannot drop table Account', 16, 13);
-- CREATE TRIGGER prevent_dropping_on_table_item ON Item INSTEAD OF DROP AS RAISERROR ('Cannot drop table Item', 16, 14);
-- CREATE TRIGGER prevent_dropping_on_table_paymentorder ON PaymentOrder INSTEAD OF DROP AS RAISERROR ('Cannot drop table PaymentOrder', 16, 15);
-- CREATE TRIGGER prevent_dropping_on_table_orderentry ON OrderEntry INSTEAD OF DROP AS RAISERROR ('Cannot drop table OrderEntry', 16, 16);
-- CREATE TRIGGER prevent_dropping_on_table_payment ON Payment INSTEAD OF DROP AS RAISERROR ('Cannot drop table Payment', 16, 17);
-- CREATE TRIGGER prevent_dropping_on_table_settings ON Settings INSTEAD OF DROP AS RAISERROR ('Cannot drop table Settings', 16, 18);
-- CREATE TRIGGER prevent_dropping_on_table_dbsettings ON DBsettings INSTEAD OF DROP AS RAISERROR ('Cannot drop table DBsettings', 16, 19);
-- CREATE TRIGGER prevent_dropping_on_table_pdf ON PDF INSTEAD OF DROP AS RAISERROR ('Cannot drop table PDF', 16, 19);
-- CREATE TRIGGER prevent_dropping_on_table_pdfdownloads ON PDFDownloads INSTEAD OF DROP AS RAISERROR ('Cannot drop table PDFDownloads', 16, 20);


---- create above / drop below ----




DROP TRIGGER prevent_deleting_on_table_vendor;
-- DROP TRIGGER prevent_deleting_on_table_accounttype;
-- DROP TRIGGER prevent_deleting_on_table_accounts;
-- DROP TRIGGER prevent_deleting_on_table_item;
-- DROP TRIGGER prevent_deleting_on_table_paymentorder;
-- DROP TRIGGER prevent_deleting_on_table_orderentry;
-- DROP TRIGGER prevent_deleting_on_table_payment;
-- DROP TRIGGER prevent_deleting_on_table_settings;
-- DROP TRIGGER prevent_deleting_on_table_dbsettings;
-- DROP TRIGGER prevent_deleting_on_table_pdf;
-- DROP TRIGGER prevent_deleting_on_table_pdfdownloads;

-- DROP TRIGGER prevent_dropping_on_table_vendor ON Vendor;
-- DROP TRIGGER prevent_dropping_on_table_accounttype ON AccountType;
-- DROP TRIGGER prevent_dropping_on_table_accounts ON Account;
-- DROP TRIGGER prevent_dropping_on_table_item ON Item;
-- DROP TRIGGER prevent_dropping_on_table_paymentorder ON PaymentOrder;
-- DROP TRIGGER prevent_dropping_on_table_orderentry ON OrderEntry;
-- DROP TRIGGER prevent_dropping_on_table_payment ON Payment;
-- DROP TRIGGER prevent_dropping_on_table_settings ON Settings;
-- DROP TRIGGER prevent_dropping_on_table_dbsettings ON DBsettings;
-- DROP TRIGGER prevent_dropping_on_table_pdf ON PDF;
-- DROP TRIGGER prevent_dropping_on_table_pdfdownloads ON PDFDownloads;



-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.