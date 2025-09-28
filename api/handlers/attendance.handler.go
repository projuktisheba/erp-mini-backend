package api

import (
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/projuktisheba/erp-mini-api/internal/dbrepo"
	"github.com/projuktisheba/erp-mini-api/internal/models"
	"github.com/projuktisheba/erp-mini-api/internal/utils"
)

// AttendanceHandler handles attendance-related requests
type AttendanceHandler struct {
	DB       *dbrepo.AttendanceRepo
	infoLog  *log.Logger
	errorLog *log.Logger
}

func NewAttendanceHandler(db *dbrepo.AttendanceRepo, infoLog *log.Logger, errorLog *log.Logger) *AttendanceHandler {
	return &AttendanceHandler{
		DB:       db,
		infoLog:  infoLog,
		errorLog: errorLog,
	}
}

// UpdateTodayAttendance updates attendance for the current day of a single employee
func (a *AttendanceHandler) UpdateTodayAttendance(w http.ResponseWriter, r *http.Request) {
	var req models.Attendance
	err := utils.ReadJSON(w, r, &req)
	if err != nil {
		a.errorLog.Println("ERROR_01_UpdateTodayAttendance:", err)
		utils.BadRequest(w, err)
		return
	}

	if req.EmployeeID == 0 {
		a.errorLog.Println("ERROR_02_UpdateTodayAttendance: Missing employee ID")
		utils.BadRequest(w, errors.New("missing employee ID"))
		return
	}

	err = a.DB.UpdateTodayAttendance(r.Context(), req.EmployeeID, req.Status, req.CheckIn, req.CheckOut, req.OvertimeHours)
	if err != nil {
		a.errorLog.Println("ERROR_03_UpdateTodayAttendance:", err)
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error   bool   `json:"error"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Attendance updated successfully"

	utils.WriteJSON(w, http.StatusOK, resp)
}

// BatchUpdateTodayAttendance updates today's attendance for multiple employees
func (a *AttendanceHandler) BatchUpdateTodayAttendance(w http.ResponseWriter, r *http.Request) {
	var req []*models.Attendance
	err := utils.ReadJSON(w, r, &req)
	if err != nil {
		a.errorLog.Println("ERROR_01_BatchUpdateTodayAttendance:", err)
		utils.BadRequest(w, err)
		return
	}

	if len(req) == 0 {
		a.errorLog.Println("ERROR_02_BatchUpdateTodayAttendance: Empty request payload")
		utils.BadRequest(w, errors.New("no employees provided"))
		return
	}

	err = a.DB.BatchUpdateTodayAttendance(r.Context(), req)
	if err != nil {
		a.errorLog.Println("ERROR_03_BatchUpdateTodayAttendance:", err)
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error   bool   `json:"error"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Batch attendance updated successfully"

	utils.WriteJSON(w, http.StatusOK, resp)
}

// GetEmployeeCalendar fetches calendar-style attendance for an employee (month or date range)
func (a *AttendanceHandler) GetEmployeeCalendar(w http.ResponseWriter, r *http.Request) {
	employeeID := chi.URLParam(r, "employeeID")
	month := r.URL.Query().Get("month")
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")

	if employeeID == "" {
		a.errorLog.Println("ERROR_01_GetEmployeeCalendar: Missing employee ID")
		utils.BadRequest(w, errors.New("missing employee ID"))
		return
	}

	calendar, err := a.DB.GetEmployeeCalendar(r.Context(), employeeID, month, start, end)
	if err != nil {
		a.errorLog.Println("ERROR_02_GetEmployeeCalendar:", err)
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error      bool                     `json:"error"`
		Status     string                   `json:"status"`
		Message    string                   `json:"message"`
		Attendance *models.EmployeeCalendar `json:"attendance"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Attendance calendar fetched successfully"
	resp.Attendance = calendar

	utils.WriteJSON(w, http.StatusOK, resp)
}

// GetEmployeeSummary fetches monthly attendance summary for one employee
func (a *AttendanceHandler) GetEmployeeSummary(w http.ResponseWriter, r *http.Request) {
	employeeID := chi.URLParam(r, "employeeID")
	month := r.URL.Query().Get("month")

	if employeeID == "" {
		a.errorLog.Println("ERROR_01_GetEmployeeSummary: Missing employee ID")
		utils.BadRequest(w, errors.New("missing employee ID"))
		return
	}

	summary, err := a.DB.GetEmployeeSummary(r.Context(), employeeID, month)
	if err != nil {
		a.errorLog.Println("ERROR_02_GetEmployeeSummary:", err)
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error   bool                      `json:"error"`
		Status  string                    `json:"status"`
		Message string                    `json:"message"`
		Summary *models.AttendanceSummary `json:"summary"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Attendance summary fetched successfully"
	resp.Summary = summary

	utils.WriteJSON(w, http.StatusOK, resp)
}

// GetBatchSummary fetches monthly attendance summaries for multiple employees
func (a *AttendanceHandler) GetBatchSummary(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")

	summaries, err := a.DB.GetBatchSummary(r.Context(), month, start, end)
	if err != nil {
		a.errorLog.Println("ERROR_01_GetBatchSummary:", err)
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error     bool                       `json:"error"`
		Status    string                     `json:"status"`
		Message   string                     `json:"message"`
		Summaries []models.AttendanceSummary `json:"summaries"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Batch attendance summaries fetched successfully"
	resp.Summaries = summaries

	utils.WriteJSON(w, http.StatusOK, resp)
}
