-- SIM-Madrasah — data nyata MGMP Madrasah Al Fath (santri putra)
-- Jalankan setelah 001_init.sql. JANGAN jalankan 002_seed.sql (itu data dummy).
USE sim_madrasah;

-- User login (admin & guru). Password keduanya: admin123
INSERT INTO users (username, password_hash, nama, role) VALUES
  ('admin', '$2a$12$Jm3nXtsm.eLv9U7hU.cFT.u9MadTgDwIswKslZyijQUYhetB1EiwS', 'Administrator', 'admin'),
  ('guru',  '$2a$12$Jm3nXtsm.eLv9U7hU.cFT.u9MadTgDwIswKslZyijQUYhetB1EiwS', 'Ustadz', 'guru')
ON DUPLICATE KEY UPDATE nama = VALUES(nama);

-- Periode aktif (tahun ajaran Masehi)
INSERT INTO periode (nama, tahun_ajaran, semester, is_active) VALUES
  ('2025/2026 Ganjil', '2025/2026', 'ganjil', 1)
ON DUPLICATE KEY UPDATE is_active = VALUES(is_active);

-- Mata pelajaran
INSERT INTO mata_pelajaran (kode, nama) VALUES
  ('FQH', 'Fiqih'), ('AQD', 'Aqidah Akhlak'), ('QHD', 'Quran Hadits'),
  ('SKI', 'Sejarah Kebudayaan Islam'), ('BAR', 'Bahasa Arab'), ('MTK', 'Matematika')
ON DUPLICATE KEY UPDATE nama = VALUES(nama);

-- Kelas
INSERT INTO kelas (nama, tingkat) VALUES
  ('Sifr A', '0'), ('Sifr B', '0'),
  ('Kelas 1', '1'), ('Kelas 2', '2'), ('Kelas 3', '3'),
  ('Kelas 4', '4'), ('Kelas 5', '5'), ('Kelas 6', '6')
ON DUPLICATE KEY UPDATE tingkat = VALUES(tingkat);

