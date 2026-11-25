ALTER TABLE item
    ADD COLUMN IF NOT EXISTS abo_duration_months INTEGER NOT NULL DEFAULT 12;

ALTER TABLE settings
    ADD COLUMN IF NOT EXISTS voucherenabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE paymentorder
    ADD COLUMN IF NOT EXISTS gift_recipient_name VARCHAR NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS gift_message VARCHAR NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS abonement_voucher (
    id SERIAL PRIMARY KEY,
    code VARCHAR NOT NULL UNIQUE,
    item_id INTEGER NOT NULL,
    order_id INTEGER,
    buyer_email VARCHAR NOT NULL DEFAULT '',
    recipient_name VARCHAR NOT NULL DEFAULT '',
    message VARCHAR NOT NULL DEFAULT '',
    duration_months INTEGER NOT NULL,
    status VARCHAR NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    redeemed_at TIMESTAMPTZ,
    redeemed_email VARCHAR,
    redeemed_customer_id INTEGER,
    abonement_id INTEGER
);

CREATE INDEX IF NOT EXISTS abonementvoucher_order_id ON abonement_voucher (order_id);

INSERT INTO mail_templates (name, subject, body, created_at, updated_at)
VALUES (
    'abonementVoucher',
    'Dein Gutschein: {{ "{{" }}.ItemName{{ "}}" }}',
    '<p>Hallo!</p>
<p>Vielen Dank f&uuml;r deinen Einkauf. Im Anhang findest du {{ "{{" }}if gt .Count 1{{ "}}" }}deine Gutscheine{{ "{{" }}else{{ "}}" }}deinen Gutschein{{ "{{" }}end{{ "}}" }} f&uuml;r <strong>{{ "{{" }}.ItemName{{ "}}" }}</strong> ({{ "{{" }}.DurationMonths{{ "}}" }} Monate) als PDF zum Ausdrucken oder Weiterleiten.</p>
<p>{{ "{{" }}if gt .Count 1{{ "}}" }}Gutscheincodes{{ "{{" }}else{{ "}}" }}Gutscheincode{{ "{{" }}end{{ "}}" }}:</p>
<ul>
{{ "{{" }}range .Codes{{ "}}" }}  <li><strong>{{ "{{" }}.{{ "}}" }}</strong></li>
{{ "{{" }}end{{ "}}" }}</ul>
<p>Eingel&ouml;st wird der Gutschein unter <a href="{{ "{{" }}.RedeemURL{{ "}}" }}">{{ "{{" }}.RedeemURL{{ "}}" }}</a> oder &uuml;ber den QR-Code auf dem PDF. Das Abo beginnt mit dem Tag der Einl&ouml;sung. Der Gutschein ist unbegrenzt g&uuml;ltig.</p>
<p>Viele Gr&uuml;&szlig;e<br/>Dein Augustin-Team</p>',
    now(),
    now()
)
ON CONFLICT (name) DO NOTHING;
