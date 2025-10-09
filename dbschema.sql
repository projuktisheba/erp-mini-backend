-- =========================
-- Table: branches
-- =========================
CREATE TABLE branches (
    id BIGSERIAL PRIMARY KEY,               
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT NOT null DEFAULT '',
    slogan VARCHAR(255) NOT null DEFAULT '',
    mobile VARCHAR(50) NOT null DEFAULT '',
    telephone VARCHAR(50) NOT null DEFAULT '',
    email VARCHAR(255) NOT null DEFAULT '',
    website VARCHAR(255) NOT null DEFAULT '',
    country VARCHAR(100) NOT null DEFAULT '',
    city VARCHAR(100) NOT null DEFAULT '',
    address VARCHAR (1000) NOT null DEFAULT '',
    postal_code VARCHAR(20) NOT null DEFAULT '',
    logo_link TEXT NOT null DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- =========================
-- Insert initial branches
-- =========================
INSERT INTO branches (
    name, description, slogan, mobile, telephone, email, website, country, city, address, postal_code, logo_link
) VALUES
('AL FANAR ABAYAT', 'Dummy description', 'Dummy slogan', '0000000000', '0000000000', 'dummy1@example.com', 'http://dummy1.com', 'Qatar', 'Doha', '123 Dummy Street', '00000', 'http://dummy1.com/logo.png'),
('DIVA ABAYAT', 'Dummy description', 'Dummy slogan', '0000000001', '0000000001', 'dummy2@example.com', 'http://dummy2.com', 'Qatar', 'Doha', '456 Dummy Street', '00001', 'http://dummy2.com/logo.png'),
('EID AL ABAYAT', 'Dummy description', 'Dummy slogan', '0000000002', '0000000002', 'dummy3@example.com', 'http://dummy3.com', 'Qatar', 'Doha', '789 Dummy Street', '00002', 'http://dummy3.com/logo.png');

-- =========================
-- Table: employees
-- =========================
-- Create employees table
CREATE TABLE employees (
    id BIGSERIAL PRIMARY KEY,  
    name VARCHAR(100) NOT NULL,  
    role VARCHAR(20) NOT NULL CHECK (role IN ('chairman', 'manager', 'salesperson', 'worker')),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK(status IN('active', 'inactive')),  
    mobile VARCHAR(20) NOT NULL UNIQUE,  
    email VARCHAR(150)NOT NULL DEFAULT '',
    password TEXT NOT NULL DEFAULT '',
    passport_no VARCHAR(50)NOT NULL DEFAULT '',  
    joining_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,  
    address VARCHAR(1000)NOT NULL DEFAULT '',  
    base_salary NUMERIC(12,2) DEFAULT 0,  
    overtime_rate NUMERIC(12,2) DEFAULT 0,
    avatar_link TEXT DEFAULT '',
    branch_id BIGINT NOT NULL REFERENCES branches(id) ON DELETE CASCADE,  
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,  
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP  
);

-- Indexes 
CREATE INDEX idx_employees_name ON employees(name);
CREATE INDEX idx_employees_mobile ON employees(mobile);
CREATE INDEX idx_employees_role ON employees(role);
CREATE INDEX idx_employees_branch_id ON employees(branch_id);

-- =========================
-- Table: attendance
-- =========================
CREATE TABLE attendance (
    id SERIAL PRIMARY KEY,
    employee_id INT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    work_date DATE NOT NULL,
    branch_id BIGINT NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'Present' CHECK (status IN ('Present', 'Absent', 'Leave')),
    advance_payment BIGINT DEFAULT 0,
    production_units BIGINT DEFAULT 0,
    overtime_hours BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(employee_id, work_date)
);

CREATE INDEX idx_attendance_employee_date ON attendance(employee_id, work_date);
CREATE INDEX idx_attendance_work_date ON attendance(work_date);
CREATE INDEX idx_attendance_status ON attendance(status);

-- =========================
-- Table: customers
-- =========================
CREATE TABLE customers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    mobile VARCHAR(20) NOT NULL UNIQUE,
    address TEXT NOT NULL DEFAULT '',
    tax_id VARCHAR(100) DEFAULT '',
    due_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
    status BOOLEAN NOT NULL DEFAULT TRUE,
    branch_id BIGINT NOT NULL REFERENCES branches(id) ON DELETE CASCADE,  

    -- Measurement fields (kept as text/varchar since no calculation needed)
    length VARCHAR(50) DEFAULT '',
    shoulder VARCHAR(50) DEFAULT '',
    bust VARCHAR(50) DEFAULT '',
    waist VARCHAR(50) DEFAULT '',
    hip VARCHAR(50) DEFAULT '',
    arm_hole VARCHAR(50) DEFAULT '',
    sleeve_length VARCHAR(50) DEFAULT '',
    sleeve_width VARCHAR(50) DEFAULT '',
    round_width VARCHAR(50) DEFAULT '',
    

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index for faster name lookups (useful for search/autocomplete)
CREATE INDEX idx_customers_name ON customers(name);
CREATE INDEX idx_customers_mobile ON customers(name);


CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL , -- e.g., Cash, Bank, ATM etc.
    type VARCHAR(50) NOT NULL CHECK (type IN ('cash', 'bank', 'mfs', 'other')),
    current_balance NUMERIC(12,2) NOT NULL DEFAULT 0.00,
    branch_id BIGINT NOT NULL REFERENCES branches(id) ON DELETE CASCADE,  
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO public.accounts (name, type, branch_id) 
VALUES 
  ('Cash', 'cash', 1),
  ('Bank', 'bank', 1),
  ('Cash', 'cash', 2),
  ('Bank', 'bank', 2),
  ('Cash', 'cash', 3),
  ('Bank', 'bank', 3);
-- =========================
-- Table: Product
-- =========================
CREATE TABLE public.products (
    id BIGSERIAL PRIMARY KEY,
    product_name character varying(255) DEFAULT ''::character varying NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO public.products (product_name) 
VALUES 
  ('Abayat Shela (L)'),
  ('Abayat Shela (M)'),
  ('Abayat Shela (S)'),
  ('Abayat Raj'),
  ('Khamar');
-- =========================
-- Table: orders
-- =========================
CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY,
    branch_id BIGINT NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    memo_no VARCHAR(100) NOT NULL UNIQUE,
    order_date DATE NOT NULL DEFAULT CURRENT_DATE,
    
    salesperson_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE RESTRICT,
    customer_id BIGINT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    
    total_payable_amount NUMERIC(12,2) NOT NULL,
    advance_payment_amount NUMERIC(12,2) DEFAULT 0,

    payment_account_id BIGINT REFERENCES accounts(id) ON DELETE SET NULL,
    
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'checkout', 'delivery', 'cancelled')),
    
    delivery_date DATE,
    exit_date DATE,
    
    notes TEXT DEFAULT '',
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- =========================
-- Optimized Indexes for orders table
-- =========================

