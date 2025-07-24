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
			Default(false), // New field for using tip instead of donation
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
