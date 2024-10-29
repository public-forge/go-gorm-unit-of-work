[![Lint and Format](https://github.com/public-forge/gorm-unit-of-work/actions/workflows/lint.yml/badge.svg)](https://github.com/public-forge/gorm-unit-of-work/actions/workflows/lint.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/public-forge/gorm-unit-of-work)](https://goreportcard.com/report/github.com/public-forge/gorm-unit-of-work)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

### README for `postgres` Package

The `postgres` package provides a set of tools to manage PostgreSQL database connections and transactions using GORM. It offers singleton database handling, transaction management, and configuration support for SQL settings.

---

#### 1. **Configuration Setup**

Start by defining your PostgreSQL database configuration in a `PgConfig` struct. This struct holds the database connection parameters and GORM-specific configurations.

Example:
```go
import "github.com/public-forge/go-gorm-unit-of-work/postgres"

config := postgres.PgConfig{
    Host:                    "localhost",
    DBName:                  "your_database",
    Schema:                  "public",
    User:                    "your_username",
    Password:                "your_password",
    MaxOpenConnections:      10,
    ConnectionMaxLifetimeMS: 60000,
    LogMode:                 true,
}
```

#### 2. **Connecting to the Database**

To establish a connection, use the `NewConnect` function, which tries to connect to the database based on your configuration and retries on failure.

```go
db := postgres.NewConnect(&config)
defer db.Close()
```

#### 3. **Using the DatabaseHolder Singleton**

`DatabaseHolder` is a singleton for managing database connections across different parts of the application. Use `NewDBHolderInstance` to get an instance:

```go
dbHolder := postgres.NewDBHolderInstance(&config)
```

#### 4. **Managing Transactions**

To manage transactions, use the `ITransactionContext` interface, which provides methods for `Begin`, `Commit`, and `Rollback` operations.

1. **Creating a Transaction Context**

   Use `GetTransactionContext` to create a transaction context and initiate transactions within a specific context.

   ```go
   ctx := context.Background()
   txContext, newCtx := postgres.GetTransactionContext(ctx)
   ```

2. **Starting and Committing a Transaction**

   Start a transaction using `Begin`, and commit it with `Commit`.

   ```go
   id, err := txContext.Begin()
   if err != nil {
       log.Fatalf("Transaction begin failed: %v", err)
   }

   // Perform database operations here
   db := txContext.Provider()
   db.Create(&yourModel)

   // Commit the transaction
   if err := txContext.Commit(id); err != nil {
       log.Fatalf("Transaction commit failed: %v", err)
   }
   ```

3. **Rolling Back a Transaction**

   Rollback the transaction if an error occurs or if you want to cancel operations within the transaction.

   ```go
   defer txContext.Rollback()
   ```

#### 5. **Testing with Mocked Database**

To run tests without connecting to an actual database, use `getTestTransactionContext` to set up a mocked transaction context using `go-sqlmock`.

Example:
```go
func TestExampleTransaction(t *testing.T) {
    tx, db, mock := postgres.getTestTransactionContext(t)
    defer db.Close()
    mock.ExpectBegin()
    
    id, err := tx.Begin()
    if err != nil {
        t.Fatalf("failed to begin transaction: %v", err)
    }

    // Test database operations within the transaction
    mock.ExpectCommit()
    if err := tx.Commit(id); err != nil {
        t.Fatalf("failed to commit transaction: %v", err)
    }
}
```

#### 6. **Error Handling**

Common errors:
- `ErrTxWasRollbacked` — transaction was already rolled back.
- `ErrNotInTransaction` — attempted to commit or roll back without starting a transaction.

#### Additional Notes

- The `postgres` package uses GORM for ORM operations, so be familiar with its API.
- This package supports nested transactions, allowing `Commit` calls within nested functions to be ignored if they’re not the transaction owner.
- Use `CheckConnection` to validate active database connections.

This README provides a quick overview of the main functions and usage examples for the `postgres` package. Adjust connection parameters and test functions according to your project requirements.


