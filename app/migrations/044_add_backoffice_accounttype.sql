-- Add Backoffice to the AccountType enum.
-- Must be in its own migration (separate transaction) because PostgreSQL
-- does not allow using a newly-added enum value in the same transaction.
ALTER TYPE AccountType ADD VALUE IF NOT EXISTS 'Backoffice';
