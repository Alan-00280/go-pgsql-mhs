-- tabel Roles
CREATE TABLE IF NOT EXISTS roles (
    name    VARCHAR(20) PRIMARY KEY,
    description VARCHAR(150) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- isi tabel roles
INSERT INTO roles (name, description) VALUES 
    ('admin', 'Akses penuh terhadap seluruh data dan pengaturan'),
    ('staff', 'Boleh melihat data seluruh user, tetapi tidak boleh mengubah'),
    ('user',  'Hanya boleh mengelola datanya sendiri')
ON CONFLICT (name) DO NOTHING;

-- tabel permissions
CREATE TABLE IF NOT EXISTS permissions (
    name VARCHAR(50) PRIMARY KEY,
    description VARCHAR(150) NOT NULL
);
-- isi table permissions
INSERT INTO permissions (name, description) VALUES
    ('user:list', 'Mendaftar semua pengguna'),
    ('user:read:any', 'Mengambil data pengguna manapun'),
    ('user:update:any', 'Mengubah data pengguna manapun'),
    ('user:delete', 'Menghapus pengguna'),
    ('role:assign', 'Mengubah role milik user lain')
ON CONFLICT (name) DO NOTHING;

-- table role_permissions
CREATE TABLE IF NOT EXISTS role_permissions (
    role_name VARCHAR(20) NOT NULL REFERENCES roles(name) ON DELETE CASCADE,
    permission_name VARCHAR(50) NOT NULL REFERENCES permissions(name) ON DELETE CASCADE,
    PRIMARY KEY (role_name, permission_name)
);
-- isi role_permissions
INSERT INTO role_permissions (role_name, permission_name) VALUES 
    ('admin', 'user:list'),
    ('admin', 'user:read:any'),
    ('admin', 'user:update:any'),
    ('admin', 'user:delete'),
    ('admin', 'role:assign'),
    ('staff', 'user:list'),
    ('staff', 'user:read:any')
ON CONFLICT DO NOTHING;

-- set roles to user defaultly
UPDATE users SET role = 'user' WHERE role NOT IN (SELECT name FROM roles);

-- update FK
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_fkey;
ALTER TABLE users
    ADD CONSTRAINT users_role_fkey
    FOREIGN KEY (role) REFERENCES roles(name) ON UPDATE CASCADE;

-- creae user role index
CREATE INDEX IF NOT EXISTS users_role_idx ON users (role);


