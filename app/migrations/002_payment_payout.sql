-- Add Field Payout to Payment

ALTER TABLE Payment ADD COLUMN Payout integer REFERENCES Payment;

---- create above / drop below ----

ALTER TABLE Payment DROP COLUMN Payout;
