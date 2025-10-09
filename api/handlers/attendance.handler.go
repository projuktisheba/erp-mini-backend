package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

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

// MarkEmployeePresent marks a single employee's attendance as present for today.
func (a *AttendanceHandler) MarkEmployeePresent(w http.ResponseWriter, r *http.Request) {
	// Read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		a.errorLog.Println("ERROR_01_MarkEmployeePresent: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}

	var reqBody models.Attendance

	// Parse JSON request
	err := utils.ReadJSON(w, r, &reqBody)
	if err != nil {
		a.errorLog.Println("ERROR_01_UpdateTodayAttendance:", err)
		utils.BadRequest(w, err)
		return
	}

	// Check required field
	if reqBody.EmployeeID == 0 {
		a.errorLog.Println("ERROR_02_UpdateTodayAttendance: Missing employee ID")
		utils.BadRequest(w, errors.New("missing employee ID"))
		return
	}

	workDate, err := time.Parse("2006-01-02", reqBody.WorkDateStr)
	if err != nil {
		a.errorLog.Printf("ERROR_XX: Invalid work_date %q for employee %d", reqBody.WorkDate, reqBody.EmployeeID)
		utils.BadRequest(w, fmt.Errorf("invalid work_date format for employee %d, expected YYYY-MM-DD", reqBody.EmployeeID))
		return
	}
	reqBody.WorkDate = workDate
	reqBody.Status = "Present"
	// Update DB
	err = a.DB.UpdateTodayAttendance(r.Context(), branchID, reqBody)
	if err != nil {
		a.errorLog.Println("ERROR_06_UpdateTodayAttendance DB:", err)
		utils.BadRequest(w, err)
		return
	}

	// Respond success
	resp := struct {
		Error           bool   `json:"error"`
		Status          string `json:"status"`
		Message         string `json:"message"`
		OvertimeHours   int64  `json:"overtime_hours"`
		ProductionUnits int64  `json:"production_units"`
		AdvancePayment  int64  `json:"advance_payment"`
	}{
		Error:           false,
		Status:          "success",
		Message:         "Attendance updated successfully",
		OvertimeHours:   reqBody.OvertimeHours,
		ProductionUnits: reqBody.ProductionUnits,
		AdvancePayment:  reqBody.AdvancePayment,
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}

// MarkEmployeesPresentBatch marks today's attendance for multiple employees as present.
func (a *AttendanceHandler) MarkEmployeesPresentBatch(w http.ResponseWriter, r *http.Request) {
	// Read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		a.errorLog.Println("ERROR_01_MarkEmployeesPresentBatch: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}

	var req []*models.Attendance

	// Parse JSON request
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

	// Validate and normalize each record
	for _, att := range req {
		if att.EmployeeID == 0 {
			a.errorLog.Println("ERROR_03_BatchUpdateTodayAttendance: Missing employee ID")
			utils.BadRequest(w, fmt.Errorf("missing employee ID for one of the records"))
			return
		}
		// Parse work_date from payload
		workDate, err := time.Parse("2006-01-02", att.WorkDateStr)
		if err != nil {
			a.errorLog.Printf("ERROR_XX: Invalid work_date %q for employee %d", att.WorkDateStr, att.EmployeeID)
			utils.BadRequest(w, fmt.Errorf("invalid work_date format for employee %d, expected YYYY-MM-DD", att.EmployeeID))
			return
		}

		att.WorkDate = workDate

		att.Status = "Present"
	}
	// Save all records
	err = a.DB.BatchUpdateTodayAttendance(r.Context(), branchID, req)
	if err != nil {
		a.errorLog.Println("ERROR_07_BatchUpdateTodayAttendance DB:", err)
		utils.BadRequest(w, err)
		return
	}

	// Respond success
	resp := struct {
		Error   bool   `json:"error"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}{
		Error:   false,
		Status:  "success",
		Message: "Batch attendance updated successfully",
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}

// GetEmployeeCalendar fetches calendar-style attendance for an employee (month or date range)
func (a *AttendanceHandler) GetEmployeeCalendar(w http.ResponseWriter, r *http.Request) {
	employeeID := r.URL.Query().Get("employee_id")
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
	// Read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		a.errorLog.Println("ERROR_01_GetEmployeeSummary: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}

	employeeID := r.URL.Query().Get("employee_id")
	month := r.URL.Query().Get("month")

	if employeeID == "" {
		a.errorLog.Println("ERROR_01_GetEmployeeSummary: Missing employee ID")
		utils.BadRequest(w, errors.New("missing employee ID"))
		return
	}

	summary, err := a.DB.GetEmployeeSummary(r.Context(), employeeID, branchID, month)
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
	// Read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		a.errorLog.Println("ERROR_01_GetEmployeeProgressReport: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}

	month := r.URL.Query().Get("month")
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")

	summaries, err := a.DB.GetBatchSummary(r.Context(), month, start, end, branchID)
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
