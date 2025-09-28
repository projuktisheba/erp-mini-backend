package dbrepo

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBRepository contains all individual repositories
type DBRepository struct {
	EmployeeRepo *EmployeeRepo
	AttendanceRepo *AttendanceRepo
	// SubscriptionRepo     *SubscriptionRepo
}


// NewDBRepository initializes all repositories with a shared connection pool
func NewDBRepository(db *pgxpool.Pool) *DBRepository {
	return &DBRepository{
		EmployeeRepo: NewEmployeeRepo(db),
		AttendanceRepo: NewAttendanceRepo(db),
	}
}
