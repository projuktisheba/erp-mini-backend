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

type ProductHandler struct {
	DB       *dbrepo.ProductRepo
	infoLog  *log.Logger
	errorLog *log.Logger
}

func NewProductHandler(db *dbrepo.ProductRepo, infoLog *log.Logger, errorLog *log.Logger) *ProductHandler {
	return &ProductHandler{
		DB:       db,
		infoLog:  infoLog,
		errorLog: errorLog,
	}
}

// GetProductsHandler fetches all products
// Example: GET /api/v1/products
func (h *ProductHandler) GetProductsHandler(w http.ResponseWriter, r *http.Request) {
	branchID := utils.GetBranchID(r)

	products, err := h.DB.GetProducts(r.Context(), branchID)
	if err != nil {
		h.errorLog.Println("ERROR_GetProductsHandler:", err)
		utils.ServerError(w, err)
		return
	}

	var resp struct {
		Error    bool              `json:"error"`
		Status   string            `json:"status"`
		Message  string            `json:"message"`
		Products []*models.Product `json:"products"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Product Names and IDs fetched successfully"
	resp.Products = products

	utils.WriteJSON(w, http.StatusOK, resp)
}

func (h *ProductHandler) RestockProducts(w http.ResponseWriter, r *http.Request) {
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		h.errorLog.Println("ERROR_01_CheckoutOrder: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}
	var requestBody struct {
		Date     time.Time        `json:"date"`
		MemoNo   string           `json:"memo_no"`
		Products []models.Product `json:"products"`
	}

	err := utils.ReadJSON(w, r, &requestBody)
	if err != nil {
		h.errorLog.Println("ERROR_01_RestockProducts: Unable to unmarshal JSON => ", err)
		utils.BadRequest(w, err)
		return
	}

	h.infoLog.Println(requestBody)

	memoNo, err := h.DB.RestockProducts(r.Context(), requestBody.Date, requestBody.MemoNo, branchID, requestBody.Products)
	if err != nil {
		h.errorLog.Println("ERROR_02_RestockProducts: Unable to update stocks => ", err)
		utils.BadRequest(w, err)
		return
	}
	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
		MemoNo  string `json:"memo_no"`
	}

	resp.Error = false
	resp.Message = "Products stored successfully"
	resp.MemoNo = memoNo
	utils.WriteJSON(w, http.StatusCreated, resp)
}

// GetProductStockHandler handles GET /api/product-stock requests
func (h *ProductHandler) GetProductStockReportHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters: start_date, end_date
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	if startDateStr == "" || endDateStr == "" {
		h.errorLog.Println("ERROR_01_GetProductStockReportHandler:Missing required parameters: branch_id, start_date, end_date")
		utils.BadRequest(w, errors.New("Missing required parameters: branch_id, start_date, end_date"))
		return
	}

	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		h.errorLog.Println("ERROR_02_GetProductStockReportHandler: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		h.errorLog.Println("ERROR_03_GetProductStockReportHandler: Branch id not found")
		utils.BadRequest(w, errors.New("Invalid start_date format (expected YYYY-MM-DD)"))
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		h.errorLog.Println("ERROR_04_GetProductStockReportHandler: Branch id not found")
		utils.BadRequest(w, errors.New("Invalid end_date format (expected YYYY-MM-DD)"))
		return
	}

	// Fetch data from database
	records, err := h.DB.GetProductStockReportByDateRange(r.Context(), branchID, startDate, endDate)
	if err != nil {
		h.errorLog.Println("ERROR_05_GetProductStockReportHandler: Branch id not found")
		utils.BadRequest(w, fmt.Errorf("Database error: %w", err))
		return
	}

	var resp struct {
		Error   bool                           `json:"error"`
		Message string                         `json:"message"`
		Report  []*models.ProductStockRegistry `json:"report"`
	}

	resp.Error = false
	resp.Message = "Report fetched successfully"
	resp.Report = records

	utils.WriteJSON(w, http.StatusOK, resp)
}

func (h *ProductHandler) SaleProducts(w http.ResponseWriter, r *http.Request) {
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		h.errorLog.Println("ERROR_01_SaleProducts: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}

	var requestBody models.Sale

	err := utils.ReadJSON(w, r, &requestBody)
	if err != nil {
		h.errorLog.Println("ERROR_02_SaleProducts: Unable to unmarshal JSON =>", err)
		utils.BadRequest(w, err)
		return
	}

	h.infoLog.Println(requestBody)

	memoNo, err := h.DB.SaleProducts(r.Context(), branchID, &requestBody)
	if err != nil {

		h.errorLog.Println("ERROR_03_SaleProducts: Unable to process sale =>", err)
		if strings.Contains(err.Error(), `duplicate key value violates unique constraint "sales_history_memo_no_branch_id_key"`) {
			err = errors.New("Duplicate memo number is not allowed")
		}
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
		MemoNo  string `json:"memo_no"`
	}

	resp.Error = false
	resp.Message = "Products sold successfully"
	resp.MemoNo = memoNo

	utils.WriteJSON(w, http.StatusCreated, resp)
}
func (h *ProductHandler) UpdateSoldProducts(w http.ResponseWriter, r *http.Request) {
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		h.errorLog.Println("ERROR_01_UpdateSoldProducts: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}
	memoNo := strings.TrimSpace(r.URL.Query().Get("memo_no"))
	if memoNo == "" {
		h.errorLog.Println("ERROR_02_UpdateSoldProducts: Memo not found in the payload")
		utils.BadRequest(w, errors.New("Memo not found in the payload"))
		return
	}
	var requestBody models.Sale
	err := utils.ReadJSON(w, r, &requestBody)
	if err != nil {
		h.errorLog.Println("ERROR_03_UpdateSoldProducts: Unable to unmarshal JSON =>", err)
		utils.BadRequest(w, err)
		return
	}
	requestBody.MemoNo = memoNo
	h.infoLog.Println(requestBody)
	for _, v := range requestBody.Items {
		fmt.Println(v.ID)
	}

	err = h.DB.UpdateSoldProducts(r.Context(), branchID, requestBody)
	if err != nil {
		h.errorLog.Println("ERROR_03_SaleProducts: Unable to process sale =>", err)
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}

	resp.Error = false
	resp.Message = "Sold products updated successfully"

	utils.WriteJSON(w, http.StatusCreated, resp)
}

// GetSaleDetails handles fetch the sales details by id
func (h *ProductHandler) GetSaleDetails(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters: start_date, end_date
	memoNo := r.URL.Query().Get("memo_no")

	if memoNo == "" {
		h.errorLog.Println("ERROR_01_GetSaleDetails:Missing required parameters: memo_no")
		utils.BadRequest(w, errors.New("Missing required parameters:memo_no"))
		return
	}

	// Fetch data from database
	soldItems, err := h.DB.GetSoldItemsByMemoNo(r.Context(), memoNo)
	if err != nil {
		h.errorLog.Println("ERROR_02_GetSaleDetails: Unable to fetch details")
		utils.BadRequest(w, fmt.Errorf("Database error: %w", err))
		return
	}

	var resp struct {
		Error        bool              `json:"error"`
		Message      string            `json:"message"`
		ProductItems []*models.Product `json:"sold_items"`
	}

	resp.Error = false
	resp.Message = "Sale details fetched successfully"
	resp.ProductItems = soldItems

	utils.WriteJSON(w, http.StatusOK, resp)
}

// GetSaleReport handles fetch the sales report
func (h *ProductHandler) GetSaleReport(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters: start_date, end_date
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	if startDateStr == "" || endDateStr == "" {
		h.errorLog.Println("ERROR_01_GetSaleReport:Missing required parameters: branch_id, start_date, end_date")
		utils.BadRequest(w, errors.New("Missing required parameters:start_date, end_date"))
		return
	}

	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		h.errorLog.Println("ERROR_02_GetSaleReport: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		h.errorLog.Println("ERROR_03_GetSaleReport: Invalid start_date format (expected YYYY-MM-DD)")
		utils.BadRequest(w, errors.New("Invalid start_date format (expected YYYY-MM-DD)"))
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		h.errorLog.Println("ERROR_04_GetSaleReport: Invalid end_date format (expected YYYY-MM-DD)")
		utils.BadRequest(w, errors.New("Invalid end_date format (expected YYYY-MM-DD)"))
		return
	}

	// Fetch data from database
	records, err := h.DB.GetAllSales(r.Context(), branchID, startDate, endDate)
	if err != nil {
		h.errorLog.Println("ERROR_05_GetSaleReport: Unable to fetch report")
		utils.BadRequest(w, fmt.Errorf("Database error: %w", err))
		return
	}

	var resp struct {
		Error   bool           `json:"error"`
		Message string         `json:"message"`
		Report  []*models.Sale `json:"report"`
	}

	resp.Error = false
	resp.Message = "Report fetched successfully"
	resp.Report = records

	utils.WriteJSON(w, http.StatusOK, resp)
}
