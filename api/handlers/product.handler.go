package api

import (
	"log"
	"net/http"

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
	products, err := h.DB.GetProducts(r.Context())
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
