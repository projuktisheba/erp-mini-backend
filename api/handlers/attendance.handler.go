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
	var req models.Attendance

	// Parse JSON request
	err := utils.ReadJSON(w, r, &req)
	if err != nil {
		a.errorLog.Println("ERROR_01_UpdateTodayAttendance:", err)
		utils.BadRequest(w, err)
		return
	}

	// Check required field
	if req.EmployeeID == 0 {
		a.errorLog.Println("ERROR_02_UpdateTodayAttendance: Missing employee ID")
		utils.BadRequest(w, errors.New("missing employee ID"))
		return
	}

	workDate, err := time.Parse("2006-01-02", req.WorkDateStr)
	if err != nil {
		a.errorLog.Printf("ERROR_XX: Invalid work_date %q for employee %d", req.WorkDate, req.EmployeeID)
		utils.BadRequest(w, fmt.Errorf("invalid work_date format for employee %d, expected YYYY-MM-DD", req.EmployeeID))
		return
	}

	// Parse check-in time (24-hour)
	checkInTime, err := time.Parse("15:04", req.CheckInStr)
	if err != nil {
		a.errorLog.Println("ERROR_03_UpdateTodayAttendance: Invalid CheckInStr:", req.CheckInStr)
		utils.BadRequest(w, errors.New("invalid check-in time format, expected HH:MM (24h)"))
		return
	}
	checkInTime = time.Date(workDate.Year(), workDate.Month(), workDate.Day(),
		checkInTime.Hour(), checkInTime.Minute(), 0, 0, time.UTC)

	// Parse check-out time (24-hour)
	checkOutTime, err := time.Parse("15:04", req.CheckOutStr)
	if err != nil {
		a.errorLog.Println("ERROR_04_UpdateTodayAttendance: Invalid CheckOutStr:", req.CheckOutStr)
		utils.BadRequest(w, errors.New("invalid check-out time format, expected HH:MM (24h)"))
		return
	}
	checkOutTime = time.Date(workDate.Year(), workDate.Month(), workDate.Day(),
		checkOutTime.Hour(), checkOutTime.Minute(), 0, 0, time.UTC)

	// Handle overnight shift
	if checkOutTime.Before(checkInTime) {
		checkOutTime = checkOutTime.Add(24 * time.Hour)
	}

	req.CheckIn = checkInTime
	req.CheckOut = checkOutTime
	req.WorkDate = workDate
	req.Status = "Present"

	// Calculate overtime if duration > 0
	calculatedOvertime := 0
	if !checkInTime.Equal(checkOutTime) {
		duration := checkOutTime.Sub(checkInTime)
		calculatedOvertime = max(int(duration.Hours())-8, 0)

		// Validate user-provided overtime
		if req.OvertimeHours != calculatedOvertime {
			errMsg := fmt.Sprintf(
				"provided overtime_hours (%d) does not match calculated overtime (%d)",
				req.OvertimeHours, calculatedOvertime,
			)
			a.errorLog.Println("ERROR_05_UpdateTodayAttendance:", errMsg)
			utils.BadRequest(w, errors.New(errMsg))
			return
		}
	}

	// Assign validated/corrected overtime
	req.OvertimeHours = calculatedOvertime

	// Update DB
	err = a.DB.UpdateTodayAttendance(r.Context(), req)
	if err != nil {
		a.errorLog.Println("ERROR_06_UpdateTodayAttendance DB:", err)
		utils.BadRequest(w, err)
		return
	}

	// Respond success
	resp := struct {
		Error         bool   `json:"error"`
		Status        string `json:"status"`
		Message       string `json:"message"`
		OvertimeHours int    `json:"overtime_hours"`
	}{
		Error:         false,
		Status:        "success",
		Message:       "Attendance updated successfully",
		OvertimeHours: req.OvertimeHours,
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}

// MarkEmployeesPresentBatch marks today's attendance for multiple employees as present.
func (a *AttendanceHandler) MarkEmployeesPresentBatch(w http.ResponseWriter, r *http.Request) {

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

		// Parse check-in
		checkInTime, err := time.Parse("15:04", att.CheckInStr)
		if err != nil {
			a.errorLog.Printf("ERROR_04_BatchUpdateTodayAttendance: Invalid CheckInStr %q for employee %d", att.CheckInStr, att.EmployeeID)
			utils.BadRequest(w, fmt.Errorf("invalid check-in time format for employee %d, expected HH:MM (24h)", att.EmployeeID))
			return
		}
		checkInTime = time.Date(workDate.Year(), workDate.Month(), workDate.Day(),
			checkInTime.Hour(), checkInTime.Minute(), 0, 0, time.UTC)

		// Parse check-out
		checkOutTime, err := time.Parse("15:04", att.CheckOutStr)
		if err != nil {
			a.errorLog.Printf("ERROR_05_BatchUpdateTodayAttendance: Invalid CheckOutStr %q for employee %d", att.CheckOutStr, att.EmployeeID)
			utils.BadRequest(w, fmt.Errorf("invalid check-out time format for employee %d, expected HH:MM (24h)", att.EmployeeID))
			return
		}
		checkOutTime = time.Date(workDate.Year(), workDate.Month(), workDate.Day(),
			checkOutTime.Hour(), checkOutTime.Minute(), 0, 0, time.UTC)

		// Handle overnight shift
		if checkOutTime.Before(checkInTime) {
			checkOutTime = checkOutTime.Add(24 * time.Hour)
		}

		// Assign times
		att.CheckIn = checkInTime
		att.CheckOut = checkOutTime
		att.WorkDate = workDate

		att.Status = "Present"

		// Calculate overtime
		var calculatedOvertime int
		if !checkInTime.Equal(checkOutTime) {
			duration := checkOutTime.Sub(checkInTime)
			calculatedOvertime = max(int(duration.Hours())-8, 0)

			// Validate user-provided overtime
			if att.OvertimeHours != calculatedOvertime {
				errMsg := fmt.Sprintf("employee %d: provided overtime_hours (%d) does not match calculated overtime (%d)",
					att.EmployeeID, att.OvertimeHours, calculatedOvertime)
				a.errorLog.Println("ERROR_06_BatchUpdateTodayAttendance:", errMsg)
				utils.BadRequest(w, errors.New(errMsg))
				return
			}
		} else {
			// Zero duration = no overtime
			calculatedOvertime = 0
			att.OvertimeHours = 0
		}

		// Set validated overtime
		att.OvertimeHours = calculatedOvertime
	}

	// Save all records
	err = a.DB.BatchUpdateTodayAttendance(r.Context(), req)
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
	employeeID := r.URL.Query().Get("employee_id")
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
