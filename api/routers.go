package api

import (
	"net"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/projuktisheba/erp-mini-api/internal/utils"
)

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()

	// --- Global middlewares ---
	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	mux.Use(app.Logger) // logger
	// mux.Use(app.AuthUser) // Authenticate User

	// --- Static file serving for images ---
	// Serves files under ./data/images → accessible via /images/*
	// Example: ./data/images/employee_5/profile.png → /images/employee_5/profile.png

	imageDir := filepath.Join(".", "data", "images")
	fs := http.StripPrefix("/api/v1/images/", http.FileServer(http.Dir(imageDir)))
	mux.Handle("/api/v1/images/*", fs)

	mux.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		ip := "unknown"
		// Try to get the server's primary outbound IP
		if conn, err := net.Dial("udp", "1.1.1.1:80"); err == nil {
			defer conn.Close()
			ip = conn.LocalAddr().(*net.UDPAddr).IP.String()
		}

		resp := map[string]interface{}{
			"status":    "live",
			"server_ip": ip,
		}
		utils.WriteJSON(w, http.StatusOK, resp)
	})

	mux.Post("/api/v1/login", app.Handlers.Auth.Signin)

	//Branches List

	// -------------------- HR(Employee) Routes --------------------
	mux.Route("/api/v1/hr", func(r chi.Router) {
		// r.Use(app.RequireRole(RoleManager))
		// Get single employee by id, email, or mobile (query param)
		// Example: GET /api/v1/hr/employee?id=5
		r.Get("/employee", app.Handlers.Employee.GetEmployeeByID)

		// Add a new employee
		// Example: POST /api/v1/hr/employee
		// Body (JSON): { employee }
		r.Post("/employee", app.Handlers.Employee.AddEmployee)

		// Get paginated list of employees with optional filters
		//allowed-role: 'chairman', 'manager', 'salesperson', 'worker'
		// Example: GET /api/v1/hr/employees?page=1&limit=20&role=salesperson&status=active
		r.Get("/employees", app.Handlers.Employee.PaginatedEmployeeList)

		// Get all active employee names and IDs (lightweight)
		// Example: GET /api/v1/hr/employees/names
		r.Get("/employees/names", app.Handlers.Employee.GetEmployeesNameAndID)

		// Upload employee profile picture (Form Data: id, profile_picture)
		// Example: POST /api/v1/hr/profile-picture
		// Form fields: id=5, profile_picture=file
		r.Post("/employee/profile-picture", app.Handlers.Employee.UploadEmployeeProfilePicture)

		// Update general employee details
		// Example: PUT /api/v1/hr/employee
		// Body (JSON): { id,first_name,last_name,bio,mobile,country,city, address, postal_code, tax_id }
		r.Put("/employee", app.Handlers.Employee.UpdateEmployee)

		// Update employee salary and overtime rate (Admin only)
		// Example: PUT /api/v1/hr/employee/salary
		// Body (JSON): { id, base_salary, overtime_rate }
		r.Put("/employee/salary", app.Handlers.Employee.UpdateEmployeeSalary)

		// Update employee role and status (Admin only)
		// Example: PUT /api/v1/hr/employee/role
		// Body (JSON): { id, role, status }
		r.Put("/employee/role", app.Handlers.Employee.UpdateEmployeeRole)

	})
	// -------------------- attendance Routes --------------------
	mux.Route("/api/v1/hr/attendance", func(r chi.Router) {
		// r.Use(app.RequireRole(RoleManager))
		// Mark or update today's attendance for a single employee
		// Example: POST /api/v1/hr/attendance/5
		// Body (JSON): { id:1, work_date:2025-10-1 overtime_hours: 2 }
		r.Post("/present/single", app.Handlers.Attendance.MarkEmployeePresent)

		// Batch update today's attendance for multiple employees
		// Example: POST /api/v1/hr/attendance/batch
		// Body (JSON): { attendances: [ {id: 5, checkin: "09:00", checkout: "18:00"}, ... ] }
		r.Post("/present/batch", app.Handlers.Attendance.MarkEmployeesPresentBatch)

		// Get calendar-style attendance for one employee (monthly or date range)
		// Example: GET /api/v1/hr/attendance/calendar?employee_id=1&month=2025-09
		// Example: GET /api/v1/hr/attendance/calendar?employee_id=1&start=2025-09-01&end=2025-09-15
		r.Get("/calendar", app.Handlers.Attendance.GetEmployeeCalendar)

		// Get monthly attendance summary (working days, overtime, etc.) for one employee
		// Example: GET /api/v1/hr/attendance/summary?employee_id=1?month=2025-09
		r.Get("/summary", app.Handlers.Attendance.GetEmployeeSummary)

		// Get batch attendance summary for multiple employees in a given month or range
		// Example: GET /api/v1/hr/attendance/batch/summary?month=2025-09
		// Example: GET /api/v1/hr/attendance/batch/summary?start=2025-09-01&end=2025-09-30
		r.Get("/batch/summary", app.Handlers.Attendance.GetBatchSummary)
	})
	// -------------------- Customer Routes --------------------
	mux.Route("/api/v1/mis", func(r chi.Router) {
		// Get single customer by id, mobile, or tax_id (query param)
		// Example: GET /api/v1/mis/customer?id=5
		// Example: GET /api/v1/mis/customer?mobile=017xxxxxxxx
		// Example: GET /api/v1/mis/customer?tax_id=123456789
		r.Get("/customer", app.Handlers.Customer.GetCustomerByID) // Can extend handler to handle mobile/tax_id query too

		// Add a new customer
		// Example: POST /api/v1/mis/customer
		// Body (JSON): { "name", "mobile", "address", "tax_id" }
		r.Post("/customer", app.Handlers.Customer.AddCustomer)

		// Update general customer details
		// Example: PUT /api/v1/mis/customer
		// Body (JSON): { "id", "name", "mobile", "address", "tax_id" }
		r.Put("/customer", app.Handlers.Customer.UpdateCustomerInfo)

		// Update customer due amount
		// Example: PUT /api/v1/mis/customer/due
		// Body (JSON): { "id", "due_amount" }
		r.Put("/customer/due", app.Handlers.Customer.UpdateCustomerDueAmount)

		// Update customer status (active/inactive)
		// Example: PUT /api/v1/mis/customer/status
		// Body (JSON): { "id", "status" }
		r.Put("/customer/status", app.Handlers.Customer.UpdateCustomerStatus)

		// Get paginated list of customers with optional filters
		// Example: GET /api/v1/mis/customers?page=1&limit=20&status=active
		r.Get("/customers", app.Handlers.Customer.GetCustomers) // Use query params: page, limit, status

		// Filter customers by name
		// Example: GET /api/v1/mis/customers/filter?name=John
		r.Get("/customers/filter", app.Handlers.Customer.FilterCustomersByName)

		// Get all active customer names and IDs (lightweight)
		// Example: GET /api/v1/mis/customers/names
		r.Get("/customers/names", app.Handlers.Customer.GetCustomersNameAndID)

		// Get all customers who have some due
		// Example: GET /api/v1/mis/customers/with-due
		r.Get("/customers/with-due", app.Handlers.Customer.GetCustomersWithDueHandler)

	})
	// -------------------- Product Routes --------------------
	mux.Route("/api/v1/products", func(r chi.Router) {
		// r.Use(app.RequireRole(RoleManager))
		// Get all products
		// Example: GET /api/v1/products
		r.Get("/", app.Handlers.Product.GetProductsHandler)
	})

	// -------------------- Order Routes --------------------
	mux.Route("/api/v1/orders", func(r chi.Router) {
		// r.Use(app.RequireRole(RoleManager))
		// Create a new order
		// Example: POST /api/v1/orders
		// Body (JSON): { order object with items }
		r.Post("/", app.Handlers.Order.AddOrder)

		// Update an existing order
		// Example: PUT /api/v1/orders
		// Body (JSON): { order object with items, id must exist }
		r.Put("/", app.Handlers.Order.UpdateOrder)

		// Cancel an order
		// Example: DELETE /api/v1/orders?order_id=123
		r.Delete("/", app.Handlers.Order.CancelOrder)

		// Update order status to checkout
		// Example: PATCH /api/v1//orders/checkout?order_id=123

		r.Patch("/checkout", app.Handlers.Order.CheckoutOrder)

		// Update order status and payment amount from customer as mark the order as delivered
		// Example: PATCH /api/v1/orders/delivery
		// request body :{order_id, paid_amount}
		r.Patch("/delivery", app.Handlers.Order.OrderDelivery)

		// Get order details by ID
		// Example: GET /api/v1/orders?order_id=12
		r.Get("/", app.Handlers.Order.GetOrderDetailsByID)

		// List orders filtered by customerID and/or salesManID
		// Example: GET /api/v1/orders/list
		// Example: GET /api/v1/orders/list?customer_id=1&sales_man_id=2
		r.Get("/list", app.Handlers.Order.ListOrdersWithFilter)

		// List orders with pagination, optional status filter, and sorting by created_at.
		// Example: GET /api/v1/orders/list/paginated?pageNo=1&pageLength=20
		// Example with status filter: GET /api/v1/orders/list/paginated?pageNo=1&pageLength=20&status=checkout
		// Example with sorting ascending: GET /api/v1/orders/list/paginated?pageNo=1&pageLength=20&sort_by_date=asc
		// Example with status filter + ascending sort:
		// GET /api/v1/orders/list/paginated?pageNo=1&pageLength=20&status=pending&sort_by_date=asc
		// If pageLength=-1, all orders will be returned without pagination
		r.Get("/list/paginated", app.Handlers.Order.ListOrdersPaginatedHandler)

		// List orders filtered by status
		// Example: GET /api/v1/orders/list/status?status=progress
		r.Get("/list/status", app.Handlers.Order.ListOrdersByStatusHandler)

		// Get order summary for a specific date range (daily/monthly)
		// Example: GET /api/v1/mis/orders/summary?start_date=2025-09-01&end_date=2025-09-30
		r.Get("/summary", app.Handlers.Order.GetOrderSummaryHandler)
	})
	// -------------------- Transaction Routes --------------------
	mux.Route("/api/v1/accounts", func(r chi.Router) {
		// r.Use(app.RequireRole(RoleManager))
		// Get all accounts
		// Example: GET /api/v1/accounts
		r.Get("/", app.Handlers.Account.GetAccountsHandler)

		// Get all accounts
		// Example: GET /api/v1/accounts/names
		r.Get("/names", app.Handlers.Account.GetAccountNamesHandler)

	})
	// -------------------- Transaction Routes --------------------
	mux.Route("/api/v1/transactions", func(r chi.Router) {
		// r.Use(app.RequireRole(RoleManager))
		// Get transaction summary with filters and date range
		// Example: GET /api/v1/transactions/summary?start_date=2025-09-01&end_date=2025-09-30
		// Optional filters: from_id, to_id, from_type, to_type, transaction_type
		r.Get("/summary", app.Handlers.Transaction.GetTransactionSummaryHandler)

		// Paginated transactions list with optional filters
		// Example: GET /api/v1/transactions/list?pageNo=1&pageLength=20&from_id=5&from_type=customer
		r.Get("/list", app.Handlers.Transaction.ListTransactionsPaginatedHandler)
	})

	// -------------------- Report Routes --------------------
	mux.Route("/api/v1/reports", func(r chi.Router) {
		// r.Use(app.RequireRole(RoleAdmin))
		// Dashboard summary for orders
		// Example: GET /api/v1/reports/dashboard/orders/overview?type=month&date=2025-09-01
		// Note: Acceptable type [daily, weekly, monthly, yearly, all]
		r.Get("/dashboard/orders/overview", app.Handlers.Report.GetOrderOverView)
	})

	return mux
}
