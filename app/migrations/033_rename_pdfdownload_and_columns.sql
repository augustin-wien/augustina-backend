-- Migration to fix PDFDownload table and column names to match Ent schema

DO $$
BEGIN
    -- Rename table
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='pdfdownload') THEN
        ALTER TABLE "pdfdownload" RENAME TO "pdf_download";
    END IF;

    -- Rename columns if table exists (either renamed or already existed)
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='pdf_download') THEN
        
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='pdf_download' AND column_name='linkid') THEN
            ALTER TABLE "pdf_download" RENAME COLUMN "linkid" TO "link_id";
        END IF;

        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='pdf_download' AND column_name='pdf') THEN
            ALTER TABLE "pdf_download" RENAME COLUMN "pdf" TO "pdf_id";
        END IF;

        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='pdf_download' AND column_name='emailsent') THEN
             ALTER TABLE "pdf_download" RENAME COLUMN "emailsent" TO "email_sent";
        END IF;

        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='pdf_download' AND column_name='lastdownload') THEN
             ALTER TABLE "pdf_download" RENAME COLUMN "lastdownload" TO "last_download";
        END IF;

        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='pdf_download' AND column_name='downloadcount') THEN
             ALTER TABLE "pdf_download" RENAME COLUMN "downloadcount" TO "download_count";
        END IF;

        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='pdf_download' AND column_name='orderid') THEN
             ALTER TABLE "pdf_download" RENAME COLUMN "orderid" TO "order_id";
        END IF;

        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='pdf_download' AND column_name='itemid') THEN
             ALTER TABLE "pdf_download" RENAME COLUMN "itemid" TO "item_id";
        END IF;
    END IF;

END $$;
