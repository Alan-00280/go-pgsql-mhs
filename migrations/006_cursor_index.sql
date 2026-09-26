-- index cursor desc `users` by created at, id
CREATE INDEX IF NOT EXISTS users_created_at_id_desc_idx
    ON users (created_at DESC, id DESC);

-- index cursor desc `students` by created at, id
CREATE INDEX IF NOT EXISTS users_created_at_id_desc_idx
    ON students (created_at DESC, id DESC);