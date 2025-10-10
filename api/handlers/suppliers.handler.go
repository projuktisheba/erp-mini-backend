package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/projuktisheba/erp-mini-api/internal/dbrepo"
	"github.com/projuktisheba/erp-mini-api/internal/models"
	"github.com/projuktisheba/erp-mini-api/internal/utils"
)

type SupplierHandler struct {
	DB       *dbrepo.SupplierRepo
	infoLog  *log.Logger
	errorLog *log.Logger
}

func NewSupplierHandler(db *dbrepo.SupplierRepo, infoLog *log.Logger, errorLog *log.Logger) *SupplierHandler {
	return &SupplierHandler{
		DB:       db,
		infoLog:  infoLog,
		errorLog: errorLog,
	}
}

// =========================
// Add Supplier
// =========================
func (h *SupplierHandler) AddSupplier(w http.ResponseWriter, r *http.Request) {
	var supplier models.Supplier
	err := utils.ReadJSON(w, r, &supplier)
	if err != nil {
		h.errorLog.Println("ERROR_01_AddSupplier:", err)
		utils.BadRequest(w, err)
		return
	}
	//read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		h.errorLog.Println("ERROR_01_AddCustomer: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}
	supplier.BranchID = branchID
	supplier.Status = "active"
	err = h.DB.CreateSupplier(r.Context(), &supplier)
	if err != nil {
		h.errorLog.Println("ERROR_02_AddSupplier:", err)
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error    bool             `json:"error"`
		Status   string           `json:"status"`
		Message  string           `json:"message"`
		Supplier *models.Supplier `json:"supplier"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Supplier added successfully"
	resp.Supplier = &supplier

	utils.WriteJSON(w, http.StatusCreated, resp)
}

// =========================
// Update Supplier
// =========================
func (h *SupplierHandler) UpdateSupplier(w http.ResponseWriter, r *http.Request) {
	var supplier models.Supplier
	err := utils.ReadJSON(w, r, &supplier)
	if err != nil {
		h.errorLog.Println("ERROR_01_UpdateSupplier:", err)
		utils.BadRequest(w, err)
		return
	}

	err = h.DB.UpdateSupplier(r.Context(), &supplier)
	if err != nil {
		h.errorLog.Println("ERROR_02_UpdateSupplier:", err)
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error    bool             `json:"error"`
		Status   string           `json:"status"`
		Message  string           `json:"message"`
		Supplier *models.Supplier `json:"supplier"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Supplier updated successfully"
	resp.Supplier = &supplier

	utils.WriteJSON(w, http.StatusOK, resp)
}

// =========================
// Get Supplier By ID
// =========================
func (h *SupplierHandler) GetSupplierByID(w http.ResponseWriter, r *http.Request) {
	supplierIDStr := r.URL.Query().Get("id")
	supplierID, err := strconv.ParseInt(supplierIDStr, 10, 64)
	if err != nil {
		h.errorLog.Println("ERROR_01_GetSupplierByID: invalid ID", err)
		utils.BadRequest(w, err)
		return
	}

	supplier, err := h.DB.GetSupplierByID(r.Context(), supplierID)
	if err != nil {
		h.errorLog.Println("ERROR_02_GetSupplierByID:", err)
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error    bool             `json:"error"`
		Status   string           `json:"status"`
		Message  string           `json:"message"`
		Supplier *models.Supplier `json:"supplier"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Supplier retrieved successfully"
	resp.Supplier = supplier

	utils.WriteJSON(w, http.StatusOK, resp)
}

// =========================
// Search Suppliers with Pagination
// =========================
func (h *SupplierHandler) ListSuppliers(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	status := r.URL.Query().Get("status")
	mobile := r.URL.Query().Get("mobile")
	//read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		h.errorLog.Println("ERROR_01_ListSuppliers: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}
	// Pagination params
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 0
	limit := 0

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

	suppliers, total, err := h.DB.ListSuppliers(r.Context(), name, status, mobile, page, limit, branchID)
	if err != nil {
		h.errorLog.Println("ERROR_01_SearchSuppliersPaginated:", err)
		utils.ServerError(w, err)
		return
	}

	var resp struct {
		Error     bool               `json:"error"`
		Status    string             `json:"status"`
		Message   string             `json:"message"`
		Total     int                `json:"total"`
		Page      int                `json:"page"`
		Limit     int                `json:"limit"`
		Suppliers []*models.Supplier `json:"suppliers"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Suppliers retrieved successfully"
	resp.Total = total
	resp.Page = page
	resp.Limit = limit
	resp.Suppliers = suppliers

	utils.WriteJSON(w, http.StatusOK, resp)
}
