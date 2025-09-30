package api

import (
	"log"
	"net/http"

	"github.com/projuktisheba/erp-mini-api/internal/dbrepo"
	"github.com/projuktisheba/erp-mini-api/internal/models"
	"github.com/projuktisheba/erp-mini-api/internal/utils"
)

type AccountHandler struct {
	DB       *dbrepo.AccountRepo
	infoLog  *log.Logger
	errorLog *log.Logger
}

func NewAccountHandler(db *dbrepo.AccountRepo, infoLog *log.Logger, errorLog *log.Logger) *AccountHandler {
	return &AccountHandler{
		DB:       db,
		infoLog:  infoLog,
		errorLog: errorLog,
	}
}

func (a *AccountHandler) GetAccountsHandler(w http.ResponseWriter, r *http.Request) {
	accounts, err := a.DB.GetAccounts(r.Context())
	if err != nil {
		http.Error(w, "Failed to fetch accounts", http.StatusInternalServerError)
		return
	}
	var resp struct {
		Error    bool              `json:"error"`
		Status   string            `json:"status"`
		Message  string            `json:"message"`
		Accounts []*models.Account `json:"accounts"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Accounts fetched successfully"
	resp.Accounts = accounts

	utils.WriteJSON(w, http.StatusCreated, resp)

}
func (a *AccountHandler) GetAccountNamesHandler(w http.ResponseWriter, r *http.Request) {
	accounts, err := a.DB.GetAccountsNames(r.Context())
	if err != nil {
		http.Error(w, "Failed to fetch accounts", http.StatusInternalServerError)
		return
	}
	var resp struct {
		Error    bool                    `json:"error"`
		Status   string                  `json:"status"`
		Message  string                  `json:"message"`
		Accounts []*models.AccountNameID `json:"accounts"`
	}
	resp.Error = false
	resp.Status = "success"
	resp.Message = "Accounts fetched successfully"
	resp.Accounts = accounts

	utils.WriteJSON(w, http.StatusCreated, resp)

}
