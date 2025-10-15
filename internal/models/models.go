package models

import (
	"time"
)

const (
	APPName    = "ERP Mini"
	APPVersion = "1.0"
)

// Response is the type for response
type Response struct {
	Error   bool   `json:"error"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// User holds the user info
type JWT struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	Issuer    string    `json:"iss"`
	Audience  string    `json:"aud"`
	ExpiresAt int64     `json:"exp"`
	IssuedAt  int64     `json:"iat"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type JWTConfig struct {
	SecretKey string
	Issuer    string
	Audience  string
	Algorithm string
	Expiry    time.Duration
	Refresh   time.Duration
}

type DBConfig struct {
	DSN    string
	DEVDSN string
}

type Config struct {
	Port int64
	Env  string
	JWT  JWTConfig
	DB   DBConfig
}

// Employee model
type Employee struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Role         string    `json:"role"`   // chairman, manager, salesperson, worker
	Status       string    `json:"status"` // active, inactive
	Mobile       string    `json:"mobile"`
	Email        string    `json:"email,omitempty"`
	Password     string    `json:"-"` // hashed password
	PassportNo   string    `json:"passport_no,omitempty"`
	JoiningDate  time.Time `json:"joining_date"`
	Address      string    `json:"address,omitempty"`
	BaseSalary   float64   `json:"base_salary"`
	OvertimeRate float64   `json:"overtime_rate"`
	AvatarLink   string    `json:"avatar_link"`
	BranchID     int64     `json:"branch_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// EmployeeNameID is a lightweight struct for fetching only customer's ID and Name.
type EmployeeNameID struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Supplier represents the suppliers table
type Supplier struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	BranchID  int64     `json:"branch_id"`
	Status    string    `json:"status"` // active/inactive
	Mobile    string    `json:"mobile"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Purchase represents the purchase table
type Purchase struct {
	ID           int64     `json:"id"`
	MemoNo       string    `json:"memo_no"`
	PurchaseDate time.Time `json:"purchase_date"`
	SupplierID   int64     `json:"supplier_id"`
	SupplierName string    `json:"supplier_name"`
	BranchID     int64     `json:"branch_id"`
	TotalAmount  float64   `json:"total_amount"`
	Notes        string    `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Customer struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Mobile    string  `json:"mobile"`
	Address   string  `json:"address"`
	TaxID     *string `json:"tax_id,omitempty"`
	DueAmount float64 `json:"due_amount"`
	Status    bool    `json:"status"`
	BranchID  int64   `json:"branch_id"`
	//Measurement
	Length       string    `json:"length,omitempty"`
	Shoulder     string    `json:"shoulder,omitempty"`
	Bust         string    `json:"bust,omitempty"`
	Waist        string    `json:"waist,omitempty"`
	Hip          string    `json:"hip,omitempty"`
	ArmHole      string    `json:"arm_hole,omitempty"`
	SleeveLength string    `json:"sleeve_length,omitempty"`
	SleeveWidth  string    `json:"sleeve_width,omitempty"`
	RoundWidth   string    `json:"round_width,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CustomerNameID is a lightweight struct for fetching only customer's ID and Name.
type CustomerNameID struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Mobile string `json:"mobile"`
}

type Attendance struct {
	ID              int64     `json:"id"`
	EmployeeID      int64     `json:"employee_id"`
	WorkDateStr     string    `json:"work_date"`
	WorkDate        time.Time `json:"-"`
	Status          string    `json:"status"`
	OvertimeHours   int64     `json:"overtime_hours"`
	AdvancePayment  int64     `json:"advance_payment"`
	ProductionUnits int64     `json:"production_units"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type AttendanceSummary struct {
	EmployeeID         int64  `json:"employee_id"`
	EmployeeName       string `json:"employee_name"`
	TotalWorkingDays   int64  `json:"total_working_days"`
	PresentDays        int64  `json:"present_days"`
	AbsentDays         int64  `json:"absent_days"`
	LeaveDays          int64  `json:"leave_days"`
	TotalOvertimeHours int64  `json:"total_overtime_hours"`
}

type EmployeeCalendar struct {
	EmployeeID   int64         `json:"employee_id"`
	EmployeeName string        `json:"employee_name"`
	Month        string        `json:"month"`
	Attendance   []*Attendance `json:"attendance"`
}

type Product struct {
	ID                int64     `json:"id"`
	ProductName       string    `json:"product_name"`
	Quantity          int64     `json:"quantity"`
	TotalPrices       int64     `json:"total_price"`
	CurrentStockLevel int64     `json:"current_stock_level"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
type Sale struct {
	SaleDate           time.Time  `json:"sale_date"`
	MemoNo             string     `json:"memo_no"`
	BranchID             string     `json:"branch_id"`
	SalespersonID      int64      `json:"salesperson_id"`
	SalespersonName    string     `json:"salesperson_name"`
	CustomerID         int64      `json:"customer_id"`
	CustomerName       string     `json:"customer_name"`
	TotalPayableAmount float64    `json:"total_payable_amount"`
	PaidAmount         float64    `json:"paid_amount"`
	DueAmount          float64    `json:"due_amount"`
	PaymentAccountID   int64      `json:"payment_account_id"`
	PaymentAccountName   string      `json:"payment_account_name"`
	Notes              string     `json:"notes"`
	Items              []*Product `json:"items"`
}

// ProductStockRegistry represents a record from product_stock_registry
type ProductStockRegistry struct {
	ID          int64     `json:"id"`
	MemoNo      string    `json:"memo_no"`
	StockDate   time.Time `json:"stock_date"`
	BranchID    int64     `json:"branch_id"`
	BranchName  string    `json:"branch_name"`
	ProductID   int64     `json:"product_id"`
	ProductName string    `json:"product_name"`
	Quantity    int64     `json:"quantity"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type Order struct {
	ID                   int64        `json:"id"`
	BranchID             int64        `json:"branch_id"`
	MemoNo               string       `json:"memo_no"`
	OrderDate            time.Time    `json:"order_date"`
	SalespersonID        int64        `json:"salesperson_id"`
	SalespersonName      string       `json:"salesperson_name,omitempty"`
	SalespersonMobile    string       `json:"salesperson_mobile,omitempty"`
	CustomerID           int64        `json:"customer_id"`
	CustomerName         string       `json:"customer_name,omitempty"`
	CustomerMobile       string       `json:"customer_mobile,omitempty"`
	TotalPayableAmount   float64      `json:"total_payable_amount"`
	AdvancePaymentAmount float64      `json:"advance_payment_amount"`
	PaymentAccountID     *int64       `json:"payment_account_id,omitempty"`
	Status               string       `json:"status"` // pending, checkout, delivery, cancelled
	DeliveryDate         *time.Time   `json:"delivery_date,omitempty"`
	ExitDate             *time.Time   `json:"exit_date,omitempty"`
	Notes                *string      `json:"notes,omitempty"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`
	Items                []*OrderItem `json:"items,omitempty"`
}

type OrderItem struct {
	ID          int64   `json:"id"`
	MemoNo      string  `json:"memo_no"` // foreign key to orders.memo_no
	ProductID   int64   `json:"product_id"`
	ProductName string  `json:"product_name,omitempty"`
	Quantity    int64   `json:"quantity"`
	TotalPrice  float64 `json:"total_price"`
}

type Account struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	CurrentBalance float64   `json:"current_balance"`
	BranchID       int64     `json:"branch_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
type AccountNameID struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Transaction struct {
	ID              int64     `json:"id"`
	TransactionID   string    `json:"transaction_id"` // optional unique identifier if needed
	BranchID        int64     `json:"branch_id"`
	FromID          int64     `json:"from_id"`
	FromAccountName string    `json:"from_account_name"`
	FromType        string    `json:"from_type"` // customers, employees, accounts, etc.
	ToID            int64     `json:"to_id"`
	ToAccountName   string    `json:"to_account_name"`
	ToType          string    `json:"to_type"` // customers, employees, accounts, etc.
	Amount          float64   `json:"amount"`
	TransactionType string    `json:"transaction_type"` // payment, refund, adjustment, salary
	CreatedAt       time.Time `json:"created_at"`
	Notes           string    `json:"notes,omitempty"`
}

// Reports
type OrderOverview struct {
	TotalOrders     int64 `json:"total_orders"`
	PendingOrders   int64 `json:"pending_orders"`
	CheckoutOrders  int64 `json:"checkout_orders"`
	CompletedOrders int64 `json:"completed_orders"`
	CancelledOrders int64 `json:"cancelled_orders"`

	TotalOrdersAmount     int64 `json:"total_orders_amount"`
	PendingOrdersAmount   int64 `json:"pending_orders_amount"`
	CheckoutOrdersAmount  int64 `json:"checkout_orders_amount"`
	CompletedOrdersAmount int64 `json:"completed_orders_amount"`
	CancelledOrdersAmount int64 `json:"cancelled_orders_amount"`
}

type SalesPersonProgressReport struct {
	SalesPersonID   int64   `json:"sales_person_id"`
	SalesPersonName string  `json:"sales_person_name"`
	Mobile          string  `json:"mobile"`
	Email           string  `json:"email"`
	BaseSalary      float64 `json:"base_salary"`
	Date            string  `json:"date"`
	ProductName     string  `json:"product_name"`
	OrderCount      int64   `json:"order_count"`
	ItemCount       int64   `json:"item_count"`
	Sale            float64 `json:"sale"`
	SaleReturn      float64 `json:"sale_return"`
}

type WorkerProgressReport struct {
	WorkerID             int64   `json:"worker_id"`
	WorkerName           string  `json:"worker_name"`
	Mobile               string  `json:"mobile"`
	Email                string  `json:"email"`
	BaseSalary           float64 `json:"base_salary"`
	Date                 string  `json:"date"`
	TotalAdvancePayment  float64 `json:"total_advance_payment"`
	TotalProductionUnits float64 `json:"total_production_units"`
	TotalOvertimeHours   float64 `json:"total_overtime_hours"`
}

type TopSheet struct {
	ID          int64     `json:"id"`
	Date        time.Time `json:"date"`
	BranchID    int64     `json:"branch_id"`
	TotalAmount float64   `json:"total_amount"`
	Expense     float64   `json:"expense"`
	Cash        float64   `json:"cash"`
	Bank        float64   `json:"bank"`
	Balance     float64   `json:"balance"`
	OrderCount  int64     `json:"order_count"`
	Delivery    int64     `json:"delivery"`
	Checkout    int64     `json:"checkout"`
	ReadyMade   int64     `json:"ready_made"`
}

// Salesperson progress struct
type SalespersonProgress struct {
	Date             time.Time `json:"sheet_date"`
	BranchID         int64     `json:"branch_id"`
	EmployeeID       int64     `json:"employee_id"`
	SaleAmount       float64   `json:"sale_amount"`
	SaleReturnAmount float64   `json:"sale_return_amount"`
	OrderCount       int64     `json:"order_count"`
	Salary           float64   `json:"salary"`
}

// Worker progress struct
type WorkerProgress struct {
	Date            time.Time `json:"work_date"`
	BranchID        int64     `json:"branch_id"`
	EmployeeID      int64     `json:"employee_id"`
	ProductionUnits int64     `json:"production_units"`
	OvertimeHours   int16     `json:"overtime_hours"`
	AdvancePayment  float64   `json:"advance_payment"`
	Salary          float64   `json:"salary"`
}

type SalaryRecord struct {
    EmployeeID   int64     `json:"employee_id"`
    EmployeeName string    `json:"employee_name"`
    Role         string    `json:"role"`
    BaseSalary   float64   `json:"base_salary"`
    OvertimeRate float64   `json:"overtime_rate"`
    TotalSalary  float64   `json:"total_salary"`
    SheetDate    time.Time `json:"sheet_date"`
}
