package sql

import "database/sql"

// GetTxFromProductRepo is a test helper to extract transaction from ProductRepository.
func GetTxFromProductRepo(repo *ProductRepository) *sql.Tx {
	return repo.txn
}

// GetTxFromEventRepo is a test helper to extract transaction from EventRepository.
func GetTxFromEventRepo(repo *EventRepository) *sql.Tx {
	return repo.txn
}
