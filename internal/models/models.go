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
	PostalCode   string    `json:"postal_code"`
	VatID        string    `json:"tax_id"` //tax_id
	BaseSalary   float64   `json:"base_salary"`
	OvertimeRate float64   `json:"overtime_rate"`
	AvatarLink   string    `json:"avatar_link"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
