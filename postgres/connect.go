package postgres

import (
	"database/sql"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	log "github.com/public-forge/go-logger"
	"time"
)

const (
	// defaultConnectionNumberOfRetries defines the maximum number of connection retries.
	defaultConnectionNumberOfRetries = 8
	// defaultConnectionSecondsBetweenRetries defines the delay in seconds between each retry.
	defaultConnectionSecondsBetweenRetries = 4
)

// NewConnect establishes a new connection to the PostgreSQL database using the provided configuration.
// It retries on failure and panics if connection attempts are exhausted.
func NewConnect(config *PgConfig) *gorm.DB {
	logger := log.FromDefaultContext()
	db, err := Open(config)
	if err != nil {
		logger.Infof("can't connect to db (connect error): %v", err)
		panic(err)
	}
	return db
}

// CheckConnection executes a basic query to verify the database connection is still active.
func CheckConnection(db *gorm.DB) {
	db.Exec("SELECT 1;")
}

// Open attempts to open a database connection using the provided PgConfig settings.
// If the connection fails, it will retry based on default retry parameters.
// On success, it applies SQL and GORM-specific configurations.
func Open(cfg *PgConfig) (db *gorm.DB, err error) {
	logger := log.FromDefaultContext()
	for retry := 0; retry < defaultConnectionNumberOfRetries; retry++ {
		logger.Infof("Connecting to postgres %s@%s... (retry %d of %d)",
			cfg.DBName, cfg.Host, retry, defaultConnectionNumberOfRetries)

		db, err = gorm.Open("postgres", fmt.Sprintf(`
			host=%s
			user=%s
			password=%s
			dbname=%s
			search_path=%s
			sslmode=disable
		`, cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Schema))

		// Log and retry on failure
		if err != nil {
			logger.Errorf("Connecting to postgres %s@%s FAILED: %s",
				cfg.DBName, cfg.Host, err)

			time.Sleep(defaultConnectionSecondsBetweenRetries * time.Second)
			continue
		}
		db.SetLogger(logger)
		// Log on successful connection
		logger.Infof("Successfully connected to postgres %s@%s", cfg.DBName, cfg.Host)

		// Apply database settings
		setSQLSettings(db.DB(), cfg)
		setGORMSettings(db, cfg)

		return
	}
	logger.Fatalf("Connecting to postgres %s@%s FAILED", cfg.DBName, cfg.Host)
	return
}

// setGORMSettings configures GORM-specific settings, such as enabling or disabling log mode.
func setGORMSettings(db *gorm.DB, pgConfig *PgConfig) {
	db.LogMode(pgConfig.LogMode)
}

// setSQLSettings applies SQL settings, including max open connections and connection lifetime.
func setSQLSettings(db *sql.DB, pgConfig *PgConfig) {
	db.SetMaxOpenConns(pgConfig.MaxOpenConnections)
	db.SetConnMaxLifetime(time.Duration(pgConfig.ConnectionMaxLifetimeMS) * time.Millisecond)
}
