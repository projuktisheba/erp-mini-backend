package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
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

	//set status to pending
	orderDetails.Status = "pending"

	// Only generate if MemoNo is empty
	if strings.TrimSpace(orderDetails.MemoNo) == "" {
		totalOrders, err := o.DB.GetOrderCount(r.Context())
		if err != nil {
			// Fallback to UUID
			orderDetails.MemoNo = uuid.NewString()
		} else {
			// Use zero-padded sequential number
			// Always pad to 6 digits: 000001, 000002, ...
			orderDetails.MemoNo = fmt.Sprintf("%06d", totalOrders+1)
		}
	}

	// Create the order
	err = o.DB.CreateOrder(r.Context(), &orderDetails)
	if err != nil {
		o.errorLog.Println("ERROR_02_AddOrder: ", err)
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

	err = o.DB.UpdateOrder(r.Context(), &orderDetails)
	if err != nil {

		o.errorLog.Println("ERROR_02_UpdateOrder: ", err)
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
	orderIDStr := r.URL.Query().Get("order_id")
	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		o.errorLog.Println("ERROR_01_UpdateOrderStatus: invalid order ID", err)
		utils.BadRequest(w, err)
		return
	}

	err = o.DB.CheckoutOrder(r.Context(), orderID, "checkout")
	if err != nil {
		o.errorLog.Println("ERROR_03_UpdateOrderStatus: ", err)
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
	var req struct{
		OrderID int64 `json:"order_id`
		DeliveredBy int64 `json:"delivered_by"`
		PaidAmount float64 `json:"paid_amount"`
		PaymentAccountID int64 `json:"payment_account_id"`
	}
	err := utils.ReadJSON(w, r, &req)
	if err != nil {
		o.errorLog.Println("ERROR_01_UpdateOrderStatus: invalid order ID", err)
		utils.BadRequest(w, err)
		return
	}

	err = o.DB.ConfirmDelivery(r.Context(), req.OrderID, req.DeliveredBy, req.PaidAmount, req.PaymentAccountID)
	if err != nil {
		o.errorLog.Println("ERROR_03_UpdateOrderStatus: ", err)
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

// CancelOrder
func (o *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	orderIDStr := r.URL.Query().Get("order_id")
	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		o.errorLog.Println("ERROR_01_CancelOrder: invalid order ID", err)
		utils.BadRequest(w, err)
		return
	}

	err = o.DB.CancelOrder(r.Context(), orderID)
	if err != nil {
		o.errorLog.Println("ERROR_02_CancelOrder: ", err)
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

// ListOrdersWithFilter - list all or filter by customerID / salesManID
func (o *OrderHandler) ListOrdersWithFilter(w http.ResponseWriter, r *http.Request) {
	customerIDStr := r.URL.Query().Get("customer_id")
	salesManIDStr := r.URL.Query().Get("sales_man_id")
	var customerID, salesManID *int64
	if customerIDStr != "" {
		id, err := strconv.ParseInt(customerIDStr, 10, 64)
		if err != nil {
			o.errorLog.Println("ERROR_01_ListOrdersWithFilter: invalid customerID", err)
			utils.BadRequest(w, err)
			return
		}
		customerID = &id
	}
	if salesManIDStr != "" {
		id, err := strconv.ParseInt(salesManIDStr, 10, 64)
		if err != nil {
			o.errorLog.Println("ERROR_02_ListOrdersWithFilter: invalid salesManID", err)
			utils.BadRequest(w, err)
			return
		}
		salesManID = &id
	}

	orders, err := o.DB.ListOrdersWithItems(r.Context(), customerID, salesManID)
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

	orders, err := o.DB.ListOrdersPaginated(r.Context(), pageNo, pageLength, status, sortByDate)
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

	orders, err := o.DB.ListOrdersByStatus(r.Context(), status)
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

	summary, err := o.DB.GetOrderSummary(r.Context(), startDate, endDate)
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
