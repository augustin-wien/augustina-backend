-- Migration to rename columns, handling potential conflicts from 030

DO $$
BEGIN
    -- Handle order_code
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='paymentorder' AND column_name='ordercode') THEN
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='paymentorder' AND column_name='order_code') THEN
             -- Both exist, migrate data and drop old
             UPDATE "paymentorder" SET "order_code" = "ordercode" WHERE "order_code" IS NULL;
             ALTER TABLE "paymentorder" DROP COLUMN "ordercode";
        ELSE
             -- Only old exists, rename
             ALTER TABLE "paymentorder" RENAME COLUMN "ordercode" TO "order_code";
        END IF;
    END IF;

    -- Handle transaction_id
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='paymentorder' AND column_name='transactionid') THEN
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='paymentorder' AND column_name='transaction_id') THEN
             UPDATE "paymentorder" SET "transaction_id" = "transactionid" WHERE "transaction_id" = '';
             ALTER TABLE "paymentorder" DROP COLUMN "transactionid";
        ELSE
             ALTER TABLE "paymentorder" RENAME COLUMN "transactionid" TO "transaction_id";
        END IF;
    END IF;

    -- Handle transaction_type_id
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='paymentorder' AND column_name='transactiontypeid') THEN
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='paymentorder' AND column_name='transaction_type_id') THEN
             UPDATE "paymentorder" SET "transaction_type_id" = "transactiontypeid" WHERE "transaction_type_id" = 0;
             ALTER TABLE "paymentorder" DROP COLUMN "transactiontypeid";
        ELSE
             ALTER TABLE "paymentorder" RENAME COLUMN "transactiontypeid" TO "transaction_type_id";
        END IF;
    END IF;

    -- Handle vendor_id
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='paymentorder' AND column_name='vendor') THEN
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='paymentorder' AND column_name='vendor_id') THEN
             UPDATE "paymentorder" SET "vendor_id" = "vendor" WHERE "vendor_id" = 0;
             ALTER TABLE "paymentorder" DROP COLUMN "vendor";
        ELSE
             ALTER TABLE "paymentorder" RENAME COLUMN "vendor" TO "vendor_id";
        END IF;
    END IF;

    -- Handle is_sale (payment)
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='payment' AND column_name='issale') THEN
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='payment' AND column_name='is_sale') THEN
             UPDATE "payment" SET "is_sale" = "issale";
             ALTER TABLE "payment" DROP COLUMN "issale";
        ELSE
             ALTER TABLE "payment" RENAME COLUMN "issale" TO "is_sale";
        END IF;
    END IF;

    -- Handle is_sale (orderentry)
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='orderentry' AND column_name='issale') THEN
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='orderentry' AND column_name='is_sale') THEN
             UPDATE "orderentry" SET "is_sale" = "issale";
             ALTER TABLE "orderentry" DROP COLUMN "issale";
        ELSE
             ALTER TABLE "orderentry" RENAME COLUMN "issale" TO "is_sale";
        END IF;
    END IF;
    
    -- Handle dbsettings
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='dbsettings') THEN
        ALTER TABLE "dbsettings" RENAME TO "db_settings";
    END IF;

    -- Handle db_settings.is_initialized
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='db_settings' AND column_name='isinitialized') THEN
         IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='db_settings' AND column_name='is_initialized') THEN
             ALTER TABLE "db_settings" DROP COLUMN "isinitialized";
         ELSE
             ALTER TABLE "db_settings" RENAME COLUMN "isinitialized" TO "is_initialized";
         END IF;
    END IF;

    -- Handle payment.order_entry
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='payment' AND column_name='orderentry') THEN
         IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='payment' AND column_name='order_entry') THEN
             UPDATE "payment" SET "order_entry" = "orderentry";
             ALTER TABLE "payment" DROP COLUMN "orderentry";
         ELSE
             ALTER TABLE "payment" RENAME COLUMN "orderentry" TO "order_entry";
         END IF;
    END IF;

END $$;

ALTER TABLE "vendor" ADD COLUMN IF NOT EXISTS "isdeleted" BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE "vendor" ADD COLUMN IF NOT EXISTS "accountproofurl" VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE "vendor" ADD COLUMN IF NOT EXISTS "debt" VARCHAR(255) NOT NULL DEFAULT '';

ALTER TABLE "payment" ADD COLUMN IF NOT EXISTS "payout" INTEGER;
ALTER TABLE "payment" ADD COLUMN IF NOT EXISTS "item" INTEGER;
