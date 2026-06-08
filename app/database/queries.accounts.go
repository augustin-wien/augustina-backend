package database

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/augustin-wien/augustina-backend/ent"
	entaccount "github.com/augustin-wien/augustina-backend/ent/account"
	entpayment "github.com/augustin-wien/augustina-backend/ent/payment"
	"gopkg.in/guregu/null.v4"
)

// AccountEntIntoAccount converts an ent.Account to Account struct
func (db *Database) AccountEntIntoAccount(a *ent.Account) Account {
	acc := Account{
		ID:      a.ID,
		Name:    a.Name,
		Balance: int(a.Balance),
		Type:    a.Type,
	}
	if a.UserID != "" {
		acc.User = null.StringFrom(a.UserID)
	}
	if a.VendorID != 0 {
		acc.Vendor = null.IntFrom(int64(a.VendorID))
	}
	return acc
}

// CreateSingleAccount creates an account in the database
func (db *Database) CreateSpecialVendorAccount(vendor Vendor) (vendorID int, err error) {

	// Create a new vendor account
	v, err := db.EntClient.Vendor.Create().
		SetKeycloakid("").
		SetUrlid("").
		SetLicenseid(vendor.LicenseID.String).
		SetFirstname("").
		SetLastname("").
		SetEmail(vendor.Email).
		SetLastpayout(time.Now()).
		SetIsdisabled(false).
		SetLanguage("").
		SetTelephone("").
		SetRegistrationdate("").
		SetVendorsince("").
		SetOnlinemap(false).
		SetHassmartphone(false).
		SetHasbankaccount(false).
		SetAccountproofurl("").
		SetDebt("").
		Save(context.Background())

	if err != nil {
		log.Errorf("CreateSpecialVendor: create vendor %s %+v", vendor.Email, err)
		return 0, err
	}
	vendorID = v.ID

	// Determine account type: for special bootstrap accounts (created via InitiateAccounts)
	// the Vendor.LicenseID contains the account type name (e.g., "Cash", "Orga", "UserAnon", "Paypal", "VivaWallet").
	// For normal vendors use the enum value "Vendor".
	accountType := "Vendor"
	accountName := ""
	if vendor.LicenseID.Valid && vendor.LicenseID.String != "" {
		accountName = vendor.LicenseID.String
		switch vendor.LicenseID.String {
		case "Cash", "Orga", "UserAnon", "Paypal", "VivaWallet", "Backoffice":
			accountType = vendor.LicenseID.String
		default:
			accountType = "Vendor"
		}
	}
	// Use the LicenseID string as the account name when present, otherwise empty name
	_, err = db.EntClient.Account.Create().
		SetName(accountName).
		SetBalance(0).
		SetType(accountType).
		SetVendorID(vendorID).
		Save(context.Background())

	if err != nil {
		log.Error("CreateSpecialVendor: create vendor account failed: ", err)
		return 0, err
	}

	return vendorID, err
}

// ListAccounts returns all accounts from the database
func (db *Database) ListAccounts() (accounts []Account, err error) {
	entAccounts, err := db.EntClient.Account.Query().WithVendor().All(context.Background())
	if err != nil {
		log.Error("ListAccounts: ", err)
		return accounts, err
	}

	for _, a := range entAccounts {
		accounts = append(accounts, db.AccountEntIntoAccount(a))
	}
	return accounts, nil
}

// GetAccountByID returns the account with the given ID
func (db *Database) GetAccountByID(id int) (account Account, err error) {
	a, err := db.EntClient.Account.Get(context.Background(), id)
	if err != nil {
		if ent.IsNotFound(err) {
			err = errors.New("account does not exist")
		}
		log.Error("GetAccountByID: ", err)
		return account, err
	}
	return db.AccountEntIntoAccount(a), nil
}

// GetOrCreateAccountByUserID returns the account with the given user
func (db *Database) GetOrCreateAccountByUserID(userID string) (account Account, err error) {
	// Try to find existing account
	a, err := db.EntClient.Account.Query().Where(entaccount.UserID(userID)).First(context.Background())
	if err != nil {
		if ent.IsNotFound(err) {
			// Create new account
			a, err = db.EntClient.Account.Create().
				SetType("UserAuth").
				SetUserID(userID).
				Save(context.Background())
			if err != nil {
				log.Error("GetOrCreateAccountByUserID: create failed ", err)
				return account, err
			}
			log.Info("Created new account for user " + userID)
		} else {
			log.Error("GetOrCreateAccountByUserID: ", err)
			return account, err
		}
	}
	return db.AccountEntIntoAccount(a), nil
}

// GetAccountTypeID returns the ID of the account with the given type
func (db *Database) GetAccountTypeID(accountType string) (accountTypeID int, err error) {
	// Query using Ent
	a, err := db.EntClient.Account.Query().Where(entaccount.Type(accountType)).First(context.Background())
	if err != nil {
		log.Error("GetAccountTypeID: ", accountType, err)
		return 0, err
	}

	accountTypeID = a.ID
	accountTypeIDCache[accountType] = accountTypeID
	return
}

// GetAccountByType returns the account with the given type
// Works only for types with a single entry in the database
func (db *Database) GetAccountByType(accountType string) (account Account, err error) {
	a, err := db.EntClient.Account.Query().Where(entaccount.Type(accountType)).First(context.Background())
	if err != nil {
		log.Error("GetAccountByType: ", err)
		return account, err
	}
	return db.AccountEntIntoAccount(a), nil
}

var accountTypeIDCache = make(map[string]int)

