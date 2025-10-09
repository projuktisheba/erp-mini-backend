package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/projuktisheba/erp-mini-api/internal/dbrepo"
	"github.com/projuktisheba/erp-mini-api/internal/models"
	"github.com/projuktisheba/erp-mini-api/internal/utils"
)

type AuthHandler struct {
	DB        *dbrepo.DBRepository
	JWTConfig models.JWTConfig
	infoLog   *log.Logger
	errorLog  *log.Logger
}

func NewAuthHandler(db *dbrepo.DBRepository, JWTConfig models.JWTConfig, infoLog *log.Logger, errorLog *log.Logger) *AuthHandler {
	return &AuthHandler{
		DB:        db,
		JWTConfig: JWTConfig,
		infoLog:   infoLog,
		errorLog:  errorLog,
	}
}

func (h *AuthHandler) Signin(w http.ResponseWriter, r *http.Request) {
	type signinRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var req signinRequest
	if err := utils.ReadJSON(w, r, &req); err != nil {
		h.errorLog.Println("ERROR_01_Signin: invalid JSON:", err)
		utils.BadRequest(w, fmt.Errorf("invalid request payload: %w", err))
		return
	}

	// Trim spaces
	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)

	if req.Username == "" || req.Password == "" {
		utils.BadRequest(w, errors.New("username and password are required"))
		return
	}

	// Fetch employee by mobile OR email
	user, err := h.DB.EmployeeRepo.GetEmployeeByUsernameOrMobile(r.Context(), req.Username)
	if err != nil {
		h.errorLog.Println("ERROR_02_Signin: user not found:", err)
		utils.BadRequest(w, errors.New("invalid username or password"))
		return
	}

	// Check password (assumes hashed in DB)
	if !utils.CheckPassword(req.Password, user.Password) {
		h.errorLog.Println("ERROR_03_Signin: password mismatch")
		utils.BadRequest(w, errors.New("invalid username or password"))
		return
	}

	// Generate JWT
	token, err := utils.GenerateJWT(models.JWT{
		ID:        user.ID,
		Name:      user.Name,
		Username:  user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, h.JWTConfig)

	if err != nil {
		h.errorLog.Println("ERROR_04_Signin: failed to generate JWT:", err)
		utils.BadRequest(w, fmt.Errorf("failed to generate token: %w", err))
		return
	}

	// Build response
	resp := struct {
		Error    bool             `json:"error"`
		Token    string           `json:"token"`
		Employee *models.Employee `json:"employee"`
	}{
		Error:    false,
		Token:    token,
		Employee: user,
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}
