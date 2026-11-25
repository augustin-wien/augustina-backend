package paymentprovider

import (
	"errors"

	"github.com/augustin-wien/augustina-backend/ent"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"

	"time"
)

// CreatePaypalTransactionCosts creates transaction costs for Paypal payments
func CreatePaypalTransactionCosts(paymentSuccessful TransactionSuccessRequest, order database.Order) (err error) {
	// Check if VivaWalletTransactionTypeIDPaypal is set
	if config.Config.VivaWalletTransactionTypeIDPaypal == 0 {
		return errors.New("viva wallet transaction type id for paypal is not set")
	}

	// PayPal transaction costs (TransactionTypeId == VivaWalletTransactionTypeIDPaypal, i.e. 48)
	// are currently not booked. The former calculation, kept for when they are again:
	// Check TransactionTypeId here: https://developer.vivawallet.com/integration-reference/response-codes/#transactiontypeid-parameter

	// // Check if PaypalPercentageCosts and PaypalFixCosts are set
	// if config.Config.PaypalPercentageCosts == 0 {
	// 	return errors.New("Env variable PaypalPercentageCosts is not set")
	// }

	// if config.Config.PaypalFixCosts == 0 {
	// 	return errors.New("Env variable PaypalFixCosts is not set")
	// }

	// // Convert percentage to multiply it with total sum i.e. 0.05 for 5% transaction costs
	// convertedPercentageCosts := (config.Config.PaypalPercentageCosts) / 100

	// // Calculate transaction costs i.e. 0.034 * 100ct + 35 = 38.4ct
	// paypalAmount := convertedPercentageCosts*float64(order.GetTotal()) + config.Config.PaypalFixCosts

	// // Given after research that Paypal rounds down on 3.4 ct to 3 ct we use math.Round
	// paypalAmount = math.Round(paypalAmount)

	// // Create order entries for transaction costs
	// // WARNING: int() always rounds down in case you stop using math.Round
	// err = CreateTransactionCostEntries(order, int(paypalAmount), "Paypal")
	// if err != nil {
	// 	return err
	// }

	return

}

// logPriceNotBooked logs why a price webhook left its transaction costs unbooked. It is the one
// error log per failed delivery, carrying enough to find the order and the cause from the alert
// alone: reason is a short stable slug to search and group by.
func logPriceNotBooked(reason string, orderID int, data PriceEventData, verification TransactionVerificationResponse, err error) {
	log.Errorw("HandlePaymentPriceResponse: transaction costs not booked",
		"reason", reason,
		"order_id", orderID,
		"order_code", data.OrderCode.String(),
		"transaction_id", data.TransactionID,
		"total_commission", data.TotalCommission,
		"viva_order_code", verification.OrderCode.String(),
		"viva_amount", verification.Amount,
		"viva_status_id", verification.StatusID,
		"viva_merchant_trns", verification.MerchantTrns,
		"viva_customer_trns", verification.CustomerTrns,
		"error", err,
	)
}

