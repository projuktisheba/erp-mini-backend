package dbrepo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/projuktisheba/erp-mini-api/internal/models"
)

// ============================== Product Repository ==============================
type ProductRepo struct {
	db *pgxpool.Pool
}

func NewProductRepo(db *pgxpool.Pool) *ProductRepo {
	return &ProductRepo{db: db}
}

// GetProducts fetches all products from the database.
func (s *ProductRepo) GetProducts(ctx context.Context) ([]*models.Product, error) {
	query := `
        SELECT 
            id, product_code, product_name, product_description, 
            product_status, mrp, warranty, category_id, brand_id, 
            stock_alert_level, created_at, updated_at
        FROM products
        ORDER BY id;
    `

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error fetching products: %w", err)
	}
	defer rows.Close()

	var products []*models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(
			&p.ID,
			&p.ProductCode,
			&p.ProductName,
			&p.ProductDescription,
			&p.ProductStatus,
			&p.MRP,
			&p.Warranty,
			&p.CategoryID,
			&p.BrandID,
			&p.StockAlertLevel,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("error scanning product: %w", err)
		}
		products = append(products, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return products, nil
}
