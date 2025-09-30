package dbrepo

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBRepository contains all individual repositories
type DBRepository struct {
	EmployeeRepo *EmployeeRepo
	AttendanceRepo *AttendanceRepo
	CustomerRepo *CustomerRepo
	OrderRepo *OrderRepo
	TransactionRepo *TransactionRepo
	AccountRepo *AccountRepo
	ProductRepo *ProductRepo
	ReportRepo *ReportRepo
}


// NewDBRepository initializes all repositories with a shared connection pool
func NewDBRepository(db *pgxpool.Pool) *DBRepository {
	return &DBRepository{
		EmployeeRepo: NewEmployeeRepo(db),		
		AttendanceRepo: NewAttendanceRepo(db),
		CustomerRepo: NewCustomerRepo(db),
		OrderRepo: NewOrderRepo(db),
		TransactionRepo: NewTransactionRepo(db),
		AccountRepo: NewAccountRepo(db),
		ProductRepo: NewProductRepo(db),
		ReportRepo: NewReportRepo(db),

	}
}
