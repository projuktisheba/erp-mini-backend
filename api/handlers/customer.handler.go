package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/projuktisheba/erp-mini-api/internal/dbrepo"
	"github.com/projuktisheba/erp-mini-api/internal/models"
	"github.com/projuktisheba/erp-mini-api/internal/utils"
)

type CustomerHandler struct {
	DB       *dbrepo.CustomerRepo
	infoLog  *log.Logger
	errorLog *log.Logger
}

func NewCustomerHandler(db *dbrepo.CustomerRepo, infoLog *log.Logger, errorLog *log.Logger) *CustomerHandler {
	return &CustomerHandler{
		DB:       db,
		infoLog:  infoLog,
		errorLog: errorLog,
	}
}

// -------------------- Add New Customer --------------------
func (c *CustomerHandler) AddCustomer(w http.ResponseWriter, r *http.Request) {
	var customer models.Customer
	if err := utils.ReadJSON(w, r, &customer); err != nil {
		c.errorLog.Println("ERROR_01_AddCustomer:", err)
		utils.BadRequest(w, err)
		return
	}

	if err := c.DB.CreateNewCustomer(r.Context(), &customer); err != nil {
		c.errorLog.Println("ERROR_02_AddCustomer:", err)
		utils.BadRequest(w, err)
		return
	}

	resp := struct {
		Error    bool             `json:"error"`
		Status   string           `json:"status"`
		Message  string           `json:"message"`
		Customer *models.Customer `json:"customer"`
	}{
		Error:    false,
		Message:  "Customer added successfully",
		Customer: &customer,
	}

	utils.WriteJSON(w, http.StatusCreated, resp)
}

// -------------------- Update Customer Info --------------------
func (c *CustomerHandler) UpdateCustomerInfo(w http.ResponseWriter, r *http.Request) {
	var customer models.Customer

	if err := utils.ReadJSON(w, r, &customer); err != nil {
		c.errorLog.Println("ERROR_01_UpdateCustomerInfo:", err)
		utils.BadRequest(w, err)
		return
	}

	updatedAt, err := c.DB.UpdateCustomerInfo(r.Context(), &customer)
	if err != nil {
		c.errorLog.Println("ERROR_02_UpdateCustomerInfo:", err)
		utils.BadRequest(w, err)
		return
	}

	resp := struct {
		Error     bool       `json:"error"`
		Status    string     `json:"status"`
		Message   string     `json:"message"`
		UpdatedAt *time.Time `json:"updated_at"`
	}{
		Error:     false,
		Message:   "Customer info updated successfully",
		UpdatedAt: updatedAt,
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}

// -------------------- Update Customer Due Amount --------------------
func (c *CustomerHandler) UpdateCustomerDueAmount(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ID        int64   `json:"id"`
		DueAmount float64 `json:"due_amount"`
	}

	if err := utils.ReadJSON(w, r, &input); err != nil {
		c.errorLog.Println("ERROR_01_UpdateCustomerDueAmount:", err)
		utils.BadRequest(w, err)
		return
	}

	if err := c.DB.UpdateCustomerDueAmount(r.Context(), input.ID, input.DueAmount); err != nil {
		c.errorLog.Println("ERROR_02_UpdateCustomerDueAmount:", err)
		utils.BadRequest(w, err)
		return
	}

	resp := struct {
		Error   bool   `json:"error"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}{
		Error:   false,
		Message: "Customer due amount updated successfully",
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}

// -------------------- Update Customer Status --------------------
func (c *CustomerHandler) UpdateCustomerStatus(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ID     int64 `json:"id"`
		Status bool  `json:"status"`
	}

	if err := utils.ReadJSON(w, r, &input); err != nil {
		c.errorLog.Println("ERROR_01_UpdateCustomerStatus:", err)
		utils.BadRequest(w, err)
		return
	}

	if err := c.DB.UpdateCustomerStatus(r.Context(), input.ID, input.Status); err != nil {
		c.errorLog.Println("ERROR_02_UpdateCustomerStatus:", err)
		utils.BadRequest(w, err)
		return
	}

	resp := struct {
		Error   bool   `json:"error"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}{
		Error:   false,
		Message: "Customer status updated successfully",
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}

// -------------------- Get Customer By ID --------------------
func (c *CustomerHandler) GetCustomerByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.BadRequest(w, errors.New("invalid id"))
		return
	}

	customer, err := c.DB.GetCustomerByID(r.Context(), id)
	if err != nil {
		c.errorLog.Println("ERROR_GetCustomerByID:", err)
		utils.BadRequest(w, err)
		return
	}

	if customer == nil {
		utils.NotFound(w, "Customer not found")
		return
	}

	resp := struct {
		Error    bool             `json:"error"`
		Status   string           `json:"status"`
		Message  string           `json:"message"`
		Customer *models.Customer `json:"customer"`
	}{
		Error:    false,
		Message:  "Customer fetched successfully",
		Customer: customer,
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}

