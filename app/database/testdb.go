package database

import (
	"os"
	"os/exec"
)

const (
    dbName      = "your_existing_db"
    dbUser      = "your_db_user"
    dbPassword  = "your_db_password"
    dbHost      = "localhost" // Change to your database host
    dumpFile    = "dumpfile.sql" // Choose a name for the dump file
    newDbName   = "new_db" // Name for the new duplicated database
)

func dumpDatabase() error {

	os.Setenv("PATH", "/path/to/postgresql/bin:"+os.Getenv("PATH"))

    cmd := exec.Command("pg_dump",
        "-h", dbHost,
        "-U", dbUser,
        "-d", dbName,
        "-f", dumpFile,
    )

    cmd.Env = append(os.Environ(), "PGPASSWORD="+dbPassword)

    output, err := cmd.CombinedOutput()
    if err != nil {
        log.Info("Error dumping database: %v\n%s", err, output)
        return err
    }

    log.Info("Database dumped successfully")
    return nil
}

func restoreDatabase() error {
    cmd := exec.Command("pg_restore",
        "-h", dbHost,
        "-U", dbUser,
        "-d", newDbName,
        dumpFile,
    )

    cmd.Env = append(os.Environ(), "PGPASSWORD="+dbPassword)

    output, err := cmd.CombinedOutput()
    if err != nil {
        log.Info("Error restoring database: %v\n%s", err, output)
        return err
    }

    log.Info("Database restored successfully")
    return nil
}

func createDatabase() error {
    cmd := exec.Command("createdb",
        "-h", dbHost,
        "-U", dbUser,
        newDbName,
    )

    cmd.Env = append(os.Environ(), "PGPASSWORD="+dbPassword)

    output, err := cmd.CombinedOutput()
    if err != nil {
        log.Info("Error creating database: %v\n%s", err, output)
        return err
    }

    log.Info("Database %s created successfully", newDbName)
    return nil
}