-- Single-column indexes
CREATE INDEX IF NOT EXISTS idx_orders_salesperson_id
ON orders(salesperson_id);

CREATE INDEX IF NOT EXISTS idx_orders_customer_id
ON orders(customer_id);

CREATE INDEX IF NOT EXISTS idx_orders_payment_account_id
ON orders(payment_account_id);

CREATE INDEX IF NOT EXISTS idx_orders_status
ON orders(status);

CREATE INDEX IF NOT EXISTS idx_orders_branch_id
ON orders(branch_id);

-- Composite indexes for common multi-column filters
-- a) Salesperson + Customer + Status + order_date DESC (for dashboard queries)
CREATE INDEX IF NOT EXISTS idx_orders_salesperson_customer_status
ON orders(salesperson_id, customer_id, status, order_date DESC);

-- b) Salesperson + Status + order_date DESC
CREATE INDEX IF NOT EXISTS idx_orders_salesperson_status
ON orders(salesperson_id, status, order_date DESC);

-- c) Customer + Status + order_date DESC
CREATE INDEX IF NOT EXISTS idx_orders_customer_status
ON orders(customer_id, status, order_date DESC);

-- Partial indexes for frequently queried statuses
CREATE INDEX IF NOT EXISTS idx_orders_pending
ON orders(order_date DESC)
WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_orders_checkout
ON orders(order_date DESC)
WHERE status = 'checkout';