// -------------------- Get Customer By Mobile --------------------
func (c *CustomerHandler) GetCustomerByMobile(w http.ResponseWriter, r *http.Request) {
	mobile := r.URL.Query().Get("mobile")
	if strings.TrimSpace(mobile) == "" {
		utils.BadRequest(w, errors.New("invalid mobile number"))
		return
	}

	customer, err := c.DB.GetCustomerByMobile(r.Context(), mobile)
	if err != nil {
		c.errorLog.Println("ERROR_GetCustomerByMobile:", err)
		utils.BadRequest(w, err)
		return
	}

	if customer == nil {
		utils.NotFound(w, "Customer not found")
		return
	}

	resp := struct {
		Error    bool             `json:"error"`
		Status   string           `json:"status"`
		Message  string           `json:"message"`
		Customer *models.Customer `json:"customer"`
	}{
		Error:    false,
		Message:  "Customer fetched successfully",
		Customer: customer,
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}

// -------------------- Get Customer By Tax ID --------------------
func (c *CustomerHandler) GetCustomerByTaxID(w http.ResponseWriter, r *http.Request) {
	mobile := r.URL.Query().Get("tax_id")
	if strings.TrimSpace(mobile) == "" {
		utils.BadRequest(w, errors.New("invalid tax ID"))
		return
	}

	customer, err := c.DB.GetCustomerByTaxID(r.Context(), mobile)
	if err != nil {
		c.errorLog.Println("ERROR_GetCustomerTaxID:", err)
		utils.BadRequest(w, err)
		return
	}

	if customer == nil {
		utils.NotFound(w, "Customer not found")
		return
	}

	resp := struct {
		Error    bool             `json:"error"`
		Status   string           `json:"status"`
		Message  string           `json:"message"`
		Customer *models.Customer `json:"customer"`
	}{
		Error:    false,
		Message:  "Customer fetched successfully",
		Customer: customer,
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}

// FilterCustomersByName handles filtering customers by name (query param)
func (h *CustomerHandler) FilterCustomersByName(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		utils.BadRequest(w, fmt.Errorf("missing required query param: name"))
		return
	}

	customers, err := h.DB.FilterCustomersByName(r.Context(), name)
	if err != nil {
		h.errorLog.Println("ERROR_01_FilterCustomersByName:", err)
		utils.ServerError(w, err)
		return
	}

	if len(customers) == 0 {
		utils.NotFound(w, "No customers found with given name")
		return
	}

	var resp struct {
		Error     bool               `json:"error"`
		Status    string             `json:"status"`
		Message   string             `json:"message"`
		Customers []*models.Customer `json:"customers"`
	}

	resp.Error = false
	resp.Status = "success"
	resp.Message = "Customers filtered successfully"
	resp.Customers = customers

	utils.WriteJSON(w, http.StatusOK, resp)
}

// GetCustomers returns a paginated list of customers with optional filters.
func (c *CustomerHandler) GetCustomers(w http.ResponseWriter, r *http.Request) {
	// Extract query params
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	// statusStr := r.URL.Query().Get("status")

	page := 1
	limit := 20
	var err error

	if pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			utils.BadRequest(w, errors.New("invalid page number"))
			return
		}
	}

	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			utils.BadRequest(w, errors.New("invalid limit"))
			return
		}
	}

	// If status is given, parse to bool
	// var statusFilter *bool
	// if statusStr != "" {
	// 	if statusStr == "active" {
	// 		tmp := true
	// 		statusFilter = &tmp
	// 	} else if statusStr == "inactive" {
	// 		tmp := false
	// 		statusFilter = &tmp
	// 	} else {
	// 		utils.BadRequest(w, errors.New("status must be 'active' or 'inactive'"))
	// 		return
	// 	}
	// }

	// Call DB repo
	customers, err := c.DB.GetCustomers(r.Context(), page, limit /* optionally add statusFilter if repo supports */)
	if err != nil {
		c.errorLog.Println("ERROR_01_GetCustomers:", err)
		utils.ServerError(w, err)
		return
	}

	// If no customers found
	if len(customers) == 0 {
		utils.NotFound(w, "No customers found")
		return
	}

	// Prepare response
	var resp struct {
		Error     bool               `json:"error"`
		Status    string             `json:"status"`
		Message   string             `json:"message"`
		Customers []*models.Customer `json:"customers"`
	}

	resp.Error = false
	resp.Status = "success"
	resp.Message = "Customer list fetched successfully"
	resp.Customers = customers

	utils.WriteJSON(w, http.StatusOK, resp)
}

func (h *CustomerHandler) GetCustomersNameAndID(w http.ResponseWriter, r *http.Request) {
	customers, err := h.DB.GetCustomersNameAndID(r.Context())
	if err != nil {
		h.errorLog.Println("ERROR_01_GetCustomersNameAndID:", err)
		utils.ServerError(w, err)
		return
	}

	if len(customers) == 0 {
		utils.NotFound(w, "No customers found")
		return
	}

	var resp struct {
		Error     bool                     `json:"error"`
		Status    string                   `json:"status"`
		Message   string                   `json:"message"`
		Customers []*models.CustomerNameID `json:"customers"`
	}

	resp.Error = false
	resp.Status = "success"
	resp.Message = "Customers fetched successfully"
	resp.Customers = customers

	utils.WriteJSON(w, http.StatusOK, resp)
}

func (c *CustomerHandler) GetCustomersWithDueHandler(w http.ResponseWriter, r *http.Request) {
	customers, err := c.DB.GetCustomersWithDue(r.Context())
	if err != nil {
		c.errorLog.Println("ERROR_01_GetCustomersWithDueHandler:", err)
		utils.BadRequest(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"error":     false,
		"status":    "success",
		"customers": customers,
	})
}
