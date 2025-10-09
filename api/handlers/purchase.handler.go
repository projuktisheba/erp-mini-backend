package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/projuktisheba/erp-mini-api/internal/dbrepo"
	"github.com/projuktisheba/erp-mini-api/internal/models"
	"github.com/projuktisheba/erp-mini-api/internal/utils"
)

type PurchaseHandler struct {
	DB       *dbrepo.PurchaseRepo
	infoLog  *log.Logger
	errorLog *log.Logger
}

func NewPurchaseHandler(db *dbrepo.PurchaseRepo, infoLog, errorLog *log.Logger) *PurchaseHandler {
	return &PurchaseHandler{
		DB:       db,
		infoLog:  infoLog,
		errorLog: errorLog,
	}
}

// =========================
// AddPurchase
// =========================
func (h *PurchaseHandler) AddPurchase(w http.ResponseWriter, r *http.Request) {
	var purchase models.Purchase
	err := utils.ReadJSON(w, r, &purchase)
	if err != nil {
		h.errorLog.Println("ERROR_01_AddPurchase:", err)
		utils.BadRequest(w, err)
		return
	}
	//read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		h.errorLog.Println("ERROR_02_AddPurchase: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}
	purchase.BranchID = branchID

	h.infoLog.Println(purchase)
	// Create the purchase
	err = h.DB.CreatePurchase(r.Context(), &purchase)
	if err != nil {
		h.errorLog.Println("ERROR_02_AddPurchase:", err)
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error    bool             `json:"error"`
		Status   string           `json:"status"`
		Message  string           `json:"message"`
		Purchase *models.Purchase `json:"purchase"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Purchase created successfully"
	resp.Purchase = &purchase

	utils.WriteJSON(w, http.StatusCreated, resp)
}

// =========================
// ListPurchase
// =========================
func (h *PurchaseHandler) ListPurchases(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Query parameters
	memoNo := r.URL.Query().Get("memo_no")
	supplierIDStr := r.URL.Query().Get("supplier_id")
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		h.errorLog.Println("ERROR_01_ListPurchases: Branch ID not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}

	// Parse supplierID
	var supplierID int64
	if supplierIDStr != "" {
		if id, err := strconv.ParseInt(supplierIDStr, 10, 64); err == nil {
			supplierID = id
		} else {
			utils.BadRequest(w, errors.New("invalid supplier_id"))
			return
		}
	}

	// Pagination parameters
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	page := 1
	limit := 20
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Optional date filters
	var fromDate, toDate *time.Time
	if f := r.URL.Query().Get("from_date"); f != "" {
		if t, err := time.Parse("2006-01-02", f); err == nil {
			fromDate = &t
		} else {
			utils.BadRequest(w, errors.New("invalid from_date format, expected YYYY-MM-DD"))
			return
		}
	}
	if t := r.URL.Query().Get("to_date"); t != "" {
		if tt, err := time.Parse("2006-01-02", t); err == nil {
			toDate = &tt
		} else {
			utils.BadRequest(w, errors.New("invalid to_date format, expected YYYY-MM-DD"))
			return
		}
	}

	// Fetch purchases with pagination
	purchases, total, err := h.DB.ListPurchasesPaginated(ctx, memoNo, supplierID, branchID, fromDate, toDate, page, limit)
	if err != nil {
		h.errorLog.Println("ERROR_01_ListPurchases:", err)
		utils.ServerError(w, err)
		return
	}

	// Build response
	var resp struct {
		Error     bool               `json:"error"`
		Status    string             `json:"status"`
		Message   string             `json:"message"`
		Total     int                `json:"total"`
		Page      int                `json:"page"`
		Limit     int                `json:"limit"`
		Purchases []*models.Purchase `json:"purchases"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Purchases retrieved successfully"
	resp.Total = total
	resp.Page = page
	resp.Limit = limit
	resp.Purchases = purchases

	utils.WriteJSON(w, http.StatusOK, resp)
}
