package postgres

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	log "github.com/public-forge/go-logger"
)

type contextKey string

// TransactionContextKey is used as the context key to store transaction contexts.
const TransactionContextKey = contextKey("TransactionContextKey")

// Important errors related to transaction handling.
var (
	ErrTxWasRollbacked  = errors.New("the transaction has been rollbacked")               // ErrTxWasRollbacked occurs when a rollback has already been performed.
	ErrNotInTransaction = errors.New("not in a transaction, Begin() has not been called") // ErrNotInTransaction occurs when a transaction is expected but not started.

	DbConfig *PgConfig = nil // Global database configuration.
)

//go:generate mockgen -source=transaction_context.go -destination=./mock_transaction_context.go -package=postgres
type (
	// ITransactionContext provides methods for handling transactions, including nested transactions.
	//
	// Begin() starts a new transaction and returns a UUID to identify it.
	// Example:
	//   txContext, _ := GetTransactionContext(ctx)
	//   id, err := txContext.Begin()
	//   if err != nil { return err }
	//
	// Commit() expects the transaction ID to confirm the transaction’s ownership.
	//   err := txContext.Commit(id)
	//   if err != nil { return err }
	//
	// Rollback() affects the transaction at any level and is recommended to handle any errors.
	//   defer txContext.Rollback() // ensure rollback on any error
	//
	// Provider() returns the *gorm.DB instance, used for database operations within the transaction.
	//   db := txContext.Provider()
	//   db.Create(&modelInstance)
	//
	ITransactionContext interface {
		Begin() (uuid.UUID, error) // Begins a transaction and returns its UUID.
		Commit(uuid.UUID) error    // Commits the transaction if the caller holds the transaction UUID.
		Rollback() error           // Rolls back the transaction.
		Provider() *gorm.DB        // Returns the *gorm.DB instance for performing database operations.
	}

	// transactionContext contains transaction details and management logic.
	transactionContext struct {
		logger          log.Logger      // Logger for transaction activity.
		dbHolder        *DatabaseHolder // Database holder providing the connection.
		tx              *gorm.DB        // Database transaction instance.
		transactionUUID *uuid.UUID      // Unique identifier for the transaction.
		rollbacked      bool            // Indicates if the transaction has been rolled back.
	}
)

// GetTransactionContext retrieves or creates a transaction context and its associated context for use within functions.
// Example:
//
//	func doSomething(ctx context.Context) {
//	  txContext, newCtx := GetTransactionContext(ctx)
//	  id, err := txContext.Begin()
//	  if err != nil { return err }
//	  defer txContext.Rollback() // Rollback on any error
//	  // ... perform operations ...
//	  return txContext.Commit(id) // Commit if no errors
//	}
func GetTransactionContext(ctx context.Context) (ITransactionContext, context.Context) {
	return getTransactionContextWithDBHolder(ctx)
}

// getTransactionContextWithDBHolder retrieves an existing transaction context from the provided context.
// If no transaction context is found, it creates a new one and returns it with an updated context.
//
// Parameters:
//   - ctx: The current context from which to retrieve or add a transaction context.
//
// Returns:
//   - ITransactionContext: An interface representing the transaction context for managing database transactions.
//   - context.Context: The updated context containing the transaction context.
//
// The function checks if an `ITransactionContext` already exists in the provided context. If not, it creates a new
// instance of `transactionContext`, stores it in a new context, and returns both.
func getTransactionContextWithDBHolder(ctx context.Context) (ITransactionContext, context.Context) {
	// Check for the presence of an existing ITransactionContext in the context.
	transactionContext, found := ctx.Value(TransactionContextKey).(ITransactionContext)
	if !found {
		// If not found, create a new instance of transactionContext.
		transactionContext := newTransactionContext(log.FromContext(ctx), NewDBHolderInstance(DbConfig))
		newContext := context.WithValue(ctx, TransactionContextKey, transactionContext)
		return transactionContext, newContext
	}
	// Return the existing ITransactionContext.
	return transactionContext, ctx
}