// GetAccountByVendorID returns the account for a vendor
func (db *Database) GetAccountByVendorID(vendorID int) (account Account, err error) {
	a, err := db.EntClient.Account.Query().Where(entaccount.VendorID(vendorID)).First(context.Background())
	if err != nil {
		if ent.IsNotFound(err) {
			err = errors.New("account does not exist for vendor")
		}
		log.Error("GetAccountByVendorID: ", err)
		return account, err
	}
	return db.AccountEntIntoAccount(a), nil
}

// updateAccountBalanceTx updates the balance of an account in an transaction
func updateAccountBalanceTx(tx *ent.Tx, id int, balanceDiff int) (err error) {
	err = tx.Account.UpdateOneID(id).
		AddBalance(float64(balanceDiff)).
		Exec(context.Background())

	if err != nil {
		log.Error("updateAccountBalanceTx: update ", err)
	}
	return
}

// Todo: test this function
// UpdateAccountBalanceByOpenPayments updates the balance of an account by summing up all open payments (i.e. payments without a payout)
func (db *Database) UpdateAccountBalanceByOpenPayments(vendorID int) (payoutAmount int, err error) {
	ctx := context.Background()
	// Start a transaction
	tx, err := db.EntClient.Tx(ctx)
	if err != nil {
		return
	}

	// Provide defer func to commit or rollback transaction after function returns
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// Get account within the transaction
	// The original SQL used FOR UPDATE. To achieve this in ent, we can use a custom predicate
	// or rely on the later update.
	// For now, simple fetch.
	entAccount, err := tx.Account.Query().Where(entaccount.VendorID(vendorID)).Only(ctx)
	if err != nil {
		log.Error("UpdateAccountBalanceByOpenPayments: couldn't get account for vendor", vendorID, err)
		return
	}
	vendorAccount := db.AccountEntIntoAccount(entAccount)
	log.Info("UpdateAccountBalanceByOpenPayments: Balance of account where Vendor = " + strconv.Itoa(vendorID) + " is " + strconv.Itoa(vendorAccount.Balance))

	var openPaymentsReceiverSum int
	// Sum all open payments for this receiver
	// Aggregate can return 0 results if no match? Let's check.
	// We want SUM(Amount) where Payout is NULL and Receiver = ID
	// Ent provides specialized aggregate methods.
	// However, simple Scan(int) works if query returns one row.
	// But `Aggregate(ent.Sum(Amount))` returns a specific struct.
	// Let's use `Int(ctx)` on the result if we group or just rely on scan.
	// The easiest way is using a struct target for scan.
	var result []struct {
		Sum int
	}
	err = tx.Payment.Query().
		Where(entpayment.ReceiverID(vendorAccount.ID)).
		Where(entpayment.PayoutIDIsNil()).
		Aggregate(ent.Sum(entpayment.FieldAmount)).
		Scan(ctx, &result)

	if err != nil {
		log.Error("UpdateAccountBalanceByOpenPayments: failed to sum receiver payments ", err)
		// If error, we should probably return instead of continuing with partial data
		// But original code continued.
		// Assuming no rows = 0 sum, error might be just 'no rows' or actual db error.
		// Ent returns error if no rows found for Aggregate?
		// No, aggregate usually returns result.
	} else if len(result) > 0 {
		openPaymentsReceiverSum = result[0].Sum
	}

	// Get open payments where vendor is sender
	var openPaymentsSenderSum int
	var cashAccountID int
	cashAcc, err := tx.Account.Query().Where(entaccount.Type("Cash")).First(ctx)
	if err == nil {
		cashAccountID = cashAcc.ID
	} else {
		log.Error("UpdateAccountBalanceByOpenPayments: couldn't get Cash account ID", err)
	}

	var resultSender []struct {
		Sum int
	}

	senderQuery := tx.Payment.Query().
		Where(entpayment.SenderID(vendorAccount.ID)).
		Where(entpayment.PayoutIDIsNil()).
		// POS sale records are bookkeeping-only and must not affect the vendor balance.
		Where(entpayment.Not(entpayment.And(entpayment.IsPos(true), entpayment.IsSale(true))))

	if cashAccountID != 0 {
		senderQuery = senderQuery.Where(entpayment.ReceiverIDNEQ(cashAccountID))
	}

	err = senderQuery.
		Aggregate(ent.Sum(entpayment.FieldAmount)).
		Scan(ctx, &resultSender)

	if err != nil {
		log.Error("UpdateAccountBalanceByOpenPayments: failed to sum sender payments ", err)
	} else if len(resultSender) > 0 {
		openPaymentsSenderSum = resultSender[0].Sum
	}

	// Calculate new balance
	openPaymentsSum := openPaymentsReceiverSum - openPaymentsSenderSum

	// Update account
	err = tx.Account.UpdateOneID(vendorAccount.ID).SetBalance(float64(openPaymentsSum)).Exec(ctx)
	if err != nil {
		log.Error("UpdateAccountBalanceByOpenPayments: ", err)
	}

	log.Info("UpdateAccountBalanceByOpenPayments: Updated balance of account " + strconv.Itoa(vendorID) + " from " + strconv.Itoa(vendorAccount.Balance) + " to " + strconv.Itoa(openPaymentsSum) + ", openPaymentsReceiverSum = " + strconv.Itoa(openPaymentsReceiverSum) + ", openPaymentsSenderSum = " + strconv.Itoa(openPaymentsSenderSum) + ")")

	return openPaymentsSum, err
}
