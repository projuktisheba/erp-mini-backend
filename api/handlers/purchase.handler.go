package api

import (
	"errors"
	"fmt"
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
// UpdatePurchase
// =========================
func (h *PurchaseHandler) UpdatePurchase(w http.ResponseWriter, r *http.Request) {
	var purchase models.Purchase
	err := utils.ReadJSON(w, r, &purchase)
	if err != nil {
		h.errorLog.Println("ERROR_01_UpdatePurchase:", err)
		utils.BadRequest(w, err)
		return
	}
	//read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		h.errorLog.Println("ERROR_02_UpdatePurchase: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}
	purchase.BranchID = branchID

	// Print each field
	fmt.Println("ID:", purchase.ID)
	fmt.Println("MemoNo:", purchase.MemoNo)
	fmt.Println("PurchaseDate:", purchase.PurchaseDate)
	fmt.Println("SupplierID:", purchase.SupplierID)
	fmt.Println("SupplierName:", purchase.SupplierName)
	fmt.Println("BranchID:", purchase.BranchID)
	fmt.Println("TotalAmount:", purchase.TotalAmount)
	fmt.Println("Notes:", purchase.Notes)
	fmt.Println("CreatedAt:", purchase.CreatedAt)
	// Update the purchase
	err = h.DB.UpdatePurchase(r.Context(), &purchase)
	if err != nil {
		h.errorLog.Println("ERROR_03_UpdatePurchase:", err)
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
	resp.Message = "Purchase updated successfully"
	resp.Purchase = &purchase

	utils.WriteJSON(w, http.StatusCreated, resp)
}

// ListPurchases handles GET /purchases
func (h *PurchaseHandler) ListPurchases(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// -------------------------
	// Get branch ID from header
	// -------------------------
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		h.errorLog.Println("ERROR_ListPurchases: Branch ID not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}

	// -------------------------
	// Query parameters
	// -------------------------
	memoNo := r.URL.Query().Get("memo_no")
	supplierID := int64(0)
	if sID := r.URL.Query().Get("supplier_id"); sID != "" {
		id, err := strconv.ParseInt(sID, 10, 64)
		if err != nil {
			utils.BadRequest(w, errors.New("invalid supplier_id"))
			return
		}
		supplierID = id
	}

	// -------------------------
	// Pagination
	// -------------------------
	page := 0
	limit := 0
	if p := r.URL.Query().Get("page"); p != "" {
		if pi, err := strconv.Atoi(p); err == nil && pi > 0 {
			page = pi
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if li, err := strconv.Atoi(l); err == nil && li > 0 {
			limit = li
		}
	}

	// -------------------------
	// Optional date filters
	// -------------------------
	var fromDate, toDate *time.Time
	if f := r.URL.Query().Get("start_date"); f != "" {
		t, err := time.Parse("2006-01-02", f)
		if err != nil {
			utils.BadRequest(w, errors.New("invalid start_date format, expected YYYY-MM-DD"))
			return
		}
		fromDate = &t
	}
	if t := r.URL.Query().Get("end_date"); t != "" {
		tt, err := time.Parse("2006-01-02", t)
		if err != nil {
			utils.BadRequest(w, errors.New("invalid end_date format, expected YYYY-MM-DD"))
			return
		}
		toDate = &tt
	}

	// -------------------------
	// Fetch purchases from DB
	// -------------------------
	purchases, total, err := h.DB.ListPurchasesPaginated(ctx, memoNo, supplierID, branchID, fromDate, toDate, page, limit)
	if err != nil {
		h.errorLog.Println("ERROR_ListPurchases: failed to list purchases:", err)
		utils.ServerError(w, err)
		return
	}

	// -------------------------
	// Response
	// -------------------------
	type PurchaseListResponse struct {
		Error     bool               `json:"error"`
		Status    string             `json:"status"`
		Message   string             `json:"message"`
		Total     int                `json:"total"`
		Page      int                `json:"page"`
		Limit     int                `json:"limit"`
		Purchases []*models.Purchase `json:"report"`
	}

	resp := PurchaseListResponse{
		Error:     false,
		Status:    "success",
		Message:   "Purchases retrieved successfully",
		Total:     total,
		Page:      page,
		Limit:     limit,
		Purchases: purchases,
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}
