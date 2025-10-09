package api

import (
	"log"

	"github.com/projuktisheba/erp-mini-api/internal/dbrepo"
	"github.com/projuktisheba/erp-mini-api/internal/models"
)

type HandlerRepo struct {
	Employee EmployeeHandler	
	Auth AuthHandler
	Attendance AttendanceHandler
	Customer CustomerHandler
	Order OrderHandler
	Transaction TransactionHandler
	Account *AccountHandler
	Product *ProductHandler
	Report *ReportHandler
	Supplier *SupplierHandler
	Purchase *PurchaseHandler
}

func NewHandlerRepo( db *dbrepo.DBRepository,JWT models.JWTConfig, infoLog *log.Logger, errorLog *log.Logger) *HandlerRepo {
	return &HandlerRepo{
		Employee: *NewEmployeeHandler(db.EmployeeRepo, infoLog, errorLog),
		Auth: *NewAuthHandler( db,JWT, infoLog, errorLog),
		Attendance: *NewAttendanceHandler( db.AttendanceRepo,infoLog, errorLog),
		Customer: *NewCustomerHandler(db.CustomerRepo, infoLog, errorLog),
		Order: *NewOrderHandler(db.OrderRepo, infoLog, errorLog),
		Transaction: *NewTransactionHandler(db.TransactionRepo, infoLog, errorLog),
		Account: NewAccountHandler(db.AccountRepo,infoLog,errorLog),
		Product: NewProductHandler(db.ProductRepo, infoLog, errorLog),
		Report: NewReportHandler(db.ReportRepo, infoLog, errorLog),
		Supplier: NewSupplierHandler(db.SupplierRepo, infoLog, errorLog),
		Purchase: NewPurchaseHandler(db.PurchaseRepo, infoLog, errorLog),
	}
}
