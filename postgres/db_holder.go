package postgres

import (
	"github.com/jinzhu/gorm"
	"sync"

	// driver for postgres
	_ "github.com/lib/pq"
)

// NewDBHolderInstance initializes and returns a singleton instance of DatabaseHolder.
// It ensures that only one instance of DatabaseHolder is created, even in concurrent contexts.
func NewDBHolderInstance(config *PgConfig) *DatabaseHolder {
	onceDBHolder.Do(func() {
		connect := NewConnect(config)   // Establishes a new database connection.
		dbHolder = NewDBHolder(connect) // Creates a new DatabaseHolder with the connection.
	})

	return dbHolder
}

var (
	dbHolder     *DatabaseHolder // Singleton instance of DatabaseHolder
	onceDBHolder sync.Once       // Ensures single initialization of dbHolder
)

// DatabaseHolder wraps a gorm.DB database connection, providing a centralized way to access it.
type DatabaseHolder struct {
	dbConnection *gorm.DB // Holds the actual database connection.
}

// NewDBHolder creates a new DatabaseHolder with the given gorm.DB connection.
func NewDBHolder(db *gorm.DB) *DatabaseHolder {
	return &DatabaseHolder{db} // Initializes DatabaseHolder with the provided db connection.
}
