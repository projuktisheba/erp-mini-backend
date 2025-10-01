package dbrepo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/projuktisheba/erp-mini-api/internal/models"
)

type OrderRepo struct {
	db *pgxpool.Pool
}

func NewOrderRepo(db *pgxpool.Pool) *OrderRepo {
	return &OrderRepo{db: db}
}

func (r *OrderRepo) CreateOrder(ctx context.Context, newOrder *models.Order) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// --- Step 1: Insert order ---
	var orderID int64
	err = tx.QueryRow(ctx, `
        INSERT INTO orders (memo_no, order_date, sales_man_id, customer_id,
                            total_payable_amount, advance_payment_amount, payment_account_id,
                            status, delivery_date, delivered_by, notes)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
        RETURNING id
    `,
		newOrder.MemoNo, newOrder.OrderDate, newOrder.SalesManID, newOrder.CustomerID,
		newOrder.TotalPayableAmount, newOrder.AdvancePaymentAmount, newOrder.PaymentAccountID,
		newOrder.Status, newOrder.DeliveryDate, newOrder.DeliveredBy, newOrder.Notes,
	).Scan(&orderID)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	// --- Step 2: Insert order items ---
	for _, it := range newOrder.Items {
		_, err := tx.Exec(ctx, `
            INSERT INTO order_items (order_id, product_id, quantity, unit_price)
            VALUES ($1,$2,$3,$4)
        `, orderID, it.ProductID, it.Quantity, it.UnitPrice)
		if err != nil {
			return fmt.Errorf("insert order item: %w", err)
		}
	}

	// --- Step 3: Update account balance ---
	if newOrder.AdvancePaymentAmount > 0 && newOrder.PaymentAccountID != nil {
		_, err := tx.Exec(ctx, `
            UPDATE accounts
            SET current_balance = current_balance + $1
            WHERE id=$2
        `, newOrder.AdvancePaymentAmount, *newOrder.PaymentAccountID)
		if err != nil {
			return fmt.Errorf("update account balance: %w", err)
		}

		// --- Step 4: Record transaction ---
		_, err = tx.Exec(ctx, `
            INSERT INTO transactions (from_entity_id, from_entity_type, to_entity_id, to_entity_type,
                                      amount, transaction_type, notes)
            VALUES ($1,'customers',$2,'accounts',$3,'payment','Advance payment for order')
        `, newOrder.CustomerID, *newOrder.PaymentAccountID, newOrder.AdvancePaymentAmount)
		if err != nil {
			return fmt.Errorf("insert transaction: %w", err)
		}
	}

	// --- Step 5: Update customer due ---
	dueAmount := newOrder.TotalPayableAmount - newOrder.AdvancePaymentAmount
	if dueAmount > 0 {
		_, err := tx.Exec(ctx, `
            UPDATE customers
            SET due_amount = due_amount + $1
            WHERE id=$2
        `, dueAmount, newOrder.CustomerID)
		if err != nil {
			return fmt.Errorf("update customer due: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// UpdateOrder updates an existing order and its items, adjusting account and customer due
func (r *OrderRepo) UpdateOrder(ctx context.Context, newOrder *models.Order) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// --- Step 1: Load old order ---
	var oldOrder models.Order
	err = tx.QueryRow(ctx, `
        SELECT id, memo_no, order_date, sales_man_id, customer_id,
               total_payable_amount, advance_payment_amount, payment_account_id,
               status, delivery_date, delivered_by, notes
        FROM orders
        WHERE id=$1
    `, newOrder.ID).Scan(
		&oldOrder.ID,
		&oldOrder.MemoNo,
		&oldOrder.OrderDate,
		&oldOrder.SalesManID,
		&oldOrder.CustomerID,
		&oldOrder.TotalPayableAmount,
		&oldOrder.AdvancePaymentAmount,
		&oldOrder.PaymentAccountID,
		&oldOrder.Status,
		&oldOrder.DeliveryDate,
		&oldOrder.DeliveredBy,
		&oldOrder.Notes,
	)
	if err != nil {
		return fmt.Errorf("load old order: %w", err)
	}

	// --- Step 2: Load old order items ---
	rows, err := tx.Query(ctx, `
        SELECT id, product_id, quantity, unit_price
        FROM order_items
        WHERE order_id=$1
    `, oldOrder.ID)
	if err != nil {
		return fmt.Errorf("load old items: %w", err)
	}
	defer rows.Close()

	oldItems := map[int64]models.OrderItem{}
	for rows.Next() {
		var it models.OrderItem
		if err := rows.Scan(&it.ID, &it.ProductID, &it.Quantity, &it.UnitPrice); err != nil {
			return fmt.Errorf("scan old item: %w", err)
		}
		oldItems[it.ProductID] = it
	}

	// --- Step 3: Adjust account balance if advance changed ---

	advanceDiff := newOrder.AdvancePaymentAmount - oldOrder.AdvancePaymentAmount
	if advanceDiff != 0 {
		if newOrder.PaymentAccountID == nil {
			return errors.New("payment_account_id required when advance changes")
		}

		// Update account balance
		_, err := tx.Exec(ctx, `
        UPDATE accounts
        SET current_balance = current_balance + $1
        WHERE id=$2
    `, advanceDiff, *newOrder.PaymentAccountID)
		if err != nil {
			return fmt.Errorf("update account balance: %w", err)
		}

		// Insert transaction for the difference
		trxType := "payment"
		notes := "Advance payment adjusted for order"
		if advanceDiff < 0 {
			trxType = "refund"
			advanceDiff = -advanceDiff
			notes = "Advance payment refund for order"
		}
		_, err = tx.Exec(ctx, `
        INSERT INTO transactions (from_entity_id, from_entity_type, to_entity_id, to_entity_type,
                                  amount, transaction_type, notes)
        VALUES ($1,'customers',$2,'accounts',$3,$4,$5)
    `, newOrder.CustomerID, *newOrder.PaymentAccountID, advanceDiff, trxType, notes)
		if err != nil {
			return fmt.Errorf("insert transaction: %w", err)
		}
	}

	// --- Step 4: Adjust customer due ---
	oldDue := oldOrder.TotalPayableAmount - oldOrder.AdvancePaymentAmount
	newDue := newOrder.TotalPayableAmount - newOrder.AdvancePaymentAmount
	dueDiff := newDue - oldDue
	if dueDiff != 0 {
		res, err := tx.Exec(ctx, `
            UPDATE customers
            SET due_amount = due_amount + $1
            WHERE id=$2
        `, dueDiff, newOrder.CustomerID)
		if err != nil {
			return fmt.Errorf("update customer due: %w", err)
		}
		if res.RowsAffected() == 0 {
			return errors.New("invalid customer_id")
		}
	}

	// --- Step 5: Update orders table ---
	_, err = tx.Exec(ctx, `
        UPDATE orders
        SET memo_no=$1, order_date=$2, sales_man_id=$3, customer_id=$4,
            total_payable_amount=$5, advance_payment_amount=$6, payment_account_id=$7,
            status=$8, delivery_date=$9, delivered_by=$10, notes=$11,
            updated_at=CURRENT_TIMESTAMP
        WHERE id=$12
    `, newOrder.MemoNo, newOrder.OrderDate, newOrder.SalesManID, newOrder.CustomerID,
		newOrder.TotalPayableAmount, newOrder.AdvancePaymentAmount, newOrder.PaymentAccountID,
		newOrder.Status, newOrder.DeliveryDate, newOrder.DeliveredBy, newOrder.Notes,
		newOrder.ID)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	// --- Step 6: Reconcile order_items ---
	newItems := map[int64]*models.OrderItem{}
	for _, it := range newOrder.Items {
		newItems[it.ProductID] = it
	}

	// Insert or update items
	for pid, it := range newItems {
		if old, ok := oldItems[pid]; ok {
			if old.Quantity != it.Quantity || old.UnitPrice != it.UnitPrice {
				_, err := tx.Exec(ctx, `
                    UPDATE order_items
                    SET quantity=$1, unit_price=$2
                    WHERE id=$3
                `, it.Quantity, it.UnitPrice, old.ID)
				if err != nil {
					return fmt.Errorf("update order_item: %w", err)
				}
			}
		} else {
			_, err := tx.Exec(ctx, `
                INSERT INTO order_items (order_id, product_id, quantity, unit_price)
                VALUES ($1,$2,$3,$4)
            `, newOrder.ID, it.ProductID, it.Quantity, it.UnitPrice)
			if err != nil {
				return fmt.Errorf("insert order_item: %w", err)
			}
		}
	}

	// Delete removed items
	for pid, old := range oldItems {
		if _, ok := newItems[pid]; !ok {
			_, err := tx.Exec(ctx, `DELETE FROM order_items WHERE id=$1`, old.ID)
			if err != nil {
				return fmt.Errorf("delete order_item: %w", err)
			}
		}
	}

	// --- Step 7: Commit transaction ---
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

// UpdateOrderStatus updates the status of an order by id
func (r *OrderRepo) CheckoutOrder(ctx context.Context, orderID int64, newStatus string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Validate status
	allowedStatuses := map[string]bool{
		"progress":  true,
		"checkout":  true,
		"delivery":  true,
		"cancelled": true,
		"returned":  true,
	}
	if !allowedStatuses[newStatus] {
		return fmt.Errorf("invalid status: %s", newStatus)
	}

	// Update the order status
	res, err := tx.Exec(ctx, `
        UPDATE orders
        SET status=$1, updated_at=CURRENT_TIMESTAMP
        WHERE id=$2
    `, newStatus, orderID)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("order id %d not found", orderID)
	}

	return tx.Commit(ctx)
}

// CancelOrder cancels an order, removes all items, and reverts financials
func (r *OrderRepo) CancelOrder(ctx context.Context, orderID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// --- Step 1: Load the order ---
	var order models.Order
	err = tx.QueryRow(ctx, `
        SELECT id, customer_id, advance_payment_amount, total_payable_amount, payment_account_id
        FROM orders
        WHERE id=$1
    `, orderID).Scan(&order.ID, &order.CustomerID, &order.AdvancePaymentAmount, &order.TotalPayableAmount, &order.PaymentAccountID)
	if err != nil {
		return fmt.Errorf("load order: %w", err)
	}

	// --- Step 2: Revert account balance ---
	// Adjusted CancelOrder function
	if order.AdvancePaymentAmount > 0 && order.PaymentAccountID != nil {
		// Revert account balance
		_, err := tx.Exec(ctx, `
        UPDATE accounts
        SET current_balance = current_balance - $1
        WHERE id=$2
    `, order.AdvancePaymentAmount, *order.PaymentAccountID)
		if err != nil {
			return fmt.Errorf("revert account balance: %w", err)
		}

		// Record refund transaction
		_, err = tx.Exec(ctx, `
        INSERT INTO transactions (from_entity_id, from_entity_type, to_entity_id, to_entity_type,
                                  amount, transaction_type, notes)
        VALUES ($1,'accounts',$2,'customers',$3,'refund','Refund for canceled order')
    `, *order.PaymentAccountID, order.CustomerID, order.AdvancePaymentAmount)
		if err != nil {
			return fmt.Errorf("insert refund transaction: %w", err)
		}
	}

	// --- Step 3: Revert customer due ---
	dueAmount := order.TotalPayableAmount - order.AdvancePaymentAmount
	if dueAmount > 0 {
		res, err := tx.Exec(ctx, `
            UPDATE customers
            SET due_amount = due_amount - $1, updated_at = CURRENT_TIMESTAMP
            WHERE id=$2
        `, dueAmount, order.CustomerID)
		if err != nil {
			return fmt.Errorf("revert customer due: %w", err)
		}
		if res.RowsAffected() == 0 {
			return fmt.Errorf("invalid customer_id")
		}
	}

	// --- Step 4: Update order status to 'cancelled' instead of deleting ---
	res, err := tx.Exec(ctx, `
        UPDATE orders
        SET status='cancelled', updated_at=CURRENT_TIMESTAMP
        WHERE id=$1
    `, orderID)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("order id %d not found", orderID)
	}

	// --- Step 5: Commit transaction ---
	return tx.Commit(ctx)
}

func (r *OrderRepo) ConfirmDelivery(ctx context.Context, orderID int64, deliveredBy int64, paidAmount float64, paymentAccountID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// --- Step 1: Load the order and customer ---
	var order models.Order
	err = tx.QueryRow(ctx, `
        SELECT customer_id, total_payable_amount, advance_payment_amount
        FROM orders
        WHERE id=$1
    `, orderID).Scan(&order.CustomerID, &order.TotalPayableAmount, &order.AdvancePaymentAmount)
	if err != nil {
		return fmt.Errorf("load order: %w", err)
	}

	// Calculate remaining due after this payment
	remainingDue := (order.TotalPayableAmount - order.AdvancePaymentAmount) - paidAmount
	if remainingDue < 0 {
		remainingDue = 0
	}

	// --- Step 2: Update customer due ---
	_, err = tx.Exec(ctx, `
        UPDATE customers
        SET due_amount = due_amount - $1
        WHERE id=$2
    `, paidAmount, order.CustomerID)
	if err != nil {
		return fmt.Errorf("update customer due: %w", err)
	}

	// --- Step 3: Update account balance ---
	_, err = tx.Exec(ctx, `
        UPDATE accounts
        SET current_balance = current_balance + $1
        WHERE id=$2
    `, paidAmount, paymentAccountID)
	if err != nil {
		return fmt.Errorf("update account balance: %w", err)
	}

	// --- Step 4: Insert transaction ---
	_, err = tx.Exec(ctx, `
        INSERT INTO transactions (from_entity_id, from_entity_type, to_entity_id, to_entity_type,
                                  amount, transaction_type, notes)
        VALUES ($1,'customers',$2,'accounts',$3,'payment','Payment on delivery')
    `, order.CustomerID, paymentAccountID, paidAmount)
	if err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}

	// --- Step 5: Update order status and delivery info ---
	_, err = tx.Exec(ctx, `
        UPDATE orders
        SET status='delivery', delivered_by=$1, delivery_date=NOW(), updated_at=NOW()
        WHERE id=$2
    `, deliveredBy, orderID)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}

	return tx.Commit(ctx)
}

