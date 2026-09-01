-- Create students table
CREATE TABLE IF NOT EXISTS students (
    id SERIAL PRIMARY KEY,
    nim CHAR(9) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    grade NUMERIC(3, 2) NOT NULL CHECK (grade >= 0 AND grade <= 4.00),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index on nim for faster lookups (UNIQUE constraint already indexes, but explicit for clarity)
CREATE INDEX IF NOT EXISTS idx_students_nim ON students(nim);

-- Create index on is_active for filtering active/inactive students
CREATE INDEX IF NOT EXISTS idx_students_is_active ON students(is_active);

-- Create composite index on is_active and created_at for common queries
CREATE INDEX IF NOT EXISTS idx_students_is_active_created_at ON students(is_active, created_at DESC);