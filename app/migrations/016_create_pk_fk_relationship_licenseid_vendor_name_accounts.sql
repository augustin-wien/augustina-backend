-- Write your migrate up statements here

-- Step 1: Ensure LicenseID in Vendor table is NOT NULL
ALTER TABLE Vendor ALTER COLUMN LicenseID SET NOT NULL;

-- Step 2: Ensure Name in Account table is NOT NULL
ALTER TABLE Account ALTER COLUMN Name SET NOT NULL;

-- Step 3: Create unique indexes to ensure uniqueness of Name in Accounts table
CREATE UNIQUE INDEX idx_account_name ON Account(Name);

-- Step 4: Drop foreign key constraints that depend on the existing primary key
ALTER TABLE Account DROP CONSTRAINT IF EXISTS account_vendor_fkey;
ALTER TABLE PaymentOrder DROP CONSTRAINT IF EXISTS paymentorder_vendor_fkey;

-- Step 5: Drop the existing primary key constraint on ID in Vendor table
ALTER TABLE Vendor DROP CONSTRAINT IF EXISTS vendor_pkey;

-- Step 6: Add a composite primary key on ID and LicenseID in Vendor table
ALTER TABLE Vendor ADD CONSTRAINT pk_vendor_id_licenseid PRIMARY KEY (ID, LicenseID);

-- Step 7: Create a unique constraint on LicenseID to satisfy foreign key requirement
ALTER TABLE Vendor ADD CONSTRAINT uq_vendor_licenseid UNIQUE (LicenseID);
ALTER TABLE Vendor ADD CONSTRAINT uq_vendor_id UNIQUE (ID);

-- Step 8: Recreate foreign key constraints to reference the new primary key and unique constraint
ALTER TABLE Account ADD CONSTRAINT account_vendor_fkey FOREIGN KEY (Vendor) REFERENCES Vendor(ID) ON DELETE SET NULL;
ALTER TABLE PaymentOrder ADD CONSTRAINT paymentorder_vendor_fkey FOREIGN KEY (Vendor) REFERENCES Vendor(ID) ON DELETE SET NULL;

-- Step 9: Add the foreign key constraint from Name in Account table to LicenseID in Vendor table
ALTER TABLE Account ADD CONSTRAINT fk_account_name_licenseid FOREIGN KEY (Name) REFERENCES Vendor(LicenseID) ON DELETE SET NULL ON UPDATE CASCADE;


---- create above / drop below ----

-- Step 1: Drop the foreign key constraint from Name in Account table to LicenseID in Vendor table
ALTER TABLE Account DROP CONSTRAINT IF EXISTS fk_account_name_licenseid;

-- Step 2: Drop the unique constraint on LicenseID in Vendor table
ALTER TABLE Vendor DROP CONSTRAINT IF EXISTS uq_vendor_licenseid;
ALTER TABLE Vendor DROP CONSTRAINT IF EXISTS uq_vendor_id;

-- Step 3: Drop the composite primary key on ID and LicenseID in Vendor table
ALTER TABLE Vendor DROP CONSTRAINT IF EXISTS pk_vendor_id_licenseid;

-- Step 4: Drop foreign key constraints that reference the composite primary key
ALTER TABLE Account DROP CONSTRAINT IF EXISTS account_vendor_fkey;
ALTER TABLE PaymentOrder DROP CONSTRAINT IF EXISTS paymentorder_vendor_fkey;

-- Step 5: Re-add the original primary key on ID in Vendor table
ALTER TABLE Vendor ADD CONSTRAINT vendor_pkey PRIMARY KEY (ID);

-- Step 6: Recreate foreign key constraints to reference the original primary key
ALTER TABLE Account ADD CONSTRAINT account_vendor_fkey FOREIGN KEY (Vendor) REFERENCES Vendor(ID) ON DELETE SET NULL;
ALTER TABLE PaymentOrder ADD CONSTRAINT paymentorder_vendor_fkey FOREIGN KEY (Vendor) REFERENCES Vendor(ID) ON DELETE SET NULL;

-- Step 7: Drop the unique indexes created during the upgrade
DROP INDEX IF EXISTS idx_account_name;

-- Step 8: Optionally, revert the NOT NULL constraints if they were not originally there
ALTER TABLE Vendor ALTER COLUMN LicenseID DROP NOT NULL;
ALTER TABLE Account ALTER COLUMN Name DROP NOT NULL;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.