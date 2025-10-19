package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/projuktisheba/erp-mini-api/internal/dbrepo"
	"github.com/projuktisheba/erp-mini-api/internal/utils"
)

type TransactionHandler struct {
	DB       *dbrepo.TransactionRepo
	infoLog  *log.Logger
	errorLog *log.Logger
}

func NewTransactionHandler(db *dbrepo.TransactionRepo, infoLog *log.Logger, errorLog *log.Logger) *TransactionHandler {
	return &TransactionHandler{
		DB:       db,
		infoLog:  infoLog,
		errorLog: errorLog,
	}
}
func (t *TransactionHandler) GetTransactionSummaryHandler(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	if startDate == "" || endDate == "" {
		utils.BadRequest(w, fmt.Errorf("start_date and end_date are required"))
		return
	}
	//read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		t.errorLog.Println("ERROR_01_GetTransactionSummaryHandler: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}

	var  trxType *string
	
	if val := r.URL.Query().Get("transaction_type"); val != "" {
		trxType = &val
	}

	transactions, err := t.DB.GetTransactionSummary(r.Context(), branchID, startDate, endDate, trxType)
	if err != nil {
		t.errorLog.Println("ERROR_01_GetTransactionSummaryHandler: ", err)
		utils.BadRequest(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"error":        false,
		"status":       "success",
		"transactions": transactions,
	})
}

func (t *TransactionHandler) ListTransactionsPaginatedHandler(w http.ResponseWriter, r *http.Request) {
	pageNo, _ := strconv.Atoi(r.URL.Query().Get("pageNo"))
	pageLength, _ := strconv.Atoi(r.URL.Query().Get("pageLength"))
	memo := r.URL.Query().Get("memo")
	var fromID, toID *int64
	if val := r.URL.Query().Get("from_id"); val != "" {
		id, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			utils.BadRequest(w, fmt.Errorf("invalid from_id"))
			return
		}
		fromID = &id
	}
	if val := r.URL.Query().Get("to_id"); val != "" {
		id, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			utils.BadRequest(w, fmt.Errorf("invalid to_id"))
			return
		}
		toID = &id
	}

	var fromType, toType, trxType *string
	if val := r.URL.Query().Get("from_type"); val != "" {
		fromType = &val
	}
	if val := r.URL.Query().Get("to_type"); val != "" {
		toType = &val
	}
	if val := r.URL.Query().Get("transaction_type"); val != "" {
		trxType = &val
	}
	//read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		t.errorLog.Println("ERROR_01_ListTransactionsPaginatedHandler: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}
	transactions, err := t.DB.ListTransactionsPaginated(r.Context(), memo, branchID, pageNo, pageLength, fromID, toID, fromType, toType, trxType)
	if err != nil {
		t.errorLog.Println("ERROR_02_ListTransactionsPaginatedHandler: ", err)
		utils.BadRequest(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"error":        false,
		"status":       "success",
		"transactions": transactions,
	})
}
