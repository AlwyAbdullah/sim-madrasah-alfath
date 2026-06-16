package handlers

import (
	"net/http"
	"strconv"

	"sim-madrasah/backend/internal/httpx"
)

// GET /rapor?santri_id=&periode_id=
// Data lengkap untuk rapor satu santri: nilai+kitab+rata kelas, jumlah/rata/peringkat, kehadiran.
func (h *Handler) RaporData(w http.ResponseWriter, r *http.Request) {
	santriID := r.URL.Query().Get("santri_id")
	periodeID := r.URL.Query().Get("periode_id")
	if santriID == "" || periodeID == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "santri_id dan periode_id wajib")
		return
	}

	// identitas + kelas
	var nis, nama, jk, kelasNama string
	var kelasID int64
	err := h.DB.QueryRow(`
		SELECT COALESCE(s.nis,''), s.nama, s.jenis_kelamin, s.kelas_id, k.nama
		FROM santri s JOIN kelas k ON k.id = s.kelas_id WHERE s.id = ?`, santriID).
		Scan(&nis, &nama, &jk, &kelasID, &kelasNama)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "Santri tidak ditemukan")
		return
	}

	// periode
	var pNama, pTahun, pSemester string
	_ = h.DB.QueryRow(`SELECT nama, tahun_ajaran, semester FROM periode WHERE id = ?`, periodeID).
		Scan(&pNama, &pTahun, &pSemester)

	// leger kelas (untuk rata kelas per mapel + peringkat santri)
	mapels, rows, err := h.buildLeger(strconv.FormatInt(kelasID, 10), periodeID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	// kitab per mapel
	kitab := map[int64]string{}
	krows, _ := h.DB.Query(`SELECT id, COALESCE(kitab,'') FROM mata_pelajaran`)
	if krows != nil {
		for krows.Next() {
			var id int64
			var k string
			_ = krows.Scan(&id, &k)
			kitab[id] = k
		}
		krows.Close()
	}

	// baris santri ini
	var me *legerRow
	for i := range rows {
		if strconv.FormatInt(rows[i].SantriID, 10) == santriID {
			me = &rows[i]
			break
		}
	}

	type nilaiRapor struct {
		Mapel      string   `json:"mata_pelajaran"`
		Kitab      string   `json:"kitab"`
		NilaiAkhir *float64 `json:"nilai_akhir"`
		RataKelas  *float64 `json:"rata_kelas"`
	}
	nilaiList := []nilaiRapor{}
	var jumlah float64
	var adaNilai bool

	for _, m := range mapels {
		if me == nil {
			break
		}
		val := me.Nilai[m.ID]
		if val == nil {
			continue // hanya mapel yang santri ini punya nilainya
		}
		// rata kelas mapel ini
		var sum float64
		var cnt int
		for i := range rows {
			if v := rows[i].Nilai[m.ID]; v != nil {
				sum += *v
				cnt++
			}
		}
		var rata *float64
		if cnt > 0 {
			rk := float64(int(sum/float64(cnt)*100+0.5)) / 100
			rata = &rk
		}
		nilaiList = append(nilaiList, nilaiRapor{
			Mapel: m.Nama, Kitab: kitab[m.ID], NilaiAkhir: val, RataKelas: rata,
		})
		jumlah += *val
		adaNilai = true
	}

	var rataRata *float64
	var peringkat int
	if me != nil {
		rataRata = me.RataRata
		peringkat = me.Peringkat
	}
	var jumlahPtr *float64
	if adaNilai {
		j := float64(int(jumlah*100+0.5)) / 100
		jumlahPtr = &j
	}

	// rekap kehadiran
	rekap := map[string]int{"hadir": 0, "izin": 0, "sakit": 0, "alpha": 0}
	arows, _ := h.DB.Query(`SELECT status, COUNT(*) FROM absensi WHERE santri_id = ? GROUP BY status`, santriID)
	if arows != nil {
		for arows.Next() {
			var st string
			var c int
			_ = arows.Scan(&st, &c)
			rekap[st] = c
		}
		arows.Close()
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"santri":   map[string]interface{}{"nis": nis, "nama": nama, "jenis_kelamin": jk, "kelas": kelasNama},
		"periode":  map[string]interface{}{"nama": pNama, "tahun_ajaran": pTahun, "semester": pSemester},
		"nilai":    nilaiList,
		"jumlah":   jumlahPtr,
		"rata":     rataRata,
		"peringkat": peringkat,
		"kehadiran": rekap,
	})
}
