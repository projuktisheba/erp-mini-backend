package api

import (
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
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	mux.Use(app.Logger) // Simple logger

	// --- Static file serving for images ---
	// Serves files under ./data/images → accessible via /images/*
	// Example: ./data/images/employee_5/profile.png → /images/employee_5/profile.png

	imageDir := filepath.Join(".", "data", "images")
	fs := http.StripPrefix("/api/v1/images/", http.FileServer(http.Dir(imageDir)))
	mux.Handle("/api/v1/images/*", fs)

	// --- Health check endpoint ---
	mux.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		utils.WriteJSON(w, 200, "Live")
	})

	mux.Post("/api/v1/login", app.Handlers.Auth.Signin)
	// --- HR (Employee) Routes ---
	mux.Route("/api/v1/hr", func(r chi.Router) {
		// Get single employee by id, email, or mobile (query param)
		// Example: GET /api/v1/hr/employee?id=5
		r.Get("/employee", app.Handlers.Employee.GetEmployee)

		// Add a new employee
		// Example: POST /api/v1/hr/employee
		// Body (JSON): { employee }
		r.Post("/employee", app.Handlers.Employee.AddEmployee)

		// Get paginated list of employees with optional filters
		// Example: GET /api/v1/hr/employees?page=1&limit=20&role=admin&status=active
		r.Get("/employees", app.Handlers.Employee.PaginatedEmployeeList)

		// Upload employee profile picture (Form Data: id, profile_picture)
		// Example: POST /api/v1/hr/profile-picture
		// Form fields: id=5, profile_picture=@/path/to/image.jpg
		r.Post("/employee/profile-picture", app.Handlers.Employee.UploadEmployeeProfilePicture)

		// Update general employee details
		// Example: PUT /api/v1/hr/employee
		// Body (JSON): { employee }
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

	return mux
}
