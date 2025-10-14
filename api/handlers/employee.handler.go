package api

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"os"

	"github.com/jackc/pgx/v5"
	"github.com/projuktisheba/erp-mini-api/internal/dbrepo"
	"github.com/projuktisheba/erp-mini-api/internal/models"
	"github.com/projuktisheba/erp-mini-api/internal/utils"
)

var RoleMap = map[string]bool{
	"":            true,
	"chairman":    true,
	"manager":     true,
	"salesperson": true,
	"worker":      true,
}

type EmployeeHandler struct {
	DB       *dbrepo.EmployeeRepo
	infoLog  *log.Logger
	errorLog *log.Logger
}

func NewEmployeeHandler(db *dbrepo.EmployeeRepo, infoLog *log.Logger, errorLog *log.Logger) *EmployeeHandler {
	return &EmployeeHandler{
		DB:       db,
		infoLog:  infoLog,
		errorLog: errorLog,
	}
}
func (e *EmployeeHandler) AddEmployee(w http.ResponseWriter, r *http.Request) {

	//read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		e.errorLog.Println("Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}

	var employeeDetails models.Employee
	err := utils.ReadJSON(w, r, &employeeDetails)

	if err != nil {
		e.errorLog.Println("ERROR_01_AddEmployee", err)
		utils.BadRequest(w, err)
		return
	}
	//set branch id
	employeeDetails.BranchID = branchID
	// Hash a password
	hashed, err := utils.HashPassword(employeeDetails.Password)
	if err != nil {
		e.errorLog.Println("ERROR_01_AddEmployee", err)
		utils.ServerError(w, errors.New("Unable generate the hash password"))
		return
	}
	employeeDetails.Password = hashed

	err = e.DB.CreateEmployee(r.Context(), &employeeDetails)
	if err != nil {
		e.errorLog.Println("ERROR_02_AddEmployee: ", err)
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error    bool             `json:"error"`
		Status   string           `json:"status"`
		Message  string           `json:"message"`
		Employee *models.Employee `json:"employee"`
	}
	resp.Error = false
	resp.Message = "Employee added successfully"
	resp.Employee = &employeeDetails
	utils.WriteJSON(w, http.StatusCreated, resp)
}
func (e *EmployeeHandler) GetEmployeeByID(w http.ResponseWriter, r *http.Request) {
	idParam := strings.TrimSpace(r.URL.Query().Get("id"))
	if idParam == "" {
		e.errorLog.Println("ERROR_01_GetEmployee: Empty user id")
		utils.BadRequest(w, errors.New("ERROR_01_GetEmployee: Empty user id"))
		return
	}
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		e.errorLog.Println("ERROR_02_GetEmployee: Invalid user id")
		utils.BadRequest(w, err)
		return
	}
	employee, err := e.DB.GetEmployeeByID(r.Context(), id)
	if err != nil {
		e.errorLog.Println("ERROR_03_GetEmployee: ", err)
		utils.BadRequest(w, err)
		return
	}
	var resp struct {
		Error    bool             `json:"error"`
		Status   string           `json:"status"`
		Message  string           `json:"message"`
		Employee *models.Employee `json:"employee"`
	}
	resp.Error = false
	resp.Message = "Employee info fetched successfully"
	resp.Employee = employee

	utils.WriteJSON(w, 200, resp)
}
func (e *EmployeeHandler) GetEmployeesNameAndID(w http.ResponseWriter, r *http.Request) {
	//read branch id
	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		e.errorLog.Println("Branch id not found")
		utils.BadRequest(w, errors.New("Branch ID not found. Please include 'X-Branch-ID' header, e.g., X-Branch-ID: 1"))
		return
	}

	employeeRole := strings.TrimSpace(r.URL.Query().Get("role"))

	if _, found := RoleMap[employeeRole]; !found {
		e.errorLog.Println("ERROR_01_GetEmployeesNameAndID: Invalid role, allowed-role:[chairman, manager, salesperson, worker]")
		utils.BadRequest(w, errors.New("Please provide correct role, allowed-role:[chairman, manager, salesperson, worker]"))
		return
	}
	employees, err := e.DB.GetEmployeesNameAndIDByBranchAndRole(r.Context(), branchID, employeeRole)
	var resp struct {
		Error     bool                     `json:"error"`
		Status    string                   `json:"status"`
		Message   string                   `json:"message"`
		Employees []*models.EmployeeNameID `json:"employees"`
	}
	if err != nil || len(employees) == 0 {
		e.errorLog.Println("ERROR_02_GetEmployeesNameAndID:", err)
		resp.Error = true
		resp.Status = "Empty"
		resp.Message = "No employees found"
		utils.WriteJSON(w, 200, resp)
		return
	}

	resp.Error = false
	resp.Status = "success"
	resp.Message = "Employee Names and IDs fetched successfully"
	resp.Employees = employees

	utils.WriteJSON(w, http.StatusOK, resp)
}

