ALTER TABLE vendor
    ADD COLUMN IF NOT EXISTS hasownemail BOOLEAN NOT NULL DEFAULT true;

-- Vendors imported via CSV got the generated internal address
-- (license ID + vendor email postfix) instead of a real mailbox
UPDATE vendor
SET hasownemail = false
WHERE EXISTS (
    SELECT 1 FROM settings
    WHERE settings.vendoremailpostfix <> ''
      AND lower(vendor.email) = lower(vendor.licenseid || settings.vendoremailpostfix)
);
