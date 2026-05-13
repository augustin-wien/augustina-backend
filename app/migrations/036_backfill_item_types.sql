-- Backfill item_type field for existing items

BEGIN;

-- Items marked as license items
UPDATE item SET item_type = 'license_item' WHERE islicenseitem = true;

-- Items that are PDF items (typically documents, treated as issues/documents)
UPDATE item SET item_type = 'issue' WHERE ispdfitem = true AND item_type = 'normal_item';
UPDATE item SET item_type = 'donation' WHERE name = 'donation' AND item_type = 'normal_item';
UPDATE item SET item_type = 'transaction_costs' WHERE name = 'transactionCosts' AND item_type = 'normal_item';

-- Note: Items with specific names matching donation/transaction costs should be set appropriately
-- These can be identified by name matching in the application config
-- For now, setting item_type based on islicenseitem and ispdfitem flags
-- Further classification can be done via application logic or direct SQL if config values are known

-- All remaining items keep default 'normal_item' item_type
-- Items can be manually adjusted via the API or direct database updates as needed

COMMIT;