// UpdateEmployee updates general employee details
func (e *EmployeeHandler) UpdateEmployee(w http.ResponseWriter, r *http.Request) {
	var employeeDetails models.Employee
	err := utils.ReadJSON(w, r, &employeeDetails)
	if err != nil {
		e.errorLog.Println("ERROR_01_UpdateEmployee", err)
		utils.BadRequest(w, err)
		return
	}
	if employeeDetails.ID == 0 {
		e.errorLog.Println("ERROR_02_UpdateEmployee: Missing employee ID")
		utils.BadRequest(w, errors.New("missing employee ID"))
		return
	}

	err = e.DB.UpdateEmployee(r.Context(), &employeeDetails)
	if err != nil {
		e.errorLog.Println("ERROR_03_UpdateEmployee: ", err)
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error    bool             `json:"error"`
		Status   string           `json:"status"`
		Message  string           `json:"message"`
		Employee *models.Employee `json:"employee"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Employee details updated successfully"
	resp.Employee = &employeeDetails

	utils.WriteJSON(w, http.StatusOK, resp)
}

// UpdateEmployeeSalary updates employee salary and overtime rate
func (e *EmployeeHandler) UpdateEmployeeSalary(w http.ResponseWriter, r *http.Request) {
	var employeeDetails models.Employee
	err := utils.ReadJSON(w, r, &employeeDetails)
	if err != nil {
		e.errorLog.Println("ERROR_01_UpdateEmployeeSalary", err)
		utils.BadRequest(w, err)
		return
	}
	e.infoLog.Println(employeeDetails)
	if employeeDetails.ID == 0 {
		e.errorLog.Println("ERROR_02_UpdateEmployeeSalary: Missing employee ID")
		utils.BadRequest(w, errors.New("missing employee ID"))
		return
	}

	err = e.DB.UpdateEmployeeSalary(r.Context(), &employeeDetails)
	if err != nil {
		e.errorLog.Println("ERROR_03_UpdateEmployeeSalary: ", err)
		if errors.Is(err, pgx.ErrNoRows) {
			utils.BadRequest(w, errors.New("Invalid user id"))
		} else {
			utils.BadRequest(w, err)
		}
		return
	}

	var resp struct {
		Error    bool             `json:"error"`
		Status   string           `json:"status"`
		Message  string           `json:"message"`
		Employee *models.Employee `json:"employee"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Employee salary updated successfully"
	resp.Employee = &employeeDetails

	utils.WriteJSON(w, http.StatusOK, resp)
}

// SubmitSalary generate and give employee salary
func (e *EmployeeHandler) SubmitSalary(w http.ResponseWriter, r *http.Request) {
	var salary struct {
		EmployeeID int64     `json:"employee_id"`
		Amount     float64   `json:"salary_amount"`
		SalaryDate time.Time `json:"salary_date"`
	}
	err := utils.ReadJSON(w, r, &salary)
	if err != nil {
		e.errorLog.Println("ERROR_01_SubmitSalary", err)
		utils.BadRequest(w, err)
		return
	}
	e.infoLog.Println(salary)
	if salary.EmployeeID == 0 {
		e.errorLog.Println("ERROR_02_SubmitSalary: Missing employee ID")
		utils.BadRequest(w, errors.New("missing employee ID"))
		return
	}

	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		e.errorLog.Println("ERROR_03_SubmitSalary: Missing branch ID")
		utils.BadRequest(w, errors.New("missing branch ID"))
		return
	}
	e.infoLog.Println("salary: ",salary, branchID)
	err = e.DB.SubmitSalary(r.Context(), salary.SalaryDate, salary.EmployeeID, branchID, salary.Amount)
	if err != nil {
		e.errorLog.Println("ERROR_04_SubmitSalary: ", err)
		if errors.Is(err, pgx.ErrNoRows) {
			utils.BadRequest(w, errors.New("Invalid user id"))
		} else {
			utils.BadRequest(w, err)
		}
		return
	}

	var resp struct {
		Error   bool   `json:"error"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Employee salary updated successfully"
	utils.WriteJSON(w, http.StatusOK, resp)
}

// UpdateEmployeeRole updates employee role and status
func (e *EmployeeHandler) UpdateEmployeeRole(w http.ResponseWriter, r *http.Request) {
	var employeeDetails models.Employee
	err := utils.ReadJSON(w, r, &employeeDetails)
	if err != nil {
		e.errorLog.Println("ERROR_01_UpdateEmployeeRole", err)
		utils.BadRequest(w, err)
		return
	}

	if employeeDetails.ID == 0 {
		e.errorLog.Println("ERROR_02_UpdateEmployeeRole: Missing employee ID")
		utils.BadRequest(w, errors.New("missing employee ID"))
		return
	}

	err = e.DB.UpdateEmployeeRole(r.Context(), &employeeDetails)
	if err != nil {
		e.errorLog.Println("ERROR_03_UpdateEmployeeRole: ", err)
		utils.BadRequest(w, err)
		return
	}

	var resp struct {
		Error    bool             `json:"error"`
		Status   string           `json:"status"`
		Message  string           `json:"message"`
		Employee *models.Employee `json:"employee"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Employee role and status updated successfully"
	resp.Employee = &employeeDetails

	utils.WriteJSON(w, http.StatusOK, resp)
}

// PaginatedEmployeeList handles fetching a paginated, filtered list of employees.
// Supports query params: page, limit, role, status
func (e *EmployeeHandler) PaginatedEmployeeList(w http.ResponseWriter, r *http.Request) {

	// Extract query params
	pageParam := strings.TrimSpace(r.URL.Query().Get("page"))
	limitParam := strings.TrimSpace(r.URL.Query().Get("limit"))
	roleFilter := strings.TrimSpace(r.URL.Query().Get("role"))
	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))

	// Defaults
	page := 0 // 0 means list all
	limit := 0
	//read branch id
	branchID := utils.GetBranchID(r)

	// Parse page param
	if pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		} else {
			e.errorLog.Println("ERROR_01_PaginatedEmployeeList: Invalid page")
			utils.BadRequest(w, errors.New("ERROR_01_PaginatedEmployeeList: Invalid page"))
			return
		}
	}

	// Parse limit param
	if limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = l
		} else {
			e.errorLog.Println("ERROR_02_PaginatedEmployeeList: Invalid limit")
			utils.BadRequest(w, errors.New("ERROR_02_PaginatedEmployeeList: Invalid limit"))
			return
		}
	}

	// Check role param
	if _, found := RoleMap[roleFilter]; !found {
		e.errorLog.Println("ERROR_03_PaginatedEmployeeList: Invalid role, allowed-role:[chairman, manager, salesperson, worker]")
		utils.BadRequest(w, errors.New("Please provide correct role, allowed-role:[chairman, manager, salesperson, worker]"))
		return
	}

	// Set default sorting
	sortBy := strings.TrimSpace(r.URL.Query().Get("sort_by"))
	if sortBy == "" {
		sortBy = "id"
	}
	sortOrder := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("sort_order")))
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "ASC"
	}

	// Fetch filtered employees from DB
	employees, total, err := e.DB.PaginatedEmployeeList(r.Context(), page, limit, branchID, roleFilter, statusFilter, sortBy, sortOrder)
	if err != nil {
		e.errorLog.Println("ERROR_04_PaginatedEmployeeList: ", err)
		utils.BadRequest(w, err)
		return
	}

	// Calculate pagination
	totalPages := 1
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
	} else {
		page = 1
		limit = total
	}

	// Build response
	var resp struct {
		Error      bool               `json:"error"`
		Status     string             `json:"status"`
		Message    string             `json:"message"`
		Page       int                `json:"page"`
		Limit      int                `json:"limit"`
		Total      int                `json:"total"`
		TotalPages int                `json:"total_pages"`
		BranchID   int64              `json:"branch_id,omitempty"`
		Role       string             `json:"role,omitempty"`
		StatusF    string             `json:"status_filter,omitempty"`
		Employees  []*models.Employee `json:"employees"`
	}

	resp.Error = false
	resp.Status = "success"
	resp.Message = "Employee list fetched successfully"
	resp.Page = page
	resp.Limit = limit
	resp.Total = total
	resp.TotalPages = totalPages
	resp.BranchID = branchID
	resp.Role = roleFilter
	resp.StatusF = statusFilter
	resp.Employees = employees

	utils.WriteJSON(w, http.StatusOK, resp)
}

