-- =========================
-- Table: companyprofile
-- =========================
CREATE TABLE companyprofile (
    id SERIAL PRIMARY KEY,               -- Auto-increment primary key
    name VARCHAR(255) NOT NULL,
    description TEXT,
    slogan VARCHAR(255),
    mobile VARCHAR(50),
    whatsapp VARCHAR(50),
    telephone VARCHAR(50),
    email VARCHAR(255) UNIQUE,           -- Email should be unique
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
-- employees table (with updated_at using CURRENT_TIMESTAMP)
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
