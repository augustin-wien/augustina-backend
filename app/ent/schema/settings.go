package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Settings holds the schema definition for the Settings entity.
type Settings struct {
	ent.Schema
}

// Fields of the Settings.
func (Settings) Fields() []ent.Field {
	fields := []ent.Field{
		field.Int("id").
			Positive(),
		field.String("AGBUrl").
			StorageKey("agburl").
			Default("https://augustina.cc/agb"),
		field.String("Color").
			Default("#000000"),
		field.String("FontColor").
			StorageKey("fontcolor").
			Default("#FFFFFF"),
		field.String("Logo").
			Default("https://augustina.cc/logo.png"),
		field.Int("MaxOrderAmount").
			StorageKey("maxorderamount").
			Default(10000), // Default max order amount in cents
		field.Bool("OrgaCoversTransactionCosts").
			StorageKey("orgacoverstransactioncosts").
			Default(true),
		field.Bool("WebshopIsClosed").
			StorageKey("webshopisclosed").
			Default(false),
		field.String("VendorNotFoundHelpUrl").
			StorageKey("vendornotfoundhelpurl").
			Default("https://augustina.cc/help/vendor-not-found"),
		field.String("MaintainanceModeHelpUrl").
			StorageKey("maintainancemodehelpurl").
			Default("https://augustina.cc/help/maintenance-mode"),
		field.String("VendorEmailPostfix").
			StorageKey("vendoremailpostfix").
			Default("@augustina.cc"),
		field.String("NewspaperName").
			StorageKey("newspapername").
			Default("Augustina"),
		field.String("QRCodeUrl").
			StorageKey("qrcodeurl").
			Default("https://augustina.cc/qrcode"),
		field.String("QRCodeLogoImgUrl").
			StorageKey("qrcodelogoimgurl").
			Default("https://augustina.cc/qrcode/logo.png"),
		field.Float("MapCenterLat").
			StorageKey("mapcenterlat").
			Default(48.2082),
		field.Float("MapCenterLong").
			StorageKey("mapcenterlong").
			Default(16.3738),
		field.Bool("UseVendorLicenseIdInShop").
			StorageKey("usevendorlicenseidinshop").
			Default(false),
		field.String("Favicon").
			Default("https://augustina.cc/favicon.ico"),
		field.String("QRCodeSettings").
			StorageKey("qrcodesettings").
			Default(`{"dotsOptions":{"color":"#000","type":"dots"},"backgroundOptions":{"color":"#fff"},"imageOptions":{"hideBackgroundDots":false,"imageSize":0.5,"crossOrigin":"anonymous","margin":0},"cornerSquareOptions":{"type":"dot","color":"#000"},"cornersDotOptions":{"type":"dot","color":"#000"},"qrCodeOptions":{"typeNumber":0,"mode":"Byte","errorCorrectionLevel":"H"}}`),
		field.Bool("QRCodeEnableLogo").
			StorageKey("qrcodeenablelogo").
			Default(false),
		field.Bool("UseTipInsteadOfDonation").
			StorageKey("usetipinsteadofdonation").
			Default(false),
		field.Bool("ShopLanding").
			StorageKey("shoplanding").
			Default(false),
		field.String("DigitalItemsUrl").
			StorageKey("digitalitemsurl").
			Default("https://augustina.cc/digital-items"),
		field.String("AbonementUrl").
			StorageKey("abonementurl").
			Default(""),
		field.Bool("AbonementEnabled").
			StorageKey("abonementenabled").
			Default(false),
		field.Bool("POSEnabled").
			StorageKey("posenabled").
			Default(true),
		field.String("WordPressInviteURL").
			StorageKey("wordpressinviteurl").
			Default(""),
		field.String("WordPressInviteAPIKey").
			StorageKey("wordpressinviteapikey").
			Default(""),
		field.Int("WordPressInviteTTL").
			StorageKey("wordpressinvitettl").
			Default(604800),
		// Linked next to the terms in the shop. Empty means no link is shown.
		field.String("PrivacyPolicyUrl").
			StorageKey("privacypolicyurl").
			Default(""),
		// Matomo is opt-in per tenant: tracking only starts once both the url and the site id
		// are set, so an empty default means no tracking at all.
		field.String("MatomoUrl").
			StorageKey("matomourl").
			Default(""),
		field.String("MatomoSiteId").
			StorageKey("matomositeid").
			Default(""),
		// The reading portal a customer is redirected to after resetting their password and
		// the URL embedded in the digital-licence/welcome emails - tenant-specific, so it has
		// to be admin-editable rather than a value baked into a shared .env template.
		field.String("OnlinePaperUrl").
			StorageKey("onlinepaperurl").
			Default(""),
		// Display name in the From header of outgoing mails. Empty falls back to the
		// SMTP_SENDER_NAME env var.
		field.String("MailSenderName").
			StorageKey("mailsendername").
			Default(""),
		// Records VivaWallet payments from other sales channels, such as the donation terminal
		field.Bool("DonationTrackingEnabled").
			StorageKey("donationtrackingenabled").
			Default(false),
		// Whether the shop asks the customer to accept the terms before paying
		field.Bool("AGBRequired").
			StorageKey("agbrequired").
			Default(true),
		// Whether the shop shows the donation/tip step
		field.Bool("DonationPageEnabled").
			StorageKey("donationpageenabled").
			Default(true),
		// Sends the products in the cart to Matomo as events. Only has an effect when Matomo is configured.
		field.Bool("MatomoTrackCart").
			StorageKey("matomotrackcart").
			Default(false),
		// Shows a feedback form at the end of the purchase and on the error pages
		field.Bool("FeedbackEnabled").
			StorageKey("feedbackenabled").
			Default(false),
		// Recipient of the feedback mails. Not public.
		field.String("FeedbackEmail").
			StorageKey("feedbackemail").
			Default(""),
		// Offers a newsletter opt-in when a digital item or abonement is bought
		field.Bool("NewsletterEnabled").
			StorageKey("newsletterenabled").
			Default(false),
	}
	for _, f := range fields {
		f.Descriptor().Tag = `json:"` + f.Descriptor().Name + `"`
	}
	return fields
}

// Edges of the Settings.
func (Settings) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("MainItem", Item.Type).StorageKey(edge.Column("mainitem")).Unique(),
	}
}
