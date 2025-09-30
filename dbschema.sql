-- =========================
-- Table: companyprofile
-- =========================
CREATE TABLE companyprofile (
    id SERIAL PRIMARY KEY,               
    name VARCHAR(255) NOT NULL,
    description TEXT,
    slogan VARCHAR(255),
    mobile VARCHAR(50),
    whatsapp VARCHAR(50),
    telephone VARCHAR(50),
    email VARCHAR(255) UNIQUE,
    website VARCHAR(255),
    country VARCHAR(100),
    city VARCHAR(100),
    postal_code VARCHAR(20),
    logo_link TEXT,
    opening_date DATE,
    terms_conditions TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for quick search on name and email
CREATE INDEX idx_company_name ON companyprofile(name);


-- =========================
-- Table: employees
-- =========================
CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    fname VARCHAR(100) NOT NULL,
    lname VARCHAR(100) NOT NULL,
    role VARCHAR(50) NOT NULL,
    status VARCHAR(50),
    bio TEXT,
    email VARCHAR(255) UNIQUE NOT NULL,
    password TEXT NOT NULL,
    mobile VARCHAR(50),
    country VARCHAR(100),
    city VARCHAR(100),
    address VARCHAR(100),
    postal_code VARCHAR(20),
    tax_id VARCHAR(50),
    base_salary NUMERIC(12,2),
    overtime_rate NUMERIC(10,2),
    avatar_link TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


-- Indexes for quick search on name, email, role
CREATE INDEX idx_employee_name ON employees(fname, lname);
CREATE INDEX idx_employee_email ON employees(email);
CREATE INDEX idx_employee_mobile ON employees(mobile);
CREATE INDEX idx_employee_role ON employees(role);

-- =========================
-- Table: attendance
-- =========================
CREATE TABLE attendance (
    id SERIAL PRIMARY KEY,
    employee_id INT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    work_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'Present' CHECK (status IN ('Present', 'Absent', 'Leave')),
    check_in TIMESTAMP NULL,
    check_out TIMESTAMP NULL,
    overtime_hours INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(employee_id, work_date)
);

CREATE INDEX idx_attendance_employee_date ON attendance(employee_id, work_date);
CREATE INDEX idx_attendance_work_date ON attendance(work_date);
CREATE INDEX idx_attendance_status ON attendance(status)

-- =========================
-- Table: customers
-- =========================
CREATE TABLE customers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    mobile VARCHAR(50) UNIQUE,
    address VARCHAR(200) DEFAULT '',
    tax_id VARCHAR(50) UNIQUE,
    due_amount NUMERIC(12,2) NOT NULL DEFAULT 0.00,
    status BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_customers_name ON customers(name);

CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE, -- e.g., Cash, Bank, bKash, Rocket, etc.
    type VARCHAR(50) NOT NULL CHECK (type IN ('cash', 'bank', 'mfs', 'other')),
    current_balance NUMERIC(12,2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- =========================
-- Table: Category
-- =========================
CREATE TABLE public.categories (
    id BIGSERIAL PRIMARY KEY,
    name character varying(255) DEFAULT ''::character varying NOT NULL,
    status boolean DEFAULT true NOT NULL,
   created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- =========================
-- Table: Brand
-- =========================
CREATE TABLE public.brands (
    id BIGSERIAL PRIMARY KEY,
    name character varying(255) DEFAULT ''::character varying NOT NULL,
    status boolean DEFAULT true NOT NULL,
   created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- =========================
-- Table: Product
-- =========================
CREATE TABLE public.products (
    id BIGSERIAL PRIMARY KEY,
    product_code character varying(255) DEFAULT ''::character varying NOT NULL,
    product_name character varying(255) DEFAULT ''::character varying NOT NULL,
    product_description character varying(255) DEFAULT ''::character varying NOT NULL,
    product_status boolean DEFAULT true NOT NULL,
    mrp bigint DEFAULT 0 NOT NULL,
    warranty integer DEFAULT 0 NOT NULL,
    category_id bigint,
    brand_id bigint ,
    stock_alert_level smallint DEFAULT 0 NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);
-- =========================
-- Table: orders
-- =========================
CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY,
    memo_no VARCHAR(100) NOT NULL UNIQUE,
    order_date DATE NOT NULL DEFAULT CURRENT_DATE,
    
    sales_man_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE RESTRICT,
    customer_id BIGINT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    
    total_payable_amount NUMERIC(12,2) NOT NULL,
    advance_payment_amount NUMERIC(12,2) DEFAULT 0,
    due_amount NUMERIC(12,2) GENERATED ALWAYS AS (total_payable_amount - advance_payment_amount) STORED,
    
    payment_account_id BIGINT REFERENCES accounts(id) ON DELETE SET NULL, -- 🔹 Added
    
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'checkout', 'delivery', 'cancelled')),
    
    delivery_date DATE,
    delivered_by BIGINT REFERENCES employees(id) ON DELETE SET NULL,
    
    notes TEXT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_orders_date_status ON orders(order_date, status);
CREATE INDEX idx_orders_date ON orders(order_date);
-- Order items for handling multiple products per order
CREATE TABLE order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity INT NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(12,2) NOT NULL,
);
CREATE TABLE transactions (
    transaction_id BIGSERIAL PRIMARY KEY,
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