// UploadEmployeeProfilePicture handles uploading a profile picture for an employee.
// It saves the image to ./data/images/employee_{id}/profile.{ext} (cross-platform safe).
func (e *EmployeeHandler) UploadEmployeeProfilePicture(w http.ResponseWriter, r *http.Request) {

	// --- Step 1: Parse multipart form (10 MB limit) ---
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		e.errorLog.Println("ERROR_01_UploadEmployeeProfilePicture:", err)
		utils.BadRequest(w, err)
		return
	}

	// --- Step 2: Validate Employee ID ---
	idValue := r.FormValue("id")
	if idValue == "" {
		e.errorLog.Println("ERROR_02_UploadEmployeeProfilePicture: Empty user id")
		utils.BadRequest(w, errors.New("empty user id"))
		return
	}
	id, err := strconv.Atoi(idValue)
	if err != nil {
		e.errorLog.Println("ERROR_03_UploadEmployeeProfilePicture: Invalid user id")
		utils.BadRequest(w, err)
		return
	}

	// --- Step 3: Get file from form field "profile_picture" ---
	file, handler, err := r.FormFile("profile_picture")
	if err != nil {
		e.errorLog.Println("ERROR_04_UploadEmployeeProfilePicture:", err)
		utils.BadRequest(w, errors.New("profile_picture field is required"))
		return
	}
	defer file.Close()

	// --- Step 4: Validate file extension (only jpg, jpeg, png allowed) ---
	ext := strings.ToLower(filepath.Ext(handler.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		e.errorLog.Println("ERROR_05_UploadEmployeeProfilePicture: Invalid file type")
		utils.BadRequest(w, errors.New("only jpg, jpeg, png files are allowed"))
		return
	}

	// --- Step 5: Build directory path ./data/images/employee_{id} ---
	uploadDir := filepath.Join(".", "data", "images", fmt.Sprintf("employee_%d", id))
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		e.errorLog.Println("ERROR_06_UploadEmployeeProfilePicture:", err)
		utils.BadRequest(w, err)
		return
	}

	// --- Step 6: Build final file path ./data/images/employee_{id}/profile{ext} ---
	filePath := filepath.Join(uploadDir, "profile"+ext)
	dst, err := os.Create(filePath)
	if err != nil {
		e.errorLog.Println("ERROR_07_UploadEmployeeProfilePicture:", err)
		utils.BadRequest(w, err)
		return
	}
	defer dst.Close()

	// --- Step 7: Copy file content to destination ---
	if _, err := io.Copy(dst, file); err != nil {
		e.errorLog.Println("ERROR_08_UploadEmployeeProfilePicture:", err)
		utils.BadRequest(w, err)
		return
	}

	// --- Step 8: check whether profile picture is saved ---
	if _, err := os.Stat(filePath); err != nil {
		e.errorLog.Println("ERROR_09_UploadEmployeeProfilePicture: File not saved", err)
		utils.BadRequest(w, errors.New("profile picture not saved"))
		return
	}
	avatarLink := fmt.Sprintf("/images/employee_%d/profile%s", id, ext) ///images/employee_1/profile.png

	err = e.DB.UpdateEmployeeAvatarLink(r.Context(), id, avatarLink)
	if err != nil {
		e.errorLog.Println("ERROR_10_UploadEmployeeProfilePicture: File saved but unable to update database", err)
		utils.BadRequest(w, errors.New("file saved but unable to update database"))
		return
	}

	resp := struct {
		Error      bool   `json:"error"`
		Status     string `json:"status"`
		Message    string `json:"message"`
		AvatarLink string `json:"avatar_link"`
	}{
		Error:      false,
		Status:     "success",
		Message:    "Profile picture uploaded successfully",
		AvatarLink: avatarLink,
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}


func (e *EmployeeHandler) RecordWorkerDailyProgress(w http.ResponseWriter, r *http.Request){

	branchID := utils.GetBranchID(r)
	if branchID == 0 {
		e.errorLog.Println("ERROR_01_RecordWorkerDailyProgress: Missing branch ID")
		utils.BadRequest(w, errors.New("missing branch ID"))
		return
	}
	var requestBody models.WorkerProgress
	err := utils.ReadJSON(w, r, &requestBody)

	if err != nil {
		e.errorLog.Println("ERROR_02_RecordWorkerDailyProgress: Unable to read JSON", err)
		utils.BadRequest(w, errors.New("Error reading JSON"))
		return
	}
	requestBody.BranchID = branchID
	err = e.DB.UpdateWorkerProgress(r.Context(), requestBody)
	if err != nil {
		e.errorLog.Println("ERROR_03_RecordWorkerDailyProgress: Database error", err)
		utils.BadRequest(w, err)
		return
	}
	var response models.Response

	response.Error = false
	response.Message = "Progress record updated successfully"
	utils.WriteJSON(w, http.StatusOK, response)
}