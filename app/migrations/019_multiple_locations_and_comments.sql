-- Write your migrate up statements here

CREATE TABLE Locations (
    id serial PRIMARY KEY,
    name text,
    address text,
    longitude float8 NOT NULL DEFAULT 0.1,
    latitude float8 NOT NULL DEFAULT 0.1,
    zip text,
    working_time text,
    vendor_locations integer REFERENCES Vendor (id) ON DELETE SET NULL
);


-- Step 2: Migrate data from Vendor to Locations
INSERT INTO locations (name, address, longitude, latitude, zip, working_time, vendor_locations)
SELECT 
    vendor.location AS name,
    vendor.address AS address,
    vendor.longitude AS longitude,
    vendor.latitude AS latitude,
    vendor.plz AS zip,
    vendor.workingtime AS working_time,
    vendor.id AS vendor_locations
FROM vendor;

-- Step 3: Drop columns from Vendor

ALTER TABLE vendor DROP COLUMN longitude;
ALTER TABLE vendor DROP COLUMN latitude;
ALTER TABLE vendor DROP COLUMN address;
ALTER TABLE vendor DROP COLUMN pLZ;
ALTER TABLE vendor DROP COLUMN location;
ALTER TABLE vendor DROP COLUMN workingtime;


-- Step 4: Add comments table
CREATE TABLE Comments (
    id serial PRIMARY KEY,
    comment text,
    warning boolean NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at timestamptz,
    vendor_comments integer REFERENCES Vendor (id) ON DELETE SET NULL
);

-- Step 5: Migrate comments from Vendor to Comments
INSERT INTO Comments (comment, warning, created_at, vendor_comments)
SELECT 
    vendor.Comment AS comment,
    FALSE AS warning,
    TO_TIMESTAMP(RegistrationDate, 'YYYY-MM-DD') AS created_at,
    vendor.id AS vendor_comments
FROM Vendor;

-- Step 6: Drop comment column from Vendor

ALTER TABLE vendor DROP COLUMN comment;

---- create above / drop below ----

DROP TABLE Locations;
DROP TABLE Comments
-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
