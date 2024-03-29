-- Add Field Payout to Payment

ALTER TABLE Payment ADD COLUMN Payout integer REFERENCES Payment;
ALTER TABLE Payment ADD COLUMN Item integer REFERENCES Item;
ALTER TABLE Payment ADD COLUMN Price integer NOT NULL DEFAULT 0;
ALTER TABLE Payment ADD COLUMN Quantity integer NOT NULL DEFAULT 0;
UPDATE Payment SET item = o.item, quantity = o.quantity, price = o.price FROM orderentry o WHERE Payment.orderentry  = o.id;


---- create above / drop below ----

ALTER TABLE Payment DROP COLUMN Payout;
ALTER TABLE Payment DROP COLUMN Item;
ALTER TABLE Payment DROP COLUMN Price;
ALTER TABLE Payment DROP COLUMN Quantity;
