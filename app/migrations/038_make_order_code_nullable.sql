-- Make order_code column nullable to match Ent schema definition

ALTER TABLE "paymentorder" ALTER COLUMN "order_code" DROP NOT NULL;
