package models

import "time"

const (
	APPName    = "ERP Mini"
	APPVersion = "1.0"
)

var Passphrase = "jM/0qr%HKU&!G%MdivH#A-{oInY*Nv20"

// Response is the type for response
type Response struct {
	Error   bool   `json:"error"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// User holds the user info
type JWT struct {
	ID        int       `json:"id"`
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
	Port int
	Env  string
	JWT  JWTConfig
	DB   DBConfig
}

// Employee model
type Employee struct {
	ID           int       `json:"id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Role         string    `json:"role"`   //admin //employee
	Status       string    `json:"status"` //active //inactive
	Bio          string    `json:"bio"`
	Email        string    `json:"email"` //username
	Password     string    `json:"-"`     // don't expose
	Mobile       string    `json:"mobile"`
	Country      string    `json:"country"`
	City         string    `json:"city"`
	Address      string    `json:"address"`
	PostalCode   string    `json:"postal_code"`
	TaxID        string    `json:"tax_id"` //tax_id
	BaseSalary   float64   `json:"base_salary"`
	OvertimeRate float64   `json:"overtime_rate"`
	AvatarLink   string    `json:"avatar_link"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Attendance struct {
	ID            int       `json:"id"`
	EmployeeID    int       `json:"employee_id"`
	WorkDateStr   string    `json:"work_date"`
	WorkDate      time.Time `json:"-"`
	Status        string    `json:"status"`
	CheckInStr    string    `json:"checkin"`
	CheckOutStr   string    `json:"checkout"`
	CheckIn       time.Time `json:"-"`
	CheckOut      time.Time `json:"-"`
	OvertimeHours int       `json:"overtime_hours"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type AttendanceSummary struct {
	EmployeeID         int    `json:"employee_id"`
	EmployeeName       string `json:"employee_name"`
	TotalWorkingDays   int    `json:"total_working_days"`
	PresentDays        int    `json:"present_days"`
	AbsentDays         int    `json:"absent_days"`
	LeaveDays          int    `json:"leave_days"`
	TotalOvertimeHours int    `json:"total_overtime_hours"`
}

type EmployeeCalendar struct {
	EmployeeID   int           `json:"employee_id"`
	EmployeeName string        `json:"employee_name"`
	Month        string        `json:"month"`
	Attendance   []*Attendance `json:"attendance"`
}
