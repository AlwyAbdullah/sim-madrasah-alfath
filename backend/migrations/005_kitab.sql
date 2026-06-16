-- SIM-Madrasah — kolom nama kitab pada mata pelajaran (untuk rapor)
USE sim_madrasah;
ALTER TABLE mata_pelajaran ADD COLUMN kitab VARCHAR(120) NULL AFTER nama;
