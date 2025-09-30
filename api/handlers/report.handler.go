package api

import (
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
		rp.errorLog.Println("ERROR_01_GetOrderOverView: Report type must be valid, allow-types:[daily, weekly, monthly, yearly, all]")
		resp.Error = true
		resp.Message = "Report type must be valid, allow-types:[daily, weekly, monthly, yearly, all]"
		utils.WriteJSON(w, http.StatusBadRequest, resp)
		return
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
	summary, err := rp.DB.GetOrderOverView(r.Context(), summaryType, refDate)
	if err != nil {
		rp.errorLog.Println("ERROR_04_GetOrderOverView: ", err)
		utils.BadRequest(w, err)
		return
	}

	resp.Error = true
	resp.Message = "Success"
	resp.OrderOverview = summary
	utils.WriteJSON(w, http.StatusBadRequest, resp)
}