// GetOrderDetailsByID fetches an order and its items by order ID
func (r *OrderRepo) GetOrderDetailsByID(ctx context.Context, orderID int64) (*models.Order, error) {
	// Step 1: Fetch order with customer and employee names
	var order models.Order
	var customerName, employeeName string

	err := r.db.QueryRow(ctx, `
        SELECT 
            o.id, o.memo_no, o.order_date, o.sales_man_id, o.customer_id,
            o.total_payable_amount, o.advance_payment_amount, o.due_amount,
            o.payment_account_id, o.status, o.delivery_date, o.delivered_by, o.notes,
            o.created_at, o.updated_at,
            c.name AS customer_name,
            e.fname || ' ' || e.lname AS employee_name
        FROM orders o
        LEFT JOIN customers c ON o.customer_id = c.id
        LEFT JOIN employees e ON o.sales_man_id = e.id
        WHERE o.id = $1
    `, orderID).Scan(
		&order.ID,
		&order.MemoNo,
		&order.OrderDate,
		&order.SalesManID,
		&order.CustomerID,
		&order.TotalPayableAmount,
		&order.AdvancePaymentAmount,
		&order.DueAmount,
		&order.PaymentAccountID,
		&order.Status,
		&order.DeliveryDate,
		&order.DeliveredBy,
		&order.Notes,
		&order.CreatedAt,
		&order.UpdatedAt,
		&customerName,
		&employeeName,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}

	order.CustomerName = customerName
	order.SalesManName = employeeName

	// Step 2: Fetch order items with product names
	rows, err := r.db.Query(ctx, `
        SELECT 
            oi.id, oi.product_id, p.product_name AS product_name, oi.quantity, oi.unit_price
        FROM order_items oi
        LEFT JOIN products p ON oi.product_id = p.id
        WHERE oi.order_id = $1
    `, orderID)
	if err != nil {
		return nil, fmt.Errorf("fetch order items: %w", err)
	}
	defer rows.Close()

	items := []*models.OrderItem{}
	for rows.Next() {
		var it models.OrderItem
		var productName string
		if err := rows.Scan(&it.ID, &it.ProductID, &productName, &it.Quantity, &it.UnitPrice); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		it.TotalPrice = it.UnitPrice * float64(it.Quantity)
		it.OrderID = orderID
		it.ProductName = productName
		items = append(items, &it)
	}
	order.Items = items

	return &order, nil
}

