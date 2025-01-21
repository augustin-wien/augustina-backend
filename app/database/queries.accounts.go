package database

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

// Accounts -------------------------------------------------------------------

// CreateSingleAccount creates an account in the database
func (db *Database) CreateSpecialVendorAccount(vendor Vendor) (vendorID int, err error) {

	// Create a new vendor account
	err = db.Dbpool.QueryRow(context.Background(), "INSERT INTO Vendor (Keycloakid, UrlID, LicenseID, FirstName, LastName, Email, LastPayout, IsDisabled, Language, Telephone, RegistrationDate, VendorSince, OnlineMap, HasSmartphone, HasBankAccount) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15) RETURNING ID",
		"", "", vendor.LicenseID, "", "", vendor.Email, time.Now(), false, "", "", "", "", false, false, false).Scan(&vendorID)
	if err != nil {
		log.Errorf("CreateSpecialVendor: create vendor %s %+v", vendor.Email, err)
		return
	}

	_, err = db.Dbpool.Exec(context.Background(), "INSERT INTO Account (Name, Balance, Type, Vendor) values ($1, 0, $2, $3) RETURNING ID", vendor.LicenseID, vendor.LicenseID, vendorID)
	if err != nil {
		log.Error("CreateSpecialVendor: create vendor account %s %+v", vendor.Email, err)
		return
	}

	return vendorID, err
}