CREATE INDEX IF NOT EXISTS idx_orders_delivery
ON orders(order_date DESC)
WHERE status = 'delivery';

CREATE INDEX IF NOT EXISTS idx_orders_cancelled
ON orders(order_date DESC)
WHERE status = 'cancelled';


-- Order items for handling multiple products per order
CREATE TABLE order_items (
    id BIGSERIAL PRIMARY KEY,
    memo_no VARCHAR(100) NOT NULL REFERENCES orders(memo_no) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity INT NOT NULL CHECK (quantity > 0),
    subtotal NUMERIC(12,2) NOT NULL
);
CREATE TABLE transactions (
    transaction_id BIGSERIAL PRIMARY KEY,
    branch_id BIGINT NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    from_entity_id BIGINT NOT NULL,
    from_entity_type VARCHAR(50),  -- optional, can be 'account', 'customer', 'employee', etc.
    
    to_entity_id BIGINT NOT NULL,
    to_entity_type VARCHAR(50),    -- optional
    
    amount NUMERIC(12,2) NOT NULL,
    transaction_type VARCHAR(20) NOT NULL 
        CHECK (transaction_type IN ('payment', 'refund', 'adjustment', 'salary')),
    
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- Indexes for quick search by purchase_date, memo_no, supplier_id
CREATE INDEX idx_transactions_created_date ON transactions(created_at);
CREATE INDEX idx_transactions_type ON transactions(transaction_type);
CREATE INDEX idx_transactions_from_entity_id ON transactions(from_entity_id);
CREATE INDEX idx_transactions_from_entity_type ON transactions(from_entity_type);
CREATE INDEX idx_transactions_to_entity_id ON transactions(to_entity_id);
CREATE INDEX idx_transactions_to_entity_type ON transactions(to_entity_type);
-- =========================
-- Table: suppliers
-- =========================
CREATE TABLE suppliers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    mobile VARCHAR(20) NOT NULL UNIQUE,
    branch_id BIGINT NOT NULL REFERENCES branches(id) ON DELETE CASCADE,  
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index for quick search by name, mobile and status
CREATE INDEX idx_suppliers_name_status ON suppliers(name, status);
CREATE INDEX idx_suppliers_mobile ON suppliers(mobile);
CREATE INDEX idx_suppliers_branch_id ON suppliers(branch_id);


-- =========================
-- Table: purchase
-- =========================
CREATE TABLE purchase (
    id BIGSERIAL PRIMARY KEY,
    memo_no VARCHAR(100) NOT NULL DEFAULT '',
    purchase_date DATE NOT NULL DEFAULT CURRENT_DATE,
    supplier_id BIGINT NOT NULL REFERENCES suppliers(id) ON DELETE RESTRICT,
    branch_id BIGINT NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    total_amount NUMERIC(12,2) NOT NULL,
    notes TEXT DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for quick search by purchase_date, memo_no, supplier_id
CREATE INDEX idx_purchase_date ON purchase(purchase_date);
CREATE INDEX idx_purchase_memo_no ON purchase(memo_no);
CREATE INDEX idx_purchase_supplier_id ON purchase(supplier_id);

-- top_sheet holds the daily balances
CREATE TABLE top_sheet (
    id BIGSERIAL PRIMARY KEY,
    sheet_date DATE NOT NULL DEFAULT CURRENT_DATE,
    branch_id BIGINT NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    expense NUMERIC(12,2) NOT NULL,
    cash NUMERIC(12,2) NOT NULL,
    bank NUMERIC(12,2) NOT NULL,
    order_count BIGINT NOT NULL,
    delivery BIGINT NOT NULL,
    checkout BIGINT NOT NULL,
    UNIQUE(sheet_date, branch_id)
);
CREATE INDEX idx_top_sheet_sheet_date ON top_sheet(sheet_date, branch_id);