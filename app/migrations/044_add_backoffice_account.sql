-- Add Backoffice to the AccountType enum
ALTER TYPE AccountType ADD VALUE IF NOT EXISTS 'Backoffice';

-- Insert Backoffice vendor + account if they don't exist yet
INSERT INTO vendor (keycloakid, urlid, licenseid, firstname, lastname, email, lastpayout, isdisabled, language, telephone, registrationdate, vendorsince, onlinemap, hassmartphone, hasbankaccount, accountproofurl, debt, isdeleted)
SELECT '', '', 'Backoffice', '', '', 'Backoffice@augustina.cc', NOW(), false, '', '', '', '', false, false, false, '', '', false
WHERE NOT EXISTS (SELECT 1 FROM vendor WHERE licenseid = 'Backoffice');

INSERT INTO account (name, balance, type, vendor)
SELECT 'Backoffice', 0, 'Backoffice', v.id
FROM vendor v
WHERE v.licenseid = 'Backoffice'
  AND NOT EXISTS (SELECT 1 FROM account WHERE type = 'Backoffice');
