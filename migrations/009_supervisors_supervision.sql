-- Migration 009: Supervisors and Supervision Student

CREATE TABLE IF NOT EXISTS supervisors (
    id SERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    quota INTEGER NOT NULL CHECK (quota > 0),
    supervised_count INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Jaring pengaman terakhir. Bila kode aplikasi benar, constraint ini
    -- TIDAK PERNAH terpicu. Bila suatu hari terpicu, artinya ada bug di
    -- kode kita — dan database menolak menyimpan keadaan yang mustahil.
    CONSTRAINT supervisor_supervised_within_quota
        CHECK (supervised_count >= 0 AND supervised_count <= quota)
);

CREATE TABLE IF NOT EXISTS supervisions (
    id BIGSERIAL PRIMARY KEY,
    student_id INTEGER NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    supervisor_id INTEGER NOT NULL REFERENCES supervisors(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Satu mahasiswa hanya boleh terdaftar satu kali di supervisor yang sama.
    CONSTRAINT supervision_supervisor_student_unique UNIQUE (supervisor_id, student_id)
);

CREATE INDEX IF NOT EXISTS supervisions_student_idx
    ON supervisions (student_id);

INSERT INTO supervisors (name, quota) VALUES
    ('Dr. Budi Santoso, M.Kom.',      10),
    ('Prof. Siti Rahmawati, Ph.D.',    8),
    ('Dr. Agus Prasetyo, M.T.',        6),
    ('Dr. Dewi Lestari, M.Sc.',       12),
    ('Ir. Hendra Wijaya, M.Eng.',      5),
    ('Dr. Rina Kusumawati, M.Kom.',   10),
    ('Prof. Eko Nugroho, Ph.D.',       4),
    ('Dr. Maya Anggraini, M.T.',       7),
    ('Dr. Fajar Setiawan, M.Sc.',      3),
    ('Dr. Lina Marlina, M.Kom.',       9);

INSERT INTO permissions (name, description)
VALUES (
    'supervisions:manage',
    'Mengubah kuota dan data supervision'
)
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name)
VALUES ('admin', 'supervisions:manage')
ON CONFLICT DO NOTHING;