-- Santri (semua laki-laki). kelas_id dipetakan dari nama kelas.
INSERT INTO santri (nis, nama, jenis_kelamin, kelas_id)
SELECT t.nis, t.nama, 'L', k.id
FROM (
  -- Kelas 1
  SELECT '2023010' AS nis, 'Ahmad Zahid Hasbullah' AS nama, 'Kelas 1' AS kelas
  UNION ALL SELECT '2025006', 'Ahmad Muhajir Assegaf', 'Kelas 1'
  UNION ALL SELECT '2025007', 'Ali Anas Mauladdawilah', 'Kelas 1'
  UNION ALL SELECT '2025008', 'Ibrahim Hafiz Al Fatih', 'Kelas 1'
  UNION ALL SELECT '2025009', 'Muhammad', 'Kelas 1'
  UNION ALL SELECT '2025010', 'Muhammad Hayiz Amal Abu Mu''adz', 'Kelas 1'
  UNION ALL SELECT '2025011', 'Muhammad Salim Abdullah', 'Kelas 1'
  UNION ALL SELECT '2025012', 'Zaidan Maulana Firdaus', 'Kelas 1'
  UNION ALL SELECT '2025013', 'Abdullah Salman Al-farisi', 'Kelas 1'
  UNION ALL SELECT '2025014', 'Ahmad Zahid Hasbullah', 'Kelas 1'
  UNION ALL SELECT '2025015', 'Javas Shiddiq Kenzie Irawan', 'Kelas 1'
  UNION ALL SELECT '2025016', 'Muhammad Achmad Mauladawilah', 'Kelas 1'
  UNION ALL SELECT '2025017', 'Muhammad Ichsan', 'Kelas 1'
  UNION ALL SELECT '2025018', 'Ahmad Hamzah BSA', 'Kelas 1'
  UNION ALL SELECT '2025019', 'Muhammad Alwi', 'Kelas 1'
  -- Kelas 2
  UNION ALL SELECT '2023008', 'Achmad Hasan Badruddin Al Imami', 'Kelas 2'
  UNION ALL SELECT '2023009', 'Al Baraa Alhamid', 'Kelas 2'
  UNION ALL SELECT '2024010', 'Muhammad Ramdhani', 'Kelas 2'
  UNION ALL SELECT '2024011', 'Muhammad Yusuf Al-Musthofa', 'Kelas 2'
  UNION ALL SELECT '2024013', 'Ahmad Mustofa', 'Kelas 2'
  UNION ALL SELECT '2024014', 'Alwi Syauqi Mauladdawilah', 'Kelas 2'
  UNION ALL SELECT '2024016', 'Muhamamd Alwi Annabhany', 'Kelas 2'
  UNION ALL SELECT '2024017', 'Muhammad Musyafa Alfarizky', 'Kelas 2'
  UNION ALL SELECT '2024018', 'Ahmad zaidan mauladawilah', 'Kelas 2'
  UNION ALL SELECT '2024019', 'Ali Anas Mauladawilah', 'Kelas 2'
  UNION ALL SELECT '2024020', 'Ibrahim Hafiz Al Fatih', 'Kelas 2'
  UNION ALL SELECT '2022001', 'Alwy Ahmad Assegaf', 'Kelas 2'
  UNION ALL SELECT '2024002', 'Haidaruddin Ahmad Dihya', 'Kelas 2'
  UNION ALL SELECT '2024005', 'Abubakar Alaydrus', 'Kelas 2'
  UNION ALL SELECT '2024008', 'Muhammad Fadhil alhabsyi', 'Kelas 2'
  UNION ALL SELECT '2023001', 'Muhammad Assegaf', 'Kelas 2'
  UNION ALL SELECT '2023002', 'Muhammad rasya atthaya', 'Kelas 2'
  UNION ALL SELECT '2023003', 'Sholeh Muhammad Zainal Abidin', 'Kelas 2'
  UNION ALL SELECT '2023005', 'Abdullah Yazid Musyaffa', 'Kelas 2'
  UNION ALL SELECT '2023006', 'Muhammad Saggaf BSA', 'Kelas 2'
  UNION ALL SELECT '2023007', 'Alwy Mauladawilah', 'Kelas 2'
  -- Kelas 3
  UNION ALL SELECT '2022002', 'Muhammad Achmad Assegaf', 'Kelas 3'
  UNION ALL SELECT '2022003', 'Hasanain Haykal Baharun', 'Kelas 3'
  UNION ALL SELECT '2022004', 'Rizky Althafaro Akbar Pahlevi', 'Kelas 3'
  UNION ALL SELECT '2022005', 'Hasan Ahmad Mauladawillah', 'Kelas 3'
  UNION ALL SELECT '2023012', 'Abdulqodir Assegaf', 'Kelas 3'
  UNION ALL SELECT '2023013', 'Kenzie Dhaniswara Apta Nararya', 'Kelas 3'
  UNION ALL SELECT '2025001', 'Muhammad Ba''agil', 'Kelas 3'
  UNION ALL SELECT '2025002', 'Muhammad bin Abdullah Mauladdawilah', 'Kelas 3'
  UNION ALL SELECT '2023015', 'M. Fakhri Zahfran Khairy', 'Kelas 3'
  UNION ALL SELECT '2020045', 'Mukhsin adni assegaf', 'Kelas 3'
  -- Kelas 4
  UNION ALL SELECT '2021057', 'M. Jalaluddin Taufiq', 'Kelas 4'
  UNION ALL SELECT '2021063', 'Abdullah Ba''abud', 'Kelas 4'
  UNION ALL SELECT '2021065', 'Abdulkadir Assegaf', 'Kelas 4'
  UNION ALL SELECT '2022006', 'Hasan Alhabsy', 'Kelas 4'
  UNION ALL SELECT '2022007', 'Muhammad Ali Ridho Al Habsyi', 'Kelas 4'
  UNION ALL SELECT '2022008', 'Muhammad Frananda Azriel', 'Kelas 4'
  UNION ALL SELECT '2022009', 'Muhammad Iqbal Arif', 'Kelas 4'
  UNION ALL SELECT '2022011', 'Abdurrahman Hasan Dzulfiqar', 'Kelas 4'
  UNION ALL SELECT '2022012', 'Faris Alfa Ansori', 'Kelas 4'
  UNION ALL SELECT '2022013', 'Moch Haidar Aly Zain', 'Kelas 4'
  UNION ALL SELECT '2022014', 'Muhammad Andrew', 'Kelas 4'
  UNION ALL SELECT '2024028', 'Mustofa Alkaf', 'Kelas 4'
  UNION ALL SELECT '2020047', 'Achmad Hasanul Khuluq', 'Kelas 4'
  UNION ALL SELECT '2024026', 'Salman Al-farizi', 'Kelas 4'
  UNION ALL SELECT '2025003', 'Valencio Rosi El Pasha', 'Kelas 4'
  UNION ALL SELECT '2025004', 'Umar Ba''agil', 'Kelas 4'
  UNION ALL SELECT '2019034', 'Haidar Ataka Anshori', 'Kelas 4'
  UNION ALL SELECT '2023025', 'Muhamad Arkan Fauzi', 'Kelas 4'
  -- Kelas 5
  UNION ALL SELECT '2022015', 'Abdul Latif', 'Kelas 5'
  UNION ALL SELECT '2022016', 'Ali Makmur', 'Kelas 5'
  UNION ALL SELECT '2022018', 'Muhammad Havid bin Agil', 'Kelas 5'
  UNION ALL SELECT '2023017', 'Umar Muchammad Mauladawilah', 'Kelas 5'
  UNION ALL SELECT '2023018', 'Mochammad Khoiruddin', 'Kelas 5'
  UNION ALL SELECT '2023019', 'Muhammad Ilham Fanani', 'Kelas 5'
  UNION ALL SELECT '2023020', 'R. Muhammad Zakky Ghufron', 'Kelas 5'
  UNION ALL SELECT '2024022', 'Muhammad Ali', 'Kelas 5'
  UNION ALL SELECT '2024029', 'Zidan Al Hamid', 'Kelas 5'
  UNION ALL SELECT '2025005', 'Abdurrahman Hasan Assegaf', 'Kelas 5'
  -- Kelas 6
  UNION ALL SELECT '2019033', 'Al Hassan Al Hamid', 'Kelas 6'
  UNION ALL SELECT '2020044', 'Salim Ridho Mulachela', 'Kelas 6'
  UNION ALL SELECT '2021066', 'Ahmad Syarifuddin', 'Kelas 6'
  UNION ALL SELECT '2021080', 'Abdul Kadir', 'Kelas 6'
  UNION ALL SELECT '2022021', 'Ahmad Alwi Al Hamid', 'Kelas 6'
  -- Sifr B
  UNION ALL SELECT '2024001', 'Muhammad Nabilunnuha', 'Sifr B'
  UNION ALL SELECT '2024003', 'Ahmad', 'Sifr B'
  UNION ALL SELECT '2024004', 'Ali', 'Sifr B'
  UNION ALL SELECT '2024006', 'Muhammad Amin Kutbi', 'Sifr B'
  UNION ALL SELECT '2024007', 'Muhammad jamal alhinduan', 'Sifr B'
  -- Sifr A
  UNION ALL SELECT '2025020', 'Achmad Bin Djafar Almasyhur', 'Sifr A'
  UNION ALL SELECT '2025021', 'Ahmad bin Muhammad', 'Sifr A'
  UNION ALL SELECT '2025022', 'Ahmad Maliki Mauladdawilah', 'Sifr A'
  UNION ALL SELECT '2025023', 'Hasan Zaki Ba''agil', 'Sifr A'
  UNION ALL SELECT '2025024', 'Muhammad Husein Al Adib', 'Sifr A'
  UNION ALL SELECT '2025025', 'Muhammad Taufiq Assegaf', 'Sifr A'
  UNION ALL SELECT '2025026', 'Ayman Abdurrahman', 'Sifr A'
) t
JOIN kelas k ON k.nama = t.kelas
ON DUPLICATE KEY UPDATE nama = VALUES(nama), kelas_id = VALUES(kelas_id);