// ListOrdersWithItems fetches a list of orders filtered by customer_id or sales_man_id
func (r *OrderRepo) ListOrdersWithItems(ctx context.Context, customerID, salesManID *int64) ([]*models.Order, error) {
	query := `
		SELECT 
		    o.id, o.memo_no, o.order_date, o.sales_man_id, o.customer_id,
		    o.total_payable_amount, o.advance_payment_amount, o.due_amount, o.payment_account_id,
		    o.status, o.delivery_date, o.delivered_by, o.notes, o.created_at, o.updated_at,
		    c.name AS customer_name,
		    e.fname || ' ' || e.lname AS employee_name
		FROM orders o
		LEFT JOIN customers c ON o.customer_id = c.id
		LEFT JOIN employees e ON o.sales_man_id = e.id
		WHERE 1=1
	`

	args := []interface{}{}
	argID := 1

	if customerID != nil {
		query += ` AND o.customer_id=$` + strconv.Itoa(argID)
		args = append(args, *customerID)
		argID++
	}
	if salesManID != nil {
		query += ` AND o.sales_man_id=$` + strconv.Itoa(argID)
		args = append(args, *salesManID)
		argID++
	}

	query += ` ORDER BY o.order_date DESC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []*models.Order{}
	for rows.Next() {
		var o models.Order
		var customerName, employeeName string

		err := rows.Scan(
			&o.ID, &o.MemoNo, &o.OrderDate, &o.SalesManID, &o.CustomerID,
			&o.TotalPayableAmount, &o.AdvancePaymentAmount, &o.DueAmount, &o.PaymentAccountID,
			&o.Status, &o.DeliveryDate, &o.DeliveredBy, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
			&customerName, &employeeName,
		)
		if err != nil {
			return nil, err
		}

		o.CustomerName = customerName
		o.SalesManName = employeeName

		// Load order items
		itemRows, err := r.db.Query(ctx, `
			SELECT id, product_id, quantity, unit_price
			FROM order_items
			WHERE order_id=$1
		`, o.ID)
		if err != nil {
			return nil, err
		}

		items := []*models.OrderItem{}
		for itemRows.Next() {
			var it models.OrderItem
			if err := itemRows.Scan(&it.ID, &it.ProductID, &it.Quantity, &it.UnitPrice); err != nil {
				itemRows.Close()
				return nil, err
			}
			it.TotalPrice = it.UnitPrice * float64(it.Quantity)
			items = append(items, &it)
		}
		itemRows.Close()
		o.Items = items

		orders = append(orders, &o)
	}

	return orders, nil
}

func (r *OrderRepo) ListOrdersPaginated(ctx context.Context, pageNo, pageLength int, status, sortByDate string) ([]*models.Order, error) {
	query := `
		SELECT 
		    o.id, o.memo_no, o.order_date, o.sales_man_id, o.customer_id, 
		    o.total_payable_amount, o.advance_payment_amount, o.due_amount, o.payment_account_id,
		    o.status, o.delivery_date, o.delivered_by, o.notes, o.created_at, o.updated_at,
		    c.name AS customer_name,
		    e.fname || ' ' || e.lname AS employee_name
		FROM orders o
		LEFT JOIN customers c ON o.customer_id = c.id
		LEFT JOIN employees e ON o.sales_man_id = e.id
		WHERE 1=1
	`

	args := []interface{}{}
	argIdx := 1

	// --- Status filter ---
	if status != "" {
		query += fmt.Sprintf(" AND o.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	// --- Sorting ---
	sortOrder := "DESC"
	if strings.ToLower(sortByDate) == "asc" {
		sortOrder = "ASC"
	}
	query += " ORDER BY o.created_at " + sortOrder

	// --- Pagination ---
	if pageLength != -1 {
		offset := (pageNo - 1) * pageLength
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
		args = append(args, pageLength, offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []*models.Order{}
	for rows.Next() {
		var o models.Order
		var customerName, employeeName string

		err := rows.Scan(
			&o.ID, &o.MemoNo, &o.OrderDate, &o.SalesManID, &o.CustomerID,
			&o.TotalPayableAmount, &o.AdvancePaymentAmount, &o.DueAmount, &o.PaymentAccountID,
			&o.Status, &o.DeliveryDate, &o.DeliveredBy, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
			&customerName, &employeeName,
		)
		if err != nil {
			return nil, err
		}

		o.CustomerName = customerName
		o.SalesManName = employeeName

		orders = append(orders, &o)
	}

	return orders, nil
}

// ListOrdersByStatus retrieves a list of orders from the database that match the specified status.
// It queries the "orders" table, filtering by the provided status and ordering the results by order date in descending order.
// Each row is scanned into a models.Order struct and appended to the result slice.
// Returns a slice of pointers to models.Order and an error if any database operation fails.
//
// Parameters:
//   - ctx: context.Context for controlling cancellation and deadlines.
//   - status: string representing the order status to filter by.
//
// Returns:
//   - []*models.Order: slice of orders matching the given status.
//   - error: error encountered during query or scanning, or nil if successful.
func (r *OrderRepo) ListOrdersByStatus(ctx context.Context, status string) ([]*models.Order, error) {
	rows, err := r.db.Query(ctx, `
		SELECT 
		    o.id, o.memo_no, o.order_date, o.sales_man_id, o.customer_id, 
		    o.total_payable_amount, o.advance_payment_amount, o.due_amount, o.payment_account_id,
		    o.status, o.delivery_date, o.delivered_by, o.notes, o.created_at, o.updated_at,
		    c.name AS customer_name,
		    e.fname || ' ' || e.lname AS employee_name
		FROM orders o
		LEFT JOIN customers c ON o.customer_id = c.id
		LEFT JOIN employees e ON o.sales_man_id = e.id
		WHERE o.status = $1
		ORDER BY o.order_date DESC
	`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []*models.Order{}
	for rows.Next() {
		var o models.Order
		var customerName, employeeName string

		if err := rows.Scan(
			&o.ID, &o.MemoNo, &o.OrderDate, &o.SalesManID, &o.CustomerID,
			&o.TotalPayableAmount, &o.AdvancePaymentAmount, &o.DueAmount, &o.PaymentAccountID,
			&o.Status, &o.DeliveryDate, &o.DeliveredBy, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
			&customerName, &employeeName,
		); err != nil {
			return nil, err
		}

		o.CustomerName = customerName
		o.SalesManName = employeeName

		orders = append(orders, &o)
	}

	return orders, nil
}

func (r *OrderRepo) GetOrderSummary(ctx context.Context, startDate, endDate string) (map[string]float64, error) {
	query := `
		SELECT
			COALESCE(SUM(total_payable_amount),0) AS total_amount,
			COALESCE(SUM(advance_payment_amount),0) AS total_advance_payment,
			COALESCE(SUM(due_amount),0) AS total_due
		FROM orders
		WHERE order_date BETWEEN $1 AND $2
	`

	var totalAmount, totalAdvance, totalDue float64
	err := r.db.QueryRow(ctx, query, startDate, endDate).Scan(&totalAmount, &totalAdvance, &totalDue)
	if err != nil {
		return nil, err
	}

	return map[string]float64{
		"total_amount":          totalAmount,
		"total_advance_payment": totalAdvance,
		"total_due":             totalDue,
	}, nil
}

func (r *OrderRepo) GetOrderCount(ctx context.Context) (int, error) {
	var total int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) as total_orders FROM orders").Scan(&total)
	return total, err
}
