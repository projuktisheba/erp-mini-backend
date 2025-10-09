package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/projuktisheba/erp-mini-api/internal/dbrepo"
	"github.com/projuktisheba/erp-mini-api/internal/models"
	"github.com/projuktisheba/erp-mini-api/internal/utils"
)

type ReportHandler struct {
	DB       *dbrepo.ReportRepo
	infoLog  *log.Logger
	errorLog *log.Logger
}

func NewReportHandler(db *dbrepo.ReportRepo, infoLog *log.Logger, errorLog *log.Logger) *ReportHandler {
	return &ReportHandler{
		DB:       db,
		infoLog:  infoLog,
		errorLog: errorLog,
	}
}
func (rp *ReportHandler) GetOrderOverView(w http.ResponseWriter, r *http.Request) {
	summaryType := strings.TrimSpace(r.URL.Query().Get("type"))
	refDateStr := strings.TrimSpace(r.URL.Query().Get("date"))

	// Read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		rp.errorLog.Println("ERROR_01_GetOrderOverView: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}

	acceptableTypes := map[string]bool{
		"daily":   true,
		"weekly":  true,
		"monthly": true,
		"yearly":  true,
		"all":     true,
	}
	var resp struct {
		Error         bool                  `json:"error"`
		Message       string                `json:"message"`
		OrderOverview *models.OrderOverview `json:"order_overview"`
	}
	if _, isAcceptable := acceptableTypes[summaryType]; !isAcceptable {
		summaryType = "monthly"
	}

	refDate, err := time.Parse("2006-01-02", refDateStr)
	if err != nil {
		rp.errorLog.Println("ERROR_03_GetOrderOverView: Invalid reference date")
		resp.Error = true
		resp.Message = "Please enter a valid date"
		utils.WriteJSON(w, http.StatusBadRequest, resp)
		return
	}
	rp.infoLog.Println(refDate)
	summary, err := rp.DB.GetOrderOverView(r.Context(), branchID, summaryType, refDate)
	if err != nil {
		rp.errorLog.Println("ERROR_04_GetOrderOverView: ", err)
		utils.BadRequest(w, err)
		return
	}

	resp.Error = true
	resp.Message = "Success"
	resp.OrderOverview = summary
	utils.WriteJSON(w, http.StatusOK, resp)
}

func (rp *ReportHandler) GetEmployeeProgressReport(w http.ResponseWriter, r *http.Request) {
	// Read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		rp.errorLog.Println("ERROR_01_GetEmployeeProgressReport: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}

	startDateStr := strings.TrimSpace(r.URL.Query().Get("start_date"))
	endDateStr := strings.TrimSpace(r.URL.Query().Get("end_date"))
	reportType := strings.TrimSpace(r.URL.Query().Get("report_type"))

	if reportType == "" {
		reportType = "monthly" // default report type
	}

	// Default: current month range
	var startDate, endDate time.Time
	var err error
	if startDateStr == "" || endDateStr == "" {
		now := time.Now()
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		// last day of current month
		endDate = startDate.AddDate(0, 1, 0).Add(-time.Nanosecond)
	} else {
		// Parse provided date range
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			utils.BadRequest(w, fmt.Errorf("Invalid start_date format, expected YYYY-MM-DD"))
			return
		}
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			utils.BadRequest(w, fmt.Errorf("Invalid end_date format, expected YYYY-MM-DD"))
			return
		}
		// Normalize end date to include full day
		endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	// Fetch report from repo
	empReport, err := rp.DB.GetSalesPersonProgressReport(r.Context(), branchID, startDate, endDate, reportType)
	if err != nil {
		rp.errorLog.Println("ERROR_03_GetEmployeeProgressReport: ", err)
		utils.BadRequest(w, err)
		return
	}

	// Prepare response
	resp := struct {
		Error                  bool                                `json:"error"`
		Message                string                              `json:"message"`
		EmployeeProgressReport []*models.SalesPersonProgressReport `json:"report"`
	}{
		Error:                  false,
		Message:                "Progress report created successfully",
		EmployeeProgressReport: empReport,
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}

func (rp *ReportHandler) GetWorkerProgressReport(w http.ResponseWriter, r *http.Request) {
	// Read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		rp.errorLog.Println("ERROR_01_GetWorkerProgressReport: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}
	startDateStr := strings.TrimSpace(r.URL.Query().Get("start_date"))
	endDateStr := strings.TrimSpace(r.URL.Query().Get("end_date"))
	reportType := strings.TrimSpace(r.URL.Query().Get("report_type"))

	if reportType == "" {
		reportType = "daily" // default report type
	}

	// Default: current month range
	var startDate, endDate time.Time
	var err error
	if startDateStr == "" || endDateStr == "" {
		now := time.Now()
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		// last day of current month
		endDate = startDate.AddDate(0, 1, 0).Add(-time.Nanosecond)
	} else {
		// Parse provided date range
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			utils.BadRequest(w, fmt.Errorf("Invalid start_date format, expected YYYY-MM-DD"))
			return
		}
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			utils.BadRequest(w, fmt.Errorf("Invalid end_date format, expected YYYY-MM-DD"))
			return
		}
		// Normalize end date to include full day
		endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	// Fetch report from repo
	empReport, err := rp.DB.GetAllWorkersProgressReport(r.Context(), branchID, startDate, endDate, reportType)
	if err != nil {
		rp.errorLog.Println("ERROR_03_GetWorkerProgressReport: ", err)
		utils.BadRequest(w, err)
		return
	}

	// Prepare response
	resp := struct {
		Error                  bool                           `json:"error"`
		Message                string                         `json:"message"`
		EmployeeProgressReport []*models.WorkerProgressReport `json:"report"`
	}{
		Error:                  false,
		Message:                "Progress report created successfully",
		EmployeeProgressReport: empReport,
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}

func (rp *ReportHandler) GetBranchReport(w http.ResponseWriter, r *http.Request) {
	// Read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		rp.errorLog.Println("ERROR_01_GetBranchReport: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}
	startDateStr := strings.TrimSpace(r.URL.Query().Get("start_date"))
	endDateStr := strings.TrimSpace(r.URL.Query().Get("end_date"))
	reportType := strings.TrimSpace(r.URL.Query().Get("report_type"))

	if reportType == "" {
		reportType = "daily" // default report type
	}

	// Default: current month range
	var startDate, endDate time.Time
	var err error
	if startDateStr == "" || endDateStr == "" {
		now := time.Now()
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		// last day of current month
		endDate = startDate.AddDate(0, 1, 0).Add(-time.Nanosecond)
	} else {
		// Parse provided date range
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			utils.BadRequest(w, fmt.Errorf("Invalid start_date format, expected YYYY-MM-DD"))
			return
		}
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			utils.BadRequest(w, fmt.Errorf("Invalid end_date format, expected YYYY-MM-DD"))
			return
		}
		// Normalize end date to include full day
		endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	// Fetch report from repo
	branchReport, err := rp.DB.GetBranchReport(r.Context(), branchID, startDate, endDate, reportType)
	if err != nil {
		rp.errorLog.Println("ERROR_03_GetEmployeeProgressReport: ", err)
		utils.BadRequest(w, err)
		return
	}

	// Prepare response
	resp := struct {
		Error        bool               `json:"error"`
		Message      string             `json:"message"`
		BranchReport []*models.TopSheet `json:"report"`
	}{
		Error:        false,
		Message:      "Progress report created successfully",
		BranchReport: branchReport,
	}

	utils.WriteJSON(w, http.StatusOK, resp)

}
