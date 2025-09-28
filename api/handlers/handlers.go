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
}

func NewHandlerRepo( db *dbrepo.DBRepository,JWT models.JWTConfig, infoLog *log.Logger, errorLog *log.Logger) *HandlerRepo {
	return &HandlerRepo{
		Employee: *NewEmployeeHandler(db.EmployeeRepo, infoLog, errorLog),
		Auth: *NewAuthHandler( db,JWT, infoLog, errorLog),
		Attendance: *NewAttendanceHandler( db.AttendanceRepo,infoLog, errorLog),
	}
}
