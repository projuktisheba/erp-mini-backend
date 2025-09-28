package api

import (
	"errors"
	"log"
	"net/http"

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
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := utils.ReadJSON(w, r, &req); err != nil {
		h.errorLog.Println("ERROR_01_Signin:", err)
		utils.BadRequest(w, err)
		return
	}

	// Validate credentials from DB
	user, err := h.DB.EmployeeRepo.GetEmployeeByEmail(r.Context(), req.Username)
	if err != nil || !utils.CheckPassword(req.Password, user.Password) {
		h.errorLog.Println("ERROR_02_Signin: invalid credentials")
		utils.BadRequest(w, errors.New("invalid username or password"))
		return
	}

	// Generate JWT
	token, err := utils.GenerateJWT(models.JWT{
		ID:        user.ID,
		Name:      user.FirstName + user.LastName,
		Username:  user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, h.JWTConfig)

	if err != nil {
		h.errorLog.Println("ERROR_03_Signin: failed to generate JWT", err)
		utils.BadRequest(w, err)
		return
	}

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