// HandlePaymentPriceResponse handles the webhook response for a price change for now only for Card transactions
func HandlePaymentPriceResponse(paymentPrice TransactionPriceRequest) (err error) {
	data := paymentPrice.EventData

	// 1. Check: Verify that webhook request belongs to VivaWallet by verifying transactionID
	verification, err := VerifyTransactionID(data.TransactionID, false)
	if err != nil {
		logPriceNotBooked("viva_verification_failed", 0, data, verification, err)
		return err
	}

	// 2. Check: Verify that order can be found by ordercode
	var order database.Order
	// Retry getting order from database to avoid race conditions
	for i := 0; i < 5; i++ {
		order, err = database.Db.GetOrderByOrderCode(data.OrderCode.String())
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	// The price event carries no source code, so the merchant reference from the transaction
	// details tells payments from other sales channels apart
	if ent.IsNotFound(err) && verification.MerchantTrns != merchantTrns {
		log.Infow("HandlePaymentPriceResponse: ignoring transaction from another sales channel",
			"order_code", data.OrderCode.String(),
			"transaction_id", data.TransactionID,
			"viva_merchant_trns", verification.MerchantTrns,
			"total_commission", data.TotalCommission,
		)
		return recordExternalPayment(database.ExternalPaymentUpdate{
			TransactionID: data.TransactionID,
			OrderCode:     data.OrderCode.String(),
			Amount:        toCents(verification.Amount),
			Fee:           toCents(data.TotalCommission),
			StatusID:      verification.StatusID,
			MerchantTrns:  verification.MerchantTrns,
			PaidAt:        parseVivaTime(verification.InsDate),
		})
	}
	if err != nil {
		logPriceNotBooked("order_not_found", 0, data, verification, err)
		return err
	}

	// 3. Check: If TotalCommission is 0.0, return without creating transaction costs
	if data.TotalCommission == 0.0 {
		return
	}

	// VivaWallet retries failed deliveries, so costs booked by an earlier delivery must not be
	// booked again
	booked, err := hasTransactionCostEntries(order)
	if err != nil {
		logPriceNotBooked("check_existing_costs_failed", order.ID, data, verification, err)
		return err
	}
	if booked {
		log.Infow("HandlePaymentPriceResponse: transaction costs already booked, skipping", "order_id", order.ID, "transaction_id", data.TransactionID)
		return nil
	}

	transactionCosts := int(data.TotalCommission * 100) // Convert to cents
	// Create order entries for transaction costs
	err = CreateTransactionCostEntries(order, transactionCosts, "VivaWallet")
	if err != nil {
		logPriceNotBooked("booking_failed", order.ID, data, verification, err)
		return err
	}

	return
}

// hasTransactionCostEntries reports whether transaction costs were already booked on the order
func hasTransactionCostEntries(order database.Order) (bool, error) {
	transactionCostsItem, err := database.Db.GetItemByName(config.Config.TransactionCostsName)
	if err != nil {
		return false, err
	}
	for _, entry := range order.Entries {
		if entry.Item == transactionCostsItem.ID {
			return true, nil
		}
	}
	return false, nil
}

// CreateTransactionCostEntries creates payments and order entries to list transaction costs
func CreateTransactionCostEntries(order database.Order, transactionCosts int, paymentProvider string) (err error) {

	if config.Config.TransactionCostsName == "" {
		return errors.New("transaction costs name is not set")
	}

	// Get ID of transaction costs item
	transactionCostsItem, err := database.Db.GetItemByName(config.Config.TransactionCostsName)
	if err != nil {
		log.Error("Getting transaction costs item failed: ", err)
		return err
	}

	// Get ID of VivaWallet account
	paymentProviderAccountID, err := database.Db.GetAccountTypeID(paymentProvider)
	if err != nil {
		log.Error("Getting account type ID failed: ", err)
		return err
	}

	// Get ID of vendor account
	vendorAccount, err := database.Db.GetAccountByVendorID(order.Vendor)
	if err != nil {
		log.Error("Getting ID of vendor account failed: ", err)
		return err
	}

	// Create order entries for transaction costs
	var entries = []database.OrderEntry{
		{
			Item:     transactionCostsItem.ID,  // ID of transaction costs item
			Quantity: transactionCosts,         // Amount of transaction costs
			Sender:   vendorAccount.ID,         // ID of vendor
			Receiver: paymentProviderAccountID, // ID of Payment Provider
		},
	}

	// Create payment with order entries
	err = database.Db.CreatePayedOrderEntries(order.ID, entries)
	if err != nil {
		log.Error("Creating payment with order entries failed: ", err)
		return err
	}

	var settings *ent.Settings
	settings, err = database.Db.GetSettings()
	if err != nil {
		log.Error("Getting settings failed: ", err)
		return err
	}

	if settings.OrgaCoversTransactionCosts {

		// Get ID of Orga account
		orgaAccountID, err := database.Db.GetAccountTypeID("Orga")
		if err != nil {
			log.Error("Getting Orga account ID failed: ", err)
			return err
		}
		// Create payment for covering transaction costs by Organization
		var entries = []database.OrderEntry{
			{
				Item:     transactionCostsItem.ID, // ID of transaction costs item
				Quantity: transactionCosts,        // Amount of transaction costs
				Sender:   orgaAccountID,           // ID of Orga
				Receiver: vendorAccount.ID,        // ID of vendor
			},
		}
		// Append transaction cost entries here
		err = database.Db.CreatePayedOrderEntries(order.ID, entries)
		if err != nil {
			log.Error("Appending transaction costs failed: ", err)
			return err
		}
	}
	return
}
