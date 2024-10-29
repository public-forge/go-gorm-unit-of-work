package postgres

import (
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"testing"
)

// Mock PgConfig struct for testing purposes
var mockPgConfig = &PgConfig{
	Host:                    "localhost",
	User:                    "testuser",
	Password:                "testpassword",
	DBName:                  "testdb",
	Schema:                  "public",
	LogMode:                 true,
	MaxOpenConnections:      5,
	ConnectionMaxLifetimeMS: 60000,
}

// Test Open function with successful connection
func TestOpen_Success(t *testing.T) {
	db, mock, err := sqlmock.New() // create a sqlmock instance
	assert.NoError(t, err)

	mock.ExpectPing() // expect a successful ping
	gormDB, err := gorm.Open("postgres", db)
	assert.NoError(t, err)
	assert.NotNil(t, gormDB)

	defer gormDB.Close()
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// Test CheckConnection function to verify the "SELECT 1" query
func TestCheckConnection(t *testing.T) {
	db, mock, err := sqlmock.New() // create a sqlmock instance
	assert.NoError(t, err)

	gormDB, err := gorm.Open("postgres", db)
	assert.NoError(t, err)
	defer gormDB.Close()

	mock.ExpectExec("SELECT 1;").WillReturnResult(sqlmock.NewResult(1, 1)) // expect SELECT 1 query
	CheckConnection(gormDB)
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// Test setSQLSettings to ensure SQL settings are applied as configured
func TestSetSQLSettings(t *testing.T) {
	db, mock, err := sqlmock.New() // create a sqlmock instance
	assert.NoError(t, err)

	gormDB, err := gorm.Open("postgres", db)
	assert.NoError(t, err)
	defer gormDB.Close()

	sqlDB := gormDB.DB()
	setSQLSettings(sqlDB, mockPgConfig)

	assert.Equal(t, mockPgConfig.MaxOpenConnections, sqlDB.Stats().MaxOpenConnections)
	//assert.Equal(t, time.Duration(mockPgConfig.ConnectionMaxLifetimeMS)*time.Millisecond, sqlDB.ConnMaxLifetime())
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// Test setGORMSettings to verify GORM-specific configurations
func TestSetGORMSettings(t *testing.T) {
	db, mock, err := sqlmock.New() // create a sqlmock instance
	assert.NoError(t, err)

	gormDB, err := gorm.Open("postgres", db)
	assert.NoError(t, err)
	defer gormDB.Close()

	setGORMSettings(gormDB, mockPgConfig)
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
