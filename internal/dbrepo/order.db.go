package dbrepo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

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

	// set values
	newOrder.Status = "pending"
	totalItems := int64(0)
	for _, it := range newOrder.Items {
		totalItems += it.Quantity
	}
	// --- Step 1: Insert order ---
	err = tx.QueryRow(ctx, `
        INSERT INTO orders (memo_no, branch_id, order_date, salesperson_id, customer_id, total_payable_amount, advance_payment_amount, payment_account_id, status, delivery_date, exit_date, notes, total_items)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12, $13)
        RETURNING id, memo_no
    `,
		newOrder.MemoNo, newOrder.BranchID, newOrder.OrderDate, newOrder.SalespersonID, newOrder.CustomerID,
		newOrder.TotalPayableAmount, newOrder.AdvancePaymentAmount, newOrder.PaymentAccountID,
		newOrder.Status, newOrder.DeliveryDate, newOrder.ExitDate, newOrder.Notes, totalItems,
	).Scan(&newOrder.ID, &newOrder.MemoNo)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	// --- Step 2: Insert order items ---
	for _, it := range newOrder.Items {
		_, err := tx.Exec(ctx, `
            INSERT INTO order_items (memo_no, product_id, quantity, subtotal)
            VALUES ($1,$2,$3,$4)
        `, newOrder.MemoNo, it.ProductID, it.Quantity, it.TotalPrice)
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
		//insert transaction
		transaction := &models.Transaction{
			BranchID:        newOrder.BranchID,
			MemoNo:          newOrder.MemoNo,
			FromID:          newOrder.CustomerID,
			FromType:        "customers",
			ToID:            *newOrder.PaymentAccountID,
			ToType:          "accounts",
			Amount:          newOrder.AdvancePaymentAmount,
			TransactionType: "payment",
			CreatedAt:       newOrder.OrderDate,
			Notes:           "Advance payment for order",
		}
		_, err = CreateTransactionTx(ctx, tx, transaction) // silently add transaction
		if err != nil {
			return fmt.Errorf("insert transaction: %w", err)
		}
	}
	// Update top_sheet => increase cash
	topSheet := &models.TopSheet{
		Date:       newOrder.OrderDate,
		BranchID:   newOrder.BranchID,
		OrderCount: totalItems,
		Pending:    totalItems,
	}

	// safer: lookup account type (cash/bank)
	var acctType string
	err = tx.QueryRow(ctx, `SELECT type FROM accounts WHERE id=$1`, *newOrder.PaymentAccountID).Scan(&acctType)
	if err != nil {
		return fmt.Errorf("lookup account type: %w", err)
	}
	if acctType == "bank" {
		topSheet.Bank = newOrder.AdvancePaymentAmount
	} else {
		topSheet.Cash = newOrder.AdvancePaymentAmount
	}

	err = SaveTopSheetTx(tx, ctx, topSheet)
	if err != nil {
		return fmt.Errorf("save top-sheet: %w", err)
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

	// Step 6: Update salesperson daily progress record
	salespersonProgress := models.SalespersonProgress{
		Date:       newOrder.OrderDate,
		BranchID:   newOrder.BranchID,
		EmployeeID: newOrder.SalespersonID,
		OrderCount: totalItems,
		// SaleAmount: newOrder.TotalPayableAmount, // DEBUG uncomment if client wants to add
	}
	err = UpdateSalespersonProgressReportTx(tx, ctx, &salespersonProgress)
	if err != nil {
		return fmt.Errorf("failed to update employee progress: %w", err)
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
        SELECT id, memo_no, order_date, salesperson_id, customer_id,
               total_payable_amount, advance_payment_amount, payment_account_id,
               status, delivery_date, exit_date, notes
        FROM orders
        WHERE id=$1
    `, newOrder.ID).Scan(
		&oldOrder.ID,
		&oldOrder.MemoNo,
		&oldOrder.OrderDate,
		&oldOrder.SalespersonID,
		&oldOrder.CustomerID,
		&oldOrder.TotalPayableAmount,
		&oldOrder.AdvancePaymentAmount,
		&oldOrder.PaymentAccountID,
		&oldOrder.Status,
		&oldOrder.DeliveryDate,
		&oldOrder.ExitDate,
		&oldOrder.Notes,
	)
	if err != nil {
		return fmt.Errorf("load old order: %w", err)
	}

	// --- Step 2: Load old order items ---
	rows, err := tx.Query(ctx, `
        SELECT id, product_id, quantity, subtotal
        FROM order_items
        WHERE memo_no=$1
    `, oldOrder.MemoNo)
	if err != nil {
		return fmt.Errorf("load old items: %w", err)
	}
	defer rows.Close()

	oldItemsCount := int64(0)
	oldItems := map[int64]models.OrderItem{}
	for rows.Next() {
		var it models.OrderItem
		if err := rows.Scan(&it.ID, &it.ProductID, &it.Quantity, &it.TotalPrice); err != nil {
			return fmt.Errorf("scan old item: %w", err)
		}
		oldItems[it.ProductID] = it
		oldItemsCount += it.Quantity
	}
	newItemsCount := int64(0)
	newItems := map[int64]*models.OrderItem{}
	for _, it := range newOrder.Items {
		newItems[it.ProductID] = it
		newItemsCount += it.Quantity
	}
	//item differences
	itemDiff := newItemsCount - oldItemsCount
	// --- Step 5: Update orders table ---
	_, err = tx.Exec(ctx, `
        UPDATE orders
        SET order_date=$2, salesperson_id=$3, customer_id=$4,
            total_payable_amount=$5, advance_payment_amount=$6, payment_account_id=$7,
            status=$8, delivery_date=$9, exit_date=$10, notes=$11, total_items=total_items+$12, updated_at=CURRENT_TIMESTAMP
        WHERE id=$13
    `, newOrder.OrderDate, newOrder.SalespersonID, newOrder.CustomerID,
		newOrder.TotalPayableAmount, newOrder.AdvancePaymentAmount, newOrder.PaymentAccountID,
		newOrder.Status, newOrder.DeliveryDate, newOrder.ExitDate, newOrder.Notes, itemDiff, newOrder.ID)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	// --- Step 6: Reconcile order_items ---

	// Insert or update items
	for pid, it := range newItems {
		if old, ok := oldItems[pid]; ok {
			if old.Quantity != it.Quantity || old.TotalPrice != it.TotalPrice {
				_, err := tx.Exec(ctx, `
                    UPDATE order_items
                    SET quantity=$1, subtotal=$2
                    WHERE id=$3
                `, it.Quantity, it.TotalPrice, old.ID)
				if err != nil {
					return fmt.Errorf("update order_item: %w", err)
				}
			}
		} else {
			_, err := tx.Exec(ctx, `
                INSERT INTO order_items (memo_no, product_id, quantity, subtotal)
                VALUES ($1,$2,$3,$4)
            `, newOrder.MemoNo, it.ProductID, it.Quantity, it.TotalPrice) // FIX: use MemoNo
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
	}

	// Update top_sheet
	topSheet := &models.TopSheet{
		Date:       newOrder.OrderDate,
		BranchID:   newOrder.BranchID,
		OrderCount: itemDiff,
		Pending:    itemDiff,
	}

	// safer: lookup account type (cash/bank)
	var acctType string
	err = tx.QueryRow(ctx, `SELECT type FROM accounts WHERE id=$1`, *newOrder.PaymentAccountID).Scan(&acctType)
	if err != nil {
		return fmt.Errorf("lookup account type: %w", err)
	}
	if acctType == "bank" {
		topSheet.Bank = advanceDiff
	} else {
		topSheet.Cash = advanceDiff
	}

	err = SaveTopSheetTx(tx, ctx, topSheet)
	if err != nil {
		return fmt.Errorf("save topsheet: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE transactions
		SET 
			from_entity_id = $1,
			from_entity_type = $2,
			to_entity_id = $3,
			to_entity_type = $4,
			amount = $5,
			notes = $6,
			updated_at = CURRENT_TIMESTAMP
		WHERE memo_no = $7
		`,
		newOrder.CustomerID,           // $1
		"customers",                   // $2
		*newOrder.PaymentAccountID,    // $3
		"accounts",                    // $4
		newOrder.AdvancePaymentAmount, // $5
		"Advance payment adjustment for order cancellation", // $6
		newOrder.MemoNo, // $7
	)
	if err != nil {
		return fmt.Errorf("update transaction: %w", err)
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
	// Step 6: Update salesperson daily progress record
	salespersonProgress := models.SalespersonProgress{
		Date:       newOrder.OrderDate,
		BranchID:   newOrder.BranchID,
		EmployeeID: newOrder.SalespersonID,
		OrderCount: itemDiff,
		// SaleAmount: newOrder.TotalPayableAmount, // DEBUG uncomment if client wants to add
	}
	err = UpdateSalespersonProgressReportTx(tx, ctx, &salespersonProgress)
	if err != nil {
		return fmt.Errorf("failed to update employee progress: %w", err)
	}

	// --- Step 7: Commit transaction ---
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

// UpdateOrderStatus updates the status of an order by id, and only increase "checkout" in top_sheet
func (r *OrderRepo) CheckoutOrder(ctx context.Context, orderID int64, branchID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	// First, get current status
	var currentStatus string
	err = tx.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1`, orderID).Scan(&currentStatus)
	if err != nil {
		return fmt.Errorf("fetch current status: %w", err)
	}

	// Prevent invalid updates
	if currentStatus != "pending" {
		return fmt.Errorf("cannot mark the order as checkout with current status '%s'", currentStatus)
	}

	newStatus := "checkout"
	memoNo := ""
	// Update the order status
	err = tx.QueryRow(ctx, `
        UPDATE orders
        SET status=$1, updated_at=CURRENT_TIMESTAMP
        WHERE id=$2
		RETURNING memo_no
    `, newStatus, orderID).Scan(&memoNo)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	totalItems := int64(0)
	err = tx.QueryRow(ctx, `SELECT SUM(COALESCE(quantity, 0)) FROM order_items WHERE memo_no=$1`, memoNo).Scan(&totalItems)
	if err != nil {
		return fmt.Errorf("unable to count total items: %w", err)
	}
	// Update top_sheet -> increase checkout
	topSheet := &models.TopSheet{
		Date:     time.Now(),
		BranchID: branchID,
		Pending:  -totalItems,
		Checkout: totalItems,
	}
	if err := SaveTopSheetTx(tx, ctx, topSheet); err != nil {
		return fmt.Errorf("save topsheet: %w", err)
	}

	return tx.Commit(ctx)
}

// CancelOrder cancels an order, removes all items, refund, revert balances, and decrease cash/bank in top_sheet
func (r *OrderRepo) CancelOrder(ctx context.Context, orderID int64, branchID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// First, get current status
	var currentStatus string
	var totalItems, ItemsDelivered int64
	err = tx.QueryRow(ctx, `SELECT status, total_items, items_delivered FROM orders WHERE id = $1`, orderID).Scan(&currentStatus, &totalItems, &ItemsDelivered)
	if err != nil {
		return fmt.Errorf("fetch current status: %w", err)
	}

	// Prevent invalid updates
	if currentStatus == "cancelled" || (currentStatus == "delivery" &&  totalItems == ItemsDelivered){
		return fmt.Errorf("cannot cancel order with current status '%s'", currentStatus)
	}

	// Load & mark order cancelled
	var order models.Order
	err = tx.QueryRow(ctx, `
		UPDATE orders
		SET status = 'cancelled',
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING id, branch_id, memo_no, customer_id, salesperson_id, advance_payment_amount, total_payable_amount, payment_account_id, total_items, items_delivered
	`, orderID).Scan(
		&order.ID,
		&order.BranchID,
		&order.MemoNo,
		&order.CustomerID,
		&order.SalespersonID,
		&order.AdvancePaymentAmount,
		&order.TotalPayableAmount,
		&order.PaymentAccountID,
		&order.TotalItems,
		&order.ItemsDelivered,
	)
	if err != nil {
		return fmt.Errorf("update order to cancelled: %w", err)
	}
	totalItems -= order.ItemsDelivered
	// Revert account balance + refund transaction
	if order.AdvancePaymentAmount > 0 && ItemsDelivered == 0 && order.PaymentAccountID != nil {
		_, err := tx.Exec(ctx, `
			UPDATE accounts
			SET current_balance = current_balance - $1
			WHERE id=$2
		`, order.AdvancePaymentAmount, *order.PaymentAccountID)
		if err != nil {
			return fmt.Errorf("revert account balance: %w", err)
		}
		//insert transaction
		transaction := &models.Transaction{
			BranchID:        order.BranchID,
			MemoNo:          order.MemoNo,
			FromID:          *order.PaymentAccountID,
			FromType:        "accounts",
			ToID:            order.CustomerID,
			ToType:          "customers",
			Amount:          order.AdvancePaymentAmount,
			TransactionType: "refund",
			CreatedAt:       time.Now().UTC(),
			Notes:           "Refund for cancelled order",
		}
		_, err = CreateTransactionTx(ctx, tx, transaction) // silently add transaction

		if err != nil {
			return fmt.Errorf("insert refund transaction: %w", err)
		}
	}
	// TopSheet -> Decrease cash/bank
	topSheet := &models.TopSheet{
		Date:      time.Now(),
		BranchID:  branchID,
		Cancelled: totalItems,
	}
	if currentStatus == "pending" {
		topSheet.Pending = -totalItems
	} else if currentStatus == "checkout" {
		topSheet.Checkout = -totalItems
		// TODO : restock existing items to the product_stock_registry
	}
	// safer: lookup account type (cash/bank)
	var acctType string
	err = tx.QueryRow(ctx, `SELECT type FROM accounts WHERE id=$1`, *order.PaymentAccountID).Scan(&acctType)
	if err != nil {
		return fmt.Errorf("lookup account type: %w", err)
	}

	if currentStatus != "partial" {
		if acctType == "bank" {
			topSheet.Bank = -order.AdvancePaymentAmount
		} else {
			topSheet.Cash = -order.AdvancePaymentAmount
		}
	}
	if err := SaveTopSheetTx(tx, ctx, topSheet); err != nil {
		return fmt.Errorf("save topsheet: %w", err)
	}
	// Step : Update salesperson daily progress record
	salespersonProgress := models.SalespersonProgress{
		Date:       time.Now().UTC(),
		BranchID:   branchID,
		EmployeeID: order.SalespersonID,
		OrderCount: -totalItems,
		// SaleAmount: order.TotalPayableAmount, // DEBUG uncomment if client wants to add
	}
	err = UpdateSalespersonProgressReportTx(tx, ctx, &salespersonProgress)
	if err != nil {
		return fmt.Errorf("failed to update employee progress: %w", err)
	}
	// Revert customer due
	dueAmount := order.TotalPayableAmount - order.AdvancePaymentAmount
	if dueAmount > 0 {
		_, err := tx.Exec(ctx, `
			UPDATE customers
			SET due_amount = due_amount - $1, updated_at = CURRENT_TIMESTAMP
			WHERE id=$2
		`, dueAmount, order.CustomerID)
		if err != nil {
			return fmt.Errorf("revert customer due: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *OrderRepo) ConfirmDelivery(ctx context.Context, branchID, orderID, totalItems int64, paidAmount float64, paymentAccountID int64, exitDate time.Time) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// --- Step 1: Load order ---
	order := &models.Order{}
	err = tx.QueryRow(ctx, `
        SELECT id, memo_no, branch_id, order_date, salesperson_id, customer_id, total_payable_amount, advance_payment_amount, payment_account_id, status, total_items, items_delivered
        FROM orders
        WHERE id=$1
    `, orderID).Scan(
		&order.ID,
		&order.MemoNo,
		&order.BranchID,
		&order.OrderDate,
		&order.SalespersonID,
		&order.CustomerID,
		&order.TotalPayableAmount,
		&order.AdvancePaymentAmount,
		&order.PaymentAccountID,
		&order.Status,
		&order.TotalItems,
		&order.ItemsDelivered,
	)
	if err != nil {
		return fmt.Errorf("load order: %w", err)
	}

	if order.Status != "checkout" && order.Status != "delivery" {
		return fmt.Errorf("order status is not 'checkout'")
	}
	fmt.Println(order, totalItems)
	if order.TotalItems-order.ItemsDelivered < totalItems {
		totalItems = order.TotalItems - order.ItemsDelivered
	}

	fmt.Println(order, totalItems)
	fmt.Println("Total Items: ", order.TotalItems, "  Items Delivered: ", order.ItemsDelivered)
	fmt.Println("Current: ", totalItems)
	// --- Step 2: Update order status, exit date, delivered items ---
	_, err = tx.Exec(ctx, `
        UPDATE orders
        SET status='delivery',
			advance_payment_amount = COALESCE(advance_payment_amount, 0) + $1,
			items_delivered = COALESCE(items_delivered, 0) + $2,
			exit_date=$3
        WHERE id=$4
    `, paidAmount, totalItems, exitDate, orderID)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	// --- Step 3: Update payment account if any payment made ---
	if paidAmount > 0 {
		_, err := tx.Exec(ctx, `
            UPDATE accounts
            SET current_balance = current_balance + $1, updated_at=CURRENT_TIMESTAMP
            WHERE id=$2
        `, paidAmount, paymentAccountID)
		if err != nil {
			return fmt.Errorf("update account balance: %w", err)
		}

		//insert transaction
		transaction := &models.Transaction{
			BranchID:        order.BranchID,
			MemoNo:          order.MemoNo,
			FromID:          order.CustomerID,
			FromType:        "customers",
			ToID:            *order.PaymentAccountID,
			ToType:          "accounts",
			Amount:          paidAmount,
			TransactionType: "payment",
			CreatedAt:       order.OrderDate,
			Notes:           "Payment during product delivery",
		}
		_, err = CreateTransactionTx(ctx, tx, transaction) // silently add transaction
		if err != nil {
			return fmt.Errorf("insert transaction: %w", err)
		}
	}

	// --- Step 5: Update top sheet ---
	topSheet := &models.TopSheet{
		Date:     exitDate,
		BranchID: branchID,
		Checkout: -totalItems,
		Delivery: totalItems,
	}
	var acctType string
	err = tx.QueryRow(ctx, `SELECT type FROM accounts WHERE id=$1`, paymentAccountID).Scan(&acctType)
	if err != nil {
		return fmt.Errorf("lookup account type: %w", err)
	}
	if acctType == "bank" {
		topSheet.Bank = paidAmount
	} else {
		topSheet.Cash = paidAmount
	}
	err = SaveTopSheetTx(tx, ctx, topSheet)
	if err != nil {
		return fmt.Errorf("save top-sheet: %w", err)
	}

	// --- Step 6: Update customer due ---
	if paidAmount != 0 {
		_, err := tx.Exec(ctx, `
            UPDATE customers
            SET due_amount = due_amount - $1
            WHERE id=$2
        `, paidAmount, order.CustomerID)
		if err != nil {
			return fmt.Errorf("update customer due: %w", err)
		}
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
            o.id, o.memo_no, o.order_date, o.salesperson_id, o.customer_id,
            o.total_payable_amount, o.advance_payment_amount,
            o.payment_account_id, o.status, o.delivery_date, o.exit_date, o.notes,
            o.created_at, o.updated_at,
            c.name AS customer_name,
            e.name AS employee_name
        FROM orders o
        LEFT JOIN customers c ON o.customer_id = c.id
        LEFT JOIN employees e ON o.salesperson_id = e.id
        WHERE o.id = $1
    `, orderID).Scan(
		&order.ID,
		&order.MemoNo,
		&order.OrderDate,
		&order.SalespersonID,
		&order.CustomerID,
		&order.TotalPayableAmount,
		&order.AdvancePaymentAmount,
		&order.PaymentAccountID,
		&order.Status,
		&order.DeliveryDate,
		&order.ExitDate,
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
	order.SalespersonName = employeeName

	// Step 2: Fetch order items with product names
	rows, err := r.db.Query(ctx, `
        SELECT 
            oi.id, oi.product_id, p.product_name AS product_name, oi.quantity, oi.subtotal
        FROM order_items oi
        LEFT JOIN products p ON oi.product_id = p.id
        WHERE oi.memo_no = $1
    `, order.MemoNo)
	if err != nil {
		return nil, fmt.Errorf("fetch order items: %w", err)
	}
	defer rows.Close()

	items := []*models.OrderItem{}
	for rows.Next() {
		var it models.OrderItem
		var productName string
		if err := rows.Scan(&it.ID, &it.ProductID, &productName, &it.Quantity, &it.TotalPrice); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		it.MemoNo = order.MemoNo
		it.ProductName = productName
		items = append(items, &it)
	}
	order.Items = items

	return &order, nil
}

// GetOrderItemsByMemoNo fetches order items by memo_no
func (r *OrderRepo) GetOrderItemsByMemoNo(ctx context.Context, memoNo string) ([]*models.OrderItem, error) {
	// Fetch order items with product names by memo no.
	rows, err := r.db.Query(ctx, `
        SELECT 
            oi.id, oi.product_id, p.product_name AS product_name, oi.quantity, oi.subtotal
        FROM order_items oi
        LEFT JOIN products p ON oi.product_id = p.id
        WHERE oi.memo_no = $1
    `, memoNo)
	if err != nil {
		return nil, fmt.Errorf("fetch order items: %w", err)
	}
	defer rows.Close()

	items := []*models.OrderItem{}
	for rows.Next() {
		var it models.OrderItem
		var productName string
		if err := rows.Scan(&it.ID, &it.ProductID, &productName, &it.Quantity, &it.TotalPrice); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		it.MemoNo = memoNo
		it.ProductName = productName
		items = append(items, &it)
	}

	return items, nil
}

// Deprecated: ListOrdersWithItems fetches a list of orders filtered by customer_id or salesperson_id
func (r *OrderRepo) ListOrdersWithItems(ctx context.Context, customerID, SalespersonID *int64, branchID int64) ([]*models.Order, error) {
	query := `
		SELECT 
		    o.id, o.memo_no, o.branch_id, o.order_date, o.salesperson_id, o.customer_id,
		    o.total_payable_amount, o.advance_payment_amount, o.payment_account_id,
		    o.status, o.delivery_date, o.exit_date, o.notes, o.created_at, o.updated_at,
			c.name, c.mobile, e.name, e.mobile
		FROM orders o
		LEFT JOIN customers c ON o.customer_id = c.id
		LEFT JOIN employees e ON o.salesperson_id = e.id
		WHERE 1=1
	`

	args := []interface{}{}
	argID := 1

	if customerID != nil {
		query += ` AND o.customer_id=$` + strconv.Itoa(argID)
		args = append(args, *customerID)
		argID++
	}
	if SalespersonID != nil {
		query += ` AND o.salesperson_id=$` + strconv.Itoa(argID)
		args = append(args, *SalespersonID)
		argID++
	}
	if branchID != 0 {
		query += ` AND o.branch_id=$` + strconv.Itoa(argID)
		args = append(args, branchID)
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
		err := rows.Scan(
			&o.ID, &o.MemoNo, &o.BranchID, &o.OrderDate, &o.SalespersonID, &o.CustomerID,
			&o.TotalPayableAmount, &o.AdvancePaymentAmount, &o.PaymentAccountID,
			&o.Status, &o.DeliveryDate, &o.ExitDate, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
			&o.CustomerName, &o.CustomerMobile, &o.SalespersonName, &o.SalespersonMobile,
		)
		if err != nil {
			return nil, err
		}
		// Load order items
		itemRows, err := r.db.Query(ctx, `
			SELECT oi.id, oi.product_id, oi.quantity, oi.subtotal, p.product_name
			FROM order_items oi
			LEFT JOIN products p ON p.id = oi.product_id
			WHERE memo_no=$1
		`, o.MemoNo)
		if err != nil {
			return nil, err
		}

		items := []*models.OrderItem{}
		for itemRows.Next() {
			var it models.OrderItem
			if err := itemRows.Scan(&it.ID, &it.ProductID, &it.Quantity, &it.TotalPrice, &it.ProductName); err != nil {
				itemRows.Close()
				return nil, err
			}
			items = append(items, &it)
		}
		itemRows.Close()
		o.Items = items

		orders = append(orders, &o)
	}

	return orders, nil
}

// ListOrders fetches a list of orders
func (r *OrderRepo) ListOrders(ctx context.Context, branchID int64) ([]*models.Order, error) {
	query := `
		SELECT 
		    o.id, o.memo_no, o.branch_id, o.order_date, o.salesperson_id, o.customer_id,
		    o.total_payable_amount, o.advance_payment_amount, o.payment_account_id,
		    o.status, o.delivery_date, o.exit_date, o.items_delivered, o.total_items, o.notes, o.created_at, o.updated_at,
			c.name, c.mobile, e.name, e.mobile
		FROM orders o
		LEFT JOIN customers c ON o.customer_id = c.id
		LEFT JOIN employees e ON o.salesperson_id = e.id
		WHERE 1=1
	`

	args := []interface{}{}
	argID := 1

	if branchID != 0 {
		query += ` AND o.branch_id=$` + strconv.Itoa(argID)
		args = append(args, branchID)
		argID++
	}

	query += ` ORDER BY o.memo_no DESC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []*models.Order{}
	for rows.Next() {
		var o models.Order
		err := rows.Scan(
			&o.ID, &o.MemoNo, &o.BranchID, &o.OrderDate, &o.SalespersonID, &o.CustomerID,
			&o.TotalPayableAmount, &o.AdvancePaymentAmount, &o.PaymentAccountID,
			&o.Status, &o.DeliveryDate, &o.ExitDate, &o.ItemsDelivered, &o.TotalItems, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
			&o.CustomerName, &o.CustomerMobile, &o.SalespersonName, &o.SalespersonMobile,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, &o)
	}

	return orders, nil
}

func (r *OrderRepo) ListOrdersPaginated(ctx context.Context, pageNo, pageLength int, status, sortByDate string, branchID int64) ([]*models.Order, error) {
	query := `
		SELECT 
		    o.id, o.memo_no, o.branch_id, o.order_date, o.salesperson_id, o.customer_id, 
		    o.total_payable_amount, o.advance_payment_amount, o.payment_account_id,
		    o.status, o.delivery_date, o.exit_date, o.notes, o.created_at, o.updated_at,
			c.name AS customer_name,
		    c.mobile AS customer_mobile,
		    e.name AS employee_name
		FROM orders o
		LEFT JOIN customers c ON o.customer_id = c.id
		LEFT JOIN employees e ON o.salesperson_id = e.id
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
	if branchID > 0 {
		query += fmt.Sprintf(" AND o.branch_id=$%d", argIdx)
		args = append(args, branchID)
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
		err := rows.Scan(
			&o.ID, &o.MemoNo, &o.BranchID, &o.OrderDate, &o.SalespersonID, &o.CustomerID,
			&o.TotalPayableAmount, &o.AdvancePaymentAmount, &o.PaymentAccountID,
			&o.Status, &o.DeliveryDate, &o.ExitDate, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
			&o.CustomerName, &o.CustomerMobile, &o.SalespersonName,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, &o)
	}

	return orders, nil
}

// ListOrdersByStatus retrieves a list of orders from the database that match the specified status.
func (r *OrderRepo) ListOrdersByStatus(ctx context.Context, status string, branchID int64) ([]*models.Order, error) {
	rows, err := r.db.Query(ctx, `
		SELECT 
		    o.id, o.memo_no, o.branch_id, o.order_date, o.salesperson_id, o.customer_id, 
		    o.total_payable_amount, o.advance_payment_amount, o.payment_account_id,
		    o.status, o.delivery_date, o.exit_date, o.notes, o.created_at, o.updated_at,
			c.name AS customer_name,
		    c.mobile AS customer_mobile,
		    e.name AS employee_name
		FROM orders o
		LEFT JOIN customers c ON o.customer_id = c.id
		LEFT JOIN employees e ON o.salesperson_id = e.id
		WHERE o.status = $1 AND o.branch_id = $2
		ORDER BY o.order_date DESC
	`, status, branchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []*models.Order{}
	for rows.Next() {
		var o models.Order

		if err := rows.Scan(
			&o.ID, &o.MemoNo, &o.BranchID, &o.OrderDate, &o.SalespersonID, &o.CustomerID,
			&o.TotalPayableAmount, &o.AdvancePaymentAmount, &o.PaymentAccountID,
			&o.Status, &o.DeliveryDate, &o.ExitDate, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
			&o.CustomerName, &o.CustomerMobile, &o.SalespersonName,
		); err != nil {
			return nil, err
		}

		orders = append(orders, &o)
	}

	return orders, nil
}

func (r *OrderRepo) GetOrderSummary(ctx context.Context, startDate, endDate string, branchID int64) (map[string]float64, error) {
	query := `
		SELECT
			COALESCE(SUM(total_payable_amount),0) AS total_amount,
			COALESCE(SUM(advance_payment_amount),0) AS total_advance_payment,
		FROM orders
		WHERE branch_id=$1 order_date BETWEEN $2 AND $3
	`

	var totalAmount, totalAdvance, totalDue float64
	err := r.db.QueryRow(ctx, query, branchID, startDate, endDate).Scan(&totalAmount, &totalAdvance)
	if err != nil {
		return nil, err
	}
	totalDue = totalAmount - totalAdvance
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
