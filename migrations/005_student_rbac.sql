-- isi table permissions
INSERT INTO permissions (name, description) VALUES
    ('student:list', 'Mendaftar semua mahasiswa'),
    ('student:read:any', 'Mengambil data mahasiswa manapun'),
    ('student:create', 'Membuat data mahasiswa'),
    ('student:update:any', 'Mengubah data mahasiswa manapun'),
    ('student:delete', 'Menghapus mahasiswa')
ON CONFLICT (name) DO NOTHING;

-- isi role_permissions
INSERT INTO role_permissions (role_name, permission_name) VALUES 
    ('admin', 'student:list'),
    ('admin', 'student:read:any'),
    ('admin', 'student:create'),
    ('admin', 'student:update:any'),
    ('admin', 'student:delete'),
    ('staff', 'student:list'),
    ('staff', 'student:read:any'),
    ('staff', 'student:create')
ON CONFLICT DO NOTHING;

-- kolom owner ID 
ALTER TABLE students
    ADD COLUMN IF NOT EXISTS owner_id INTEGER;

-- kasih value ke kolom baru di student
UPDATE students SET owner_id = 1 WHERE owner_id IS NULL;

-- kasih constraint
ALTER TABLE students DROP CONSTRAINT IF EXISTS students_owner_fk;
ALTER TABLE students
    ADD CONSTRAINT students_owner_fk
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE SET NULL;

-- mungkin paling idealnya untuk handle ON DELETE pada FK owner_id:
--   buat satu staff bernama null_staff
--   staff ini bersifat menjadi nilai default owner_id
--   jadi ketika dihapus bisa ON DELETE SET DEFAULT
--   Namun, ini tidak bisa diimplementasikan di beberapa engine seperti MySQL / MariaDB (InnoDB)

