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
CREATE INDEX idx_employees_role ON employees(role);
CREATE INDEX idx_employees_branch_id ON employees(branch_id);

-- =========================
-- Table: attendance
-- =========================
CREATE TABLE attendance (
    id SERIAL PRIMARY KEY,
    employee_id INT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    work_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'Present' CHECK (status IN ('Present', 'Absent', 'Leave')),
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


