-- Skrip Reset untuk Uji Beban Supervisor (D.3)
-- Menghapus pendaftaran supervisi dan menghitung ulang supervised_count dari tabel supervisions

DELETE FROM supervisions;

UPDATE supervisors s
SET supervised_count = (SELECT COUNT(*) FROM supervisions ss WHERE ss.supervisor_id = s.id);
