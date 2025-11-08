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

type OrderHandler struct {
	DB       *dbrepo.OrderRepo
	infoLog  *log.Logger
	errorLog *log.Logger
}

func NewOrderHandler(db *dbrepo.OrderRepo, infoLog *log.Logger, errorLog *log.Logger) *OrderHandler {
	return &OrderHandler{
		DB:       db,
		infoLog:  infoLog,
		errorLog: errorLog,
	}
}

func (o *OrderHandler) AddOrder(w http.ResponseWriter, r *http.Request) {

	var orderDetails models.Order
	err := utils.ReadJSON(w, r, &orderDetails)
	if err != nil {
		o.errorLog.Println("ERROR_01_AddOrder", err)
		utils.BadRequest(w, err)
		return
	}
	//read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		o.errorLog.Println("ERROR_02_AddOrder: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}
	orderDetails.BranchID = branchID
	o.infoLog.Println(orderDetails)
	// Create the order
	err = o.DB.CreateOrder(r.Context(), &orderDetails)
	if err != nil {
		o.errorLog.Println("ERROR_03_AddOrder: ", err)
		if strings.Contains(err.Error(), "orders_memo_no_branch_id_key") {
			err = errors.New("Duplicate memo is not allowed. Please enter unique memo number")
		}
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error   bool          `json:"error"`
		Status  string        `json:"status"`
		Message string        `json:"message"`
		Order   *models.Order `json:"order"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Order added successfully"
	resp.Order = &orderDetails

	utils.WriteJSON(w, http.StatusCreated, resp)
}

// UpdateOrder
func (o *OrderHandler) UpdateOrder(w http.ResponseWriter, r *http.Request) {

	var orderDetails models.Order
	err := utils.ReadJSON(w, r, &orderDetails)
	if err != nil {
		o.errorLog.Println("ERROR_01_UpdateOrder", err)
		utils.BadRequest(w, err)
		return
	}

	//read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		o.errorLog.Println("ERROR_02_UpdateOrder: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}
	orderDetails.BranchID = branchID
	err = o.DB.UpdateOrder(r.Context(), &orderDetails, o.errorLog)
	if err != nil {
		o.errorLog.Println("ERROR_03_UpdateOrder: ", err)
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error   bool          `json:"error"`
		Status  string        `json:"status"`
		Message string        `json:"message"`
		Order   *models.Order `json:"order"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Order updated successfully"
	resp.Order = &orderDetails

	utils.WriteJSON(w, http.StatusOK, resp)
}

// CheckoutOrder
func (o *OrderHandler) CheckoutOrder(w http.ResponseWriter, r *http.Request) {
	// Read branch id
	// branchID := utils.GetBranchID(r)
	// o.infoLog.Println("branch id",branchID)
	// if branchID == 0 {
	// 	o.errorLog.Println("ERROR_01_CheckoutOrder: Branch id not found")
	// 	utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
	// 	return
	// }
	orderIDStr := r.URL.Query().Get("order_id")
	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		o.errorLog.Println("ERROR_02_CheckoutOrder: invalid order ID", err)
		utils.BadRequest(w, err)
		return
	}
	branchIDStr := r.URL.Query().Get("branch_id")
	branchID, err := strconv.ParseInt(branchIDStr, 10, 64)
	if err != nil {
		o.errorLog.Println("ERROR_01_CheckoutOrder: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}

	err = o.DB.CheckoutOrder(r.Context(), orderID, branchID)
	if err != nil {
		o.errorLog.Println("ERROR_03_CheckoutOrder: ", err)
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error   bool   `json:"error"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Order is ready to deliver"

	utils.WriteJSON(w, http.StatusOK, resp)
}

// orderDelivery
func (o *OrderHandler) OrderDelivery(w http.ResponseWriter, r *http.Request) {
	// Read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		o.errorLog.Println("ERROR_01_OrderDelivery: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}
	var req struct {
		OrderID          int64   `json:"order_id"`
		ExitDate         string  `json:"exit_date"`
		PaidAmount       float64 `json:"paid_amount"`
		TotalItems       int64   `json:"total_items_delivered"`
		PaymentAccountID int64   `json:"payment_account_id"`
	}
	err := utils.ReadJSON(w, r, &req)
	if err != nil {
		o.errorLog.Println("ERROR_02_OrderDelivery: invalid order ID", err)
		utils.BadRequest(w, err)
		return
	}
	exitDate, err := time.Parse("2006-01-02", req.ExitDate)
	if err != nil {
		o.errorLog.Println("ERROR_02_OrderDelivery: invalid exit date", err)
		utils.BadRequest(w, err)
		return
	}
	fmt.Printf("Request data: %+v\n", req)
	fmt.Printf("Exit date: %+v\n", exitDate)

	err = o.DB.ConfirmDelivery(r.Context(), branchID, req.OrderID, req.TotalItems, req.PaidAmount, req.PaymentAccountID, exitDate)
	if err != nil {
		o.errorLog.Println("ERROR_03_OrderDelivery: ", err)
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error   bool   `json:"error"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Products successfully handed over to the customer."

	utils.WriteJSON(w, http.StatusOK, resp)
}

// CancelOrder
func (o *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	// Read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		o.errorLog.Println("ERROR_01_CancelOrder: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}

	orderIDStr := r.URL.Query().Get("order_id")
	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		o.errorLog.Println("ERROR_02_CancelOrder: invalid order ID", err)
		utils.BadRequest(w, err)
		return
	}

	err = o.DB.CancelOrder(r.Context(), orderID, branchID)
	if err != nil {
		o.errorLog.Println("ERROR_03_CancelOrder: ", err)
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error   bool   `json:"error"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Order cancelled successfully"

	utils.WriteJSON(w, http.StatusOK, resp)
}

// GetOrderDetailsByID
func (o *OrderHandler) GetOrderDetailsByID(w http.ResponseWriter, r *http.Request) {
	orderIDStr := r.URL.Query().Get("order_id")
	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		o.errorLog.Println("ERROR_01_GetOrderDetailsByID: invalid order ID", err)
		utils.BadRequest(w, err)
		return
	}

	order, err := o.DB.GetOrderDetailsByID(r.Context(), orderID)
	if err != nil {
		o.errorLog.Println("ERROR_02_GetOrderDetailsByID: ", err)
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error   bool          `json:"error"`
		Status  string        `json:"status"`
		Message string        `json:"message"`
		Order   *models.Order `json:"order"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Order fetched successfully"
	resp.Order = order

	utils.WriteJSON(w, http.StatusOK, resp)
}

// GetOrderItemsByMemoNo
func (o *OrderHandler) GetOrderItemsByMemoNo(w http.ResponseWriter, r *http.Request) {
	memoNo := strings.TrimSpace(r.URL.Query().Get("memo_no"))
	if memoNo == "" {
		o.errorLog.Println("ERROR_01_GetOrderItemsByMemoNo: Missing memo no")
		utils.BadRequest(w, errors.New("Missing memo_no"))
		return
	}

	items, err := o.DB.GetOrderItemsByMemoNo(r.Context(), memoNo)
	if err != nil {
		o.errorLog.Println("ERROR_02_GetOrderItemsByMemoNo: ", err)
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error   bool                `json:"error"`
		Message string              `json:"message"`
		Items   []*models.OrderItem `json:"items"`
	}
	resp.Error = false
	resp.Message = "Order items fetched successfully"
	resp.Items = items

	utils.WriteJSON(w, http.StatusOK, resp)
}

// ListOrdersWithFilter - list all or filter by customerID / salesManID
func (o *OrderHandler) ListOrders(w http.ResponseWriter, r *http.Request) {

	//read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		o.errorLog.Println("ERROR_03_ListOrdersWithFilter: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}
	orders, err := o.DB.ListOrders(r.Context(), branchID)
	if err != nil {
		o.errorLog.Println("ERROR_03_ListOrdersWithFilter: ", err)
		utils.BadRequest(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"error":  false,
		"status": "success",
		"orders": orders,
	})
}

// ListOrdersPaginatedHandler
func (o *OrderHandler) ListOrdersPaginatedHandler(w http.ResponseWriter, r *http.Request) {
	pageNoStr := r.URL.Query().Get("pageNo")
	pageLengthStr := r.URL.Query().Get("pageLength")
	status := r.URL.Query().Get("status")
	sortByDate := r.URL.Query().Get("sort_by_date") // "asc" or "desc"

	pageNo, _ := strconv.Atoi(pageNoStr)
	if pageNo <= 0 {
		pageNo = 1
	}
	pageLength, _ := strconv.Atoi(pageLengthStr)
	if pageLength == 0 {
		pageLength = 10
	}
	//read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		o.errorLog.Println("ERROR_01_ListOrdersPaginatedHandler: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}
	orders, err := o.DB.ListOrdersPaginated(r.Context(), pageNo, pageLength, status, sortByDate, branchID)
	if err != nil {
		o.errorLog.Println("ERROR_01_ListOrdersPaginated: ", err)
		utils.BadRequest(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"error":  false,
		"status": "success",
		"orders": orders,
	})
}

// ListOrdersByStatusHandler
func (o *OrderHandler) ListOrdersByStatusHandler(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		utils.BadRequest(w, errors.New("status query param required"))
		return
	}
	//read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		o.errorLog.Println("ERROR_01_ListOrdersByStatusHandler: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}
	orders, err := o.DB.ListOrdersByStatus(r.Context(), status, branchID)
	if err != nil {
		o.errorLog.Println("ERROR_01_ListOrdersByStatus: ", err)
		utils.BadRequest(w, err)
		return
	}
	var resp struct {
		Error   bool            `json:"error"`
		Status  string          `json:"status"`
		Message string          `json:"message"`
		Orders  []*models.Order `json:"orders"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = strings.ToTitle(status) + " orders retrieved successfully"
	resp.Orders = orders
	utils.WriteJSON(w, http.StatusOK, resp)
}
func (o *OrderHandler) GetOrderSummaryHandler(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if startDate == "" || endDate == "" {
		utils.BadRequest(w, fmt.Errorf("start_date and end_date query params are required"))
		return
	}

	//read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		o.errorLog.Println("ERROR_01_GetOrderSummaryHandler: Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}
	summary, err := o.DB.GetOrderSummary(r.Context(), startDate, endDate, branchID)
	if err != nil {
		o.errorLog.Println("ERROR_01_GetOrderSummaryHandler: ", err)
		utils.BadRequest(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"error":   false,
		"status":  "success",
		"summary": summary,
	})
}
