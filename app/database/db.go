package database

import (
	"context"
	"database/sql"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/utils"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

var log = utils.GetLogger()

// Database struct
type Database struct {
	IsProduction bool
	EntClient    *ent.Client
	DB           *sql.DB
}

// Db is the global database connection pool that is used by all handlers
var Db Database

// InitDb connects to production database and stores it in the global Db variable
func (db *Database) InitDb() (err error) {
	log.Info("Initializing production database")
	err = db.initDb(true, true)
	if err != nil {
		return err
	}
	err = initData(db)
	return err
}

// InitEmptyTestDb connects to an empty testing database and store it in the global Db variable
func (db *Database) InitEmptyTestDb() (err error) {
	log.Info("Initializing empty test database")
	err = db.initDb(false, false)
	if err != nil {
		return err
	}

	// Ensure schema is created/migrated for tests
	// This uses Ent's auto-migration. We log error but proceed because legacy tables (e.g. blocked_ips) might conflict.
	if err := db.EntClient.Schema.Create(context.Background()); err != nil {
		log.Error("InitEmptyTestDb schema creation warning: ", err)
	}

	// Hotfix for test DB schema drift
	ctx := context.Background()
	queries := []string{
		// Renames
		`ALTER TABLE "paymentorder" RENAME COLUMN "ordercode" TO "order_code";`,
		`ALTER TABLE "paymentorder" RENAME COLUMN "transactionid" TO "transaction_id";`,
		`ALTER TABLE "paymentorder" RENAME COLUMN "transactiontypeid" TO "transaction_type_id";`,
		`ALTER TABLE "paymentorder" RENAME COLUMN "vendor" TO "vendor_id";`,
		`ALTER TABLE "payment" RENAME COLUMN "issale" TO "is_sale";`,
		`ALTER TABLE "payment" RENAME COLUMN "orderentry" TO "order_entry";`,
		`ALTER TABLE "orderentry" RENAME COLUMN "issale" TO "is_sale";`,
		`ALTER TABLE "dbsettings" RENAME TO "db_settings";`,
		`ALTER TABLE "db_settings" RENAME COLUMN "isinitialized" TO "is_initialized";`,

		// Drop constraints on legacy fields if they persist (i.e. if rename failed because new column exists)
		`ALTER TABLE "paymentorder" ALTER COLUMN "ordercode" DROP NOT NULL;`,

		// Add missing columns (idempotent)
		`ALTER TABLE "payment" ADD COLUMN IF NOT EXISTS "authorized_by" VARCHAR(255) NOT NULL DEFAULT '';`,
		`ALTER TABLE "payment" ADD COLUMN IF NOT EXISTS "is_sale" BOOLEAN NOT NULL DEFAULT FALSE;`,
		`ALTER TABLE "payment" ADD COLUMN IF NOT EXISTS "payout" INTEGER;`,
		`ALTER TABLE "payment" ADD COLUMN IF NOT EXISTS "item" INTEGER;`,
		`ALTER TABLE "payment" ADD COLUMN IF NOT EXISTS "order_entry" INTEGER;`,
		`ALTER TABLE "paymentorder" ADD COLUMN IF NOT EXISTS "order_code" VARCHAR(255);`,
		`ALTER TABLE "paymentorder" ADD COLUMN IF NOT EXISTS "transaction_id" VARCHAR(255) NOT NULL DEFAULT '';`,
		`ALTER TABLE "paymentorder" ADD COLUMN IF NOT EXISTS "verified_at" TIMESTAMP;`,
		`ALTER TABLE "paymentorder" ADD COLUMN IF NOT EXISTS "transaction_type_id" INTEGER NOT NULL DEFAULT 0;`,
		`ALTER TABLE "paymentorder" ADD COLUMN IF NOT EXISTS "timestamp" TIMESTAMP NOT NULL DEFAULT NOW();`,
		`ALTER TABLE "paymentorder" ADD COLUMN IF NOT EXISTS "user_id" VARCHAR(255);`,
		`ALTER TABLE "paymentorder" ADD COLUMN IF NOT EXISTS "vendor_id" INTEGER NOT NULL DEFAULT 0;`,
		`ALTER TABLE "paymentorder" ADD COLUMN IF NOT EXISTS "customer_email" VARCHAR(255);`,
		`ALTER TABLE "orderentry" ADD COLUMN IF NOT EXISTS "is_sale" BOOLEAN NOT NULL DEFAULT FALSE;`,
		`ALTER TABLE "vendor" ADD COLUMN IF NOT EXISTS "isdeleted" BOOLEAN NOT NULL DEFAULT FALSE;`,
		`ALTER TABLE "vendor" ADD COLUMN IF NOT EXISTS "accountproofurl" VARCHAR(255) NOT NULL DEFAULT '';`,
		`ALTER TABLE "vendor" ADD COLUMN IF NOT EXISTS "debt" VARCHAR(255) NOT NULL DEFAULT '';`,
	}

	for _, q := range queries {
		// Log errors for debugging if needed, but for now ignore as some will fail by design (e.g. rename if column doesn't exist)
		_, errExec := db.DB.ExecContext(ctx, q)
		if errExec != nil {
			log.Info("InitEmptyTestDb query failed but proceeding: ", q, " Error: ", errExec)
		} else {
			log.Info("InitEmptyTestDb query succeeded: ", q)
		}
	}

	err = db.EmptyDatabase()
	if err != nil {
		return err
	}
	err = initData(db)
	return
}

// initData initializes the database with default data
func initData(db *Database) (err error) {
	// Initializes DBSettings if not already initialized
	err = db.InitiateDBSettings()
	if err != nil {
		log.Error("Settings creation failed ", zap.Error(err))
	}

	// Get DBSettings
	var dbSettings DBSettings
	dbSettings, err = db.GetDBSettings()
	if err != nil {
		log.Error("Settings retrieval failed ", zap.Error(err))
	}

	if !dbSettings.IsInitialized {

		// Create default settings
		err = db.InitiateSettings()
		if err != nil {
			log.Error("Settings creation failed ", zap.Error(err))
		}
		// Create default accounts
		err = db.InitiateAccounts()
		if err != nil {
			log.Error("Default accounts creation failed ", zap.Error(err))
		}

		// Create default items
		err = db.InitiateItems()
		if err != nil {
			log.Error("Default items creation failed ", zap.Error(err))
		}

		if db.IsProduction {
			err = db.UpdateInitialSettings()
			if err != nil {
				log.Error("Updating initial Settings failed ", zap.Error(err))
			}

			if config.Config.CreateDemoData {
				err = db.CreateDevData()
				if err != nil {
					log.Error("Dev data creation failed ", zap.Error(err))
				}
			}
		}

		// Update DBSettings to initialized
		err = db.UpdateDBSettings(DBSettings{IsInitialized: true})
		if err != nil {
			log.Error("Updating DBSettings failed ", zap.Error(err))
		}
		log.Info("Database successfully initialized")
	} else {
		log.Info("Database already initialized")
	}

	return
}

func (db *Database) generatePostgresUrl() string {
	var extraKey string
	if !db.IsProduction {
		extraKey = "_TEST"
	}
	url := "postgres://" +
		utils.GetEnv("DB_USER", "user") +
		":" +
		utils.GetEnv("DB_PASS", "password") +
		"@" +
		utils.GetEnv("DB_HOST"+extraKey, "localhost") +
		":" +
		utils.GetEnv("DB_PORT"+extraKey, "5432") +
		"/" +
		utils.GetEnv("DB_NAME", "product_api") +
		"?sslmode=disable"
	return url
}

// initDb initializes the database connection pool and stores it in the global Db variable
func (db *Database) initDb(isProduction bool, logInfo bool) (err error) {
	db.IsProduction = isProduction

	// Set up the Ent client with postgres driver
	pdb, err := sql.Open("postgres", db.generatePostgresUrl())
	if err != nil {
		log.Error("Unable to connect to database", zap.Error(err))
		return
	}
	db.DB = pdb

	drv := entsql.OpenDB(dialect.Postgres, pdb)
	db.EntClient = ent.NewClient(ent.Driver(NewDebugDriver(drv)))

	// Check if database is reachable
	var greeting string
	greeting, err = db.GetHelloWorld()
	if err != nil {
		log.Error("InitDb failed", err)
		return
	}
	if logInfo {
		log.Info("Connection to database successful: ", greeting)
	}
	return
}

// CloseDbPool closes the database connection pool
func (db *Database) CloseDbPool() {
	if db.EntClient != nil {
		db.EntClient.Close()
	}
}

// EmptyDatabase truncates all tables in the database
func (db *Database) EmptyDatabase() (err error) {
	log.Info("Emptying database executed")
	if db.IsProduction {
		log.Fatal("Cannot empty production database")
		return
	}
	ctx := context.Background()

	// Show number of accounts existing in database before truncation
	count, err := db.EntClient.Account.Query().Count(ctx)
	if err != nil {
		log.Error("EmptyDatabase show number of accounts before truncation: ", err)
		return
	}
	log.Info("Number of accounts before truncation: ", count)

	// Execute truncation
	// "SELECT truncate_tables('user')" is a function call
	_, err = db.DB.ExecContext(ctx, "SELECT truncate_tables('user')")
	log.Info("Database emptied")
	if err != nil {
		log.Error("EmptyDatabase truncation failed: ", err)
	}

	// Show number of accounts existing in database after truncation
	count, err = db.EntClient.Account.Query().Count(ctx)
	if err != nil {
		log.Error("EmptyDatabase show number of accounts after truncation: ", err)
	}
	log.Info("Number of accounts after truncation: ", count)

	err = db.CheckRolePermissions()
	if err != nil {
		log.Error("CheckRolePermissions failed: ", err)
	}

	// Clear in-memory caches that may hold IDs from the previous database state
	// This prevents stale IDs (e.g., accountTypeIDCache) being reused after truncation
	accountTypeIDCache = make(map[string]int)

	return
}

func (db *Database) CheckRolePermissions() error {
	ctx := context.Background()
	// Log the current user (role) in use
	var currentUser string
	rows, err := db.DB.QueryContext(ctx, "SELECT current_user;")
	if err != nil {
		log.Error("Failed to get current user: ", err)
		return err
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.Scan(&currentUser); err != nil {
			log.Error("Failed to scan current user: ", err)
			return err
		}
	}
	log.Info("Current database user: ", currentUser)

	// Check TRUNCATE privilege on the 'Account' table
	var hasTruncate bool
	// QueryContext args are variadic interface{}
	rows2, err := db.DB.QueryContext(ctx, "SELECT has_table_privilege($1, 'Account', 'TRUNCATE');", currentUser)
	if err != nil {
		log.Error("Failed to check TRUNCATE privilege: ", err)
		return err
	}
	defer rows2.Close()
	if rows2.Next() {
		if err := rows2.Scan(&hasTruncate); err != nil {
			log.Error("Failed to scan permission: ", err)
			return err
		}
	}
	log.Info("User", currentUser, "has TRUNCATE privilege on 'Account': ", hasTruncate)

	return nil
}