// Begin starts a new transaction and returns its unique identifier.
// Example:
//
//	txContext, _ := GetTransactionContext(ctx)
//	id, err := txContext.Begin()
//	if err != nil { return err }
//	defer txContext.Rollback()
func (c *transactionContext) Begin() (id uuid.UUID, err error) {
	if c.wasRollbacked() {
		err = ErrTxWasRollbacked
		return
	}

	id, err = uuid.NewRandom()
	if err != nil {
		return
	}

	if !c.inTransaction() {
		c.transactionUUID = &id
		c.tx = c.dbHolder.dbConnection.Begin()

		if err = c.tx.Error; err != nil {
			c.logger.Errorf("cannot begin transaction (%v)", id)
			return
		}

		c.logger.Debugf("new transaction: %v", c.transactionUUID)
	} else {
		c.logger.Debugf("use existing transaction: %v", c.transactionUUID)
	}

	return
}

// Provider returns the *gorm.DB instance for database operations within the transaction.
// Example:
//
//	txContext, _ := GetTransactionContext(ctx)
//	db := txContext.Provider()
//	db.Create(&modelInstance)
func (c *transactionContext) Provider() *gorm.DB {
	if c.wasRollbacked() {
		c.logger.Error("transaction has been rolled back!")
		return nil
	}

	if c.inTransaction() {
		return c.tx
	}

	return c.providerWithoutTransaction()
}

// Commit finalizes the transaction, saving changes if the caller holds the transaction UUID.
// Example:
//
//	err := txContext.Commit(id)
//	if err != nil { return err }
func (c *transactionContext) Commit(id uuid.UUID) error {
	if c.wasRollbacked() {
		return ErrTxWasRollbacked
	}

	if !c.inTransaction() {
		return ErrNotInTransaction
	}

	// Only the transaction owner can commit.
	if *c.transactionUUID != id {
		return nil
	}

	defer c.dispose()

	if err := c.tx.Commit().Error; err != nil {
		c.logger.Errorf("cannot commit transaction: %v; err: %s", c.transactionUUID, err)
		return err
	}

	return nil
}

// Rollback cancels the transaction and discards changes made within it.
// Example:
//
//	defer txContext.Rollback() // ensure rollback on any error
func (c *transactionContext) Rollback() error {
	if c.wasRollbacked() {
		return ErrTxWasRollbacked
	}
	if !c.inTransaction() {
		c.logger.Debug("no active transaction to roll back")
		return nil
	}

	defer c.disposeAfterRollback()

	if err := c.tx.Rollback().Error; err != nil {
		c.logger.Errorf("cannot rollback (%v): %s", c.transactionUUID, err)
		return err
	}

	return nil
}

// inTransaction checks if a transaction is currently active.
func (c *transactionContext) inTransaction() bool {
	return c.tx != nil && c.transactionUUID != nil
}

// dispose clears transaction data after a successful commit or rollback.
func (c *transactionContext) dispose() {
	c.logger.Debugf("disposing transaction (%v)", c.transactionUUID)
	c.tx = nil
	c.transactionUUID = nil
}

// disposeAfterRollback marks the transaction as rolled back and disposes of it.
func (c *transactionContext) disposeAfterRollback() {
	c.rollbacked = true
	c.dispose()
}

// wasRollbacked returns true if the transaction has already been rolled back.
func (c *transactionContext) wasRollbacked() bool {
	return c.rollbacked
}

// providerWithoutTransaction returns the dbConnection without starting a new transaction.
func (c *transactionContext) providerWithoutTransaction() *gorm.DB {
	return c.dbHolder.dbConnection
}

// newTransactionContext creates a new instance of transactionContext with the given logger and dbHolder.
func newTransactionContext(logger log.Logger, dbHolder *DatabaseHolder) *transactionContext {
	return &transactionContext{logger: logger, dbHolder: dbHolder}
}

// Interface compliance check
var _ ITransactionContext = (*transactionContext)(nil)
