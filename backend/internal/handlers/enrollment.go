package handlers

import "database/sql"

// santriRef = data ringkas santri untuk menyusun roster.
type santriRef struct {
	ID   int64
	NIS  string
	Nama string
}

// rosterSantri mengembalikan daftar santri untuk (kelas, periode):
//   - periode AKTIF → anggota kelas SAAT INI (dinamis, termasuk santri baru pindahan)
//   - periode LAMA  → enrollment beku dari santri_kelas (kelas saat periode itu berjalan).
//     Tidak ada fallback ke kelas sekarang untuk periode lama, supaya santri yang sudah
//     pindah tidak "bocor" ke kelas lain; kelas tanpa catatan enrollment → kosong.
//
// Efeknya: setelah santri naik kelas / ganti tahun ajaran, leger & rapor periode lama
// tetap menampilkan santri pada kelas & peringkat saat itu — data tidak "berpindah".
func (h *Handler) rosterSantri(kelasID, periodeID string) ([]santriRef, error) {
	var isActive bool
	_ = h.DB.QueryRow(`SELECT is_active FROM periode WHERE id = ?`, periodeID).Scan(&isActive)

	var rows *sql.Rows
	var err error
	if isActive {
		rows, err = h.DB.Query(`
			SELECT id, COALESCE(nis,''), nama
			FROM santri WHERE kelas_id = ? AND is_active = 1
			ORDER BY nama`, kelasID)
	} else {
		rows, err = h.DB.Query(`
			SELECT s.id, COALESCE(s.nis,''), s.nama
			FROM santri s JOIN santri_kelas sk ON sk.santri_id = s.id
			WHERE sk.kelas_id = ? AND sk.periode_id = ?
			ORDER BY s.nama`, kelasID, periodeID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []santriRef{}
	for rows.Next() {
		var s santriRef
		_ = rows.Scan(&s.ID, &s.NIS, &s.Nama)
		out = append(out, s)
	}
	return out, nil
}

// enrollSantriTx mencatat kelas santri untuk sebuah periode. Dipanggil saat menyimpan
// nilai/tugas, sehingga kelas historis "membeku" sesuai kelas ketika dinilai.
func enrollSantriTx(tx *sql.Tx, santriID, kelasID, periodeID int64) error {
	_, err := tx.Exec(`
		INSERT INTO santri_kelas (santri_id, periode_id, kelas_id) VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE kelas_id = VALUES(kelas_id)`,
		santriID, periodeID, kelasID)
	return err
}
