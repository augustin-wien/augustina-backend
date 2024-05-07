-- Write your migrate up statements here
CREATE TRIGGER prevent_deleting_column_in_table_account ON Account INSTEAD OF DELETE AS RAISERROR('You cannot delete from table account', 16, 1)
CREATE TRIGGER prevent_deleting_column_in_table_accounttype ON AccountType INSTEAD OF DELETE AS RAISERROR('You cannot delete from table accountType', 16, 2)
CREATE TRIGGER prevent_deleting_column_in_table_vendor ON Vendor INSTEAD OF DELETE AS RAISERROR('You cannot delete from table vendor', 16, 3)
CREATE TRIGGER prevent_deleting_column_in_table_item ON Item INSTEAD OF DELETE AS RAISERROR('You cannot delete from table item', 16, 4)
CREATE TRIGGER prevent_deleting_column_in_table_paymentorder ON PaymentOrder INSTEAD OF DELETE AS RAISERROR('You cannot delete from table paymentOrder', 16, 5)
CREATE TRIGGER prevent_deleting_column_in_table_orderentry ON OrderEntry INSTEAD OF DELETE AS RAISERROR('You cannot delete from table orderEntry', 16, 6)
CREATE TRIGGER prevent_deleting_column_in_table_payment ON Payment INSTEAD OF DELETE AS RAISERROR('You cannot delete from table payment', 16, 7)
CREATE TRIGGER prevent_deleting_column_in_table_settings ON Settings INSTEAD OF DELETE AS RAISERROR('You cannot delete from table settings', 16, 8)
CREATE TRIGGER prevent_deleting_column_in_table_pdf ON PDF INSTEAD OF DELETE AS RAISERROR('You cannot delete from table pdf', 16, 9)
CREATE TRIGGER prevent_deleting_column_in_table_pdfdownload ON PDFDownload INSTEAD OF DELETE AS RAISERROR('You cannot delete from table pdfDownload', 16, 10)

CREATE TRIGGER prevent_dropping_table_account ON Account INSTEAD OF DROP AS RAISERROR('You cannot drop table account', 16, 11)
CREATE TRIGGER prevent_dropping_table_accounttype ON AccountType INSTEAD OF DROP AS RAISERROR('You cannot drop table accountType', 16, 12)
CREATE TRIGGER prevent_dropping_table_vendor ON Vendor INSTEAD OF DROP AS RAISERROR('You cannot drop table vendor', 16, 13)
CREATE TRIGGER prevent_dropping_table_item ON Item INSTEAD OF DROP AS RAISERROR('You cannot drop table item', 16, 14)
CREATE TRIGGER prevent_dropping_table_paymentorder ON PaymentOrder INSTEAD OF DROP AS RAISERROR('You cannot drop table paymentOrder', 16, 15)
CREATE TRIGGER prevent_dropping_table_orderentry ON OrderEntry INSTEAD OF DROP AS RAISERROR('You cannot drop table orderEntry', 16, 16)
CREATE TRIGGER prevent_dropping_table_payment ON Payment INSTEAD OF DROP AS RAISERROR('You cannot drop table payment', 16, 17)
CREATE TRIGGER prevent_dropping_table_settings ON Settings INSTEAD OF DROP AS RAISERROR('You cannot drop table settings', 16, 18)
CREATE TRIGGER prevent_dropping_table_pdf ON PDF INSTEAD OF DROP AS RAISERROR('You cannot drop table pdf', 16, 19)
CREATE TRIGGER prevent_dropping_table_pdfdownload ON PDFDownload INSTEAD OF DROP AS RAISERROR('You cannot drop table pdfDownload', 16, 20)
---- create above / drop below ----

DROP TRIGGER IF EXISTS prevent_deleting_column_in_table_account
DROP TRIGGER IF EXISTS prevent_deleting_column_in_table_accounttype
DROP TRIGGER IF EXISTS prevent_deleting_column_in_table_vendor
DROP TRIGGER IF EXISTS prevent_deleting_column_in_table_item
DROP TRIGGER IF EXISTS prevent_deleting_column_in_table_paymentorder
DROP TRIGGER IF EXISTS prevent_deleting_column_in_table_orderentry
DROP TRIGGER IF EXISTS prevent_deleting_column_in_table_payment
DROP TRIGGER IF EXISTS prevent_deleting_column_in_table_settings
DROP TRIGGER IF EXISTS prevent_deleting_column_in_table_pdf
DROP TRIGGER IF EXISTS prevent_deleting_column_in_table_pdfdownload

DROP TRIGGER IF EXISTS prevent_dropping_table_account ON Account
DROP TRIGGER IF EXISTS prevent_dropping_table_accounttype ON AccountType
DROP TRIGGER IF EXISTS prevent_dropping_table_vendor ON Vendor
DROP TRIGGER IF EXISTS prevent_dropping_table_item ON Item
DROP TRIGGER IF EXISTS prevent_dropping_table_paymentorder ON PaymentOrder
DROP TRIGGER IF EXISTS prevent_dropping_table_orderentry ON OrderEntry
DROP TRIGGER IF EXISTS prevent_dropping_table_payment ON Payment
DROP TRIGGER IF EXISTS prevent_dropping_table_settings ON Settings
DROP TRIGGER IF EXISTS prevent_dropping_table_pdf ON PDF
DROP TRIGGER IF EXISTS prevent_dropping_table_pdfdownload ON PDFDownload

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