// ListAccounts returns all accounts from the database
func (db *Database) ListAccounts() (accounts []Account, err error) {
	rows, err := db.Dbpool.Query(context.Background(), "select * from Account")
	if err != nil {
		log.Error("ListAccounts: ", err)
		return accounts, err
	}
	defer rows.Close()
	for rows.Next() {
		var account Account
		err = rows.Scan(&account.ID, &account.Name, &account.Balance, &account.Type, &account.User, &account.Vendor)
		if err != nil {
			log.Error("ListAccounts: ", err)
			return accounts, err
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

// GetAccountByID returns the account with the given ID
func (db *Database) GetAccountByID(id int) (account Account, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Account WHERE ID = $1", id).Scan(&account.ID, &account.Name, &account.Balance, &account.Type, &account.User, &account.Vendor)
	if err != nil {
		if err.Error() == "no rows in result set" {
			err = errors.New("account does not exist")
		}
		log.Error("GetAccountByID: ", err)
	}
	return
}

// GetOrCreateAccountByUserID returns the account with the given user
func (db *Database) GetOrCreateAccountByUserID(userID string) (account Account, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Account WHERE UserID = $1", userID).Scan(&account.ID, &account.Name, &account.Balance, &account.Type, &account.User, &account.Vendor)
	if err != nil {
		if err.Error() == "no rows in result set" {
			err = db.Dbpool.QueryRow(context.Background(), "INSERT INTO Account (Type, UserID) values ($1, $2) RETURNING *", "UserAuth", userID).Scan(&account.ID, &account.Name, &account.Balance, &account.Type, &account.User, &account.Vendor)
			log.Info("Created new account for user " + userID)
		} else {
			log.Error("GetOrCreateAccountByUserID: ", err)
		}
	}
	return
}

// GetAccountByVendorID returns the account with the given vendor
func (db *Database) GetAccountByVendorID(vendorID int) (account Account, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Account WHERE Vendor = $1", vendorID).Scan(&account.ID, &account.Name, &account.Balance, &account.Type, &account.User, &account.Vendor)
	if err != nil {
		if err.Error() == "no rows in result set" {
			err = errors.New("vendor does not exist or has no account")
		}
		log.Error("GetAccountByVendorID: ", err, vendorID)
	}
	return
}

// Memory cache for account type id's
var accountTypeIDCache map[string]int

// GetAccountTypeID returns the (cached) ID of an account type
func (db *Database) GetAccountTypeID(accountType string) (accountTypeID int, err error) {
	if accountTypeIDCache == nil {
		accountTypeIDCache = make(map[string]int)
	}
	if accountTypeIDCache[accountType] != 0 {
		accountTypeID = accountTypeIDCache[accountType]
		return
	}
	err = db.Dbpool.QueryRow(context.Background(), "SELECT ID FROM Account WHERE Type = $1", accountType).Scan(&accountTypeID)
	if err != nil {
		log.Error("GetAccountTypeID: ", accountType, err)
		return
	}
	accountTypeIDCache[accountType] = accountTypeID
	return
}

// GetAccountByType returns the account with the given type
// Works only for types with a single entry in the database
func (db *Database) GetAccountByType(accountType string) (account Account, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Account WHERE Type = $1", accountType).Scan(&account.ID, &account.Name, &account.Balance, &account.Type, &account.User, &account.Vendor)
	if err != nil {
		log.Error("GetAccountByType: ", err)
	}
	return
}

// updateAccountBalanceTx updates the balance of an account in an transaction
func updateAccountBalanceTx(tx pgx.Tx, id int, balanceDiff int) (err error) {

	var account Account

	// Lock account balance via "for update"
	// https://stackoverflow.com/a/45871295/19932351
	err = tx.QueryRow(context.Background(), "SELECT Balance FROM Account WHERE ID = $1", id).Scan(&account.Balance)
	if err != nil {
		log.Error("updateAccountBalanceTx: query row", err)
	}
	newBalance := account.Balance + balanceDiff

	_, err = tx.Exec(context.Background(), `
	UPDATE Account
	SET Balance = $2
	WHERE ID = $1
	`, id, newBalance)
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
	tx, err := db.Dbpool.Begin(ctx)
	if err != nil {
		return
	}

	// Provide defer func to commit or rollback transaction after function returns
	defer func() { err = DeferTx(tx, err) }()

	// Get account
	vendorAccount, err := db.GetAccountByVendorID(vendorID)
	if err != nil {
		return
	}

	err = db.Dbpool.QueryRow(ctx, "SELECT Balance FROM Account WHERE ID = $1", vendorAccount.ID).Scan(&vendorAccount.Balance)
	if err != nil {
		log.Error("UpdateAccountBalanceByOpenPayments: ", err)
	}
	log.Info("UpdateAccountBalanceByOpenPayments: Balance of account where Vendor = " + strconv.Itoa(vendorID) + " is " + strconv.Itoa(vendorAccount.Balance))

	var openPaymentsReceiverSum int
	err = db.Dbpool.QueryRow(ctx, "SELECT COALESCE(SUM(Amount), 0) FROM Payment WHERE Payout IS NULL AND Paymentorder IS NOT NULL AND Receiver = $1", vendorAccount.ID).Scan(&openPaymentsReceiverSum)
	if err != nil {
		log.Error("UpdateAccountBalanceByOpenPayments: ", err)
	}

	// Get open payments where vendor is sender
	var openPaymentsSenderSum int
	err = db.Dbpool.QueryRow(ctx, "SELECT COALESCE(SUM(Amount), 0) FROM Payment WHERE Payout IS NULL AND Paymentorder IS NOT NULL AND Sender = $1", vendorAccount.ID).Scan(&openPaymentsSenderSum)
	if err != nil {
		log.Error("UpdateAccountBalanceByOpenPayments: ", err)
	}

	// Calculate new balance
	openPaymentsSum := openPaymentsReceiverSum - openPaymentsSenderSum

	_, err = tx.Exec(ctx, "UPDATE Account SET Balance = $1 WHERE ID = $2", openPaymentsSum, vendorAccount.ID)
	if err != nil {
		log.Error("UpdateAccountBalanceByOpenPayments: ", err)
	}

	log.Info("UpdateAccountBalanceByOpenPayments: Updated balance of account " + strconv.Itoa(vendorID) + " from " + strconv.Itoa(vendorAccount.Balance) + " to " + strconv.Itoa(openPaymentsSum) + ", openPaymentsReceiverSum = " + strconv.Itoa(openPaymentsReceiverSum) + ", openPaymentsSenderSum = " + strconv.Itoa(openPaymentsSenderSum) + ")")

	return openPaymentsSum, err
}
