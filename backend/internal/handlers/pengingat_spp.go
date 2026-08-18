package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sim-madrasah/backend/internal/httpx"
)

// JenisPengingatSPP menandai baris antrean yang berasal dari pengingat SPP bulanan.
const JenisPengingatSPP = "pengingat_spp"

// batasPesanTelegram: Telegram menolak pesan di atas 4096 karakter, dan kolom
// notifikasi.pesan dibatasi 4000. Ambil 3500 sebagai ambang aman — sisanya jadi
// ruang untuk penutup pesan.
const batasPesanTelegram = 3500

// TunggakanKelas = daftar santri satu kelas yang belum melunasi SPP bulan tertentu.
type TunggakanKelas struct {
	KelasID int64    `json:"kelas_id"`
	Kelas   string   `json:"kelas"`
	Total   int      `json:"total"` // seluruh santri aktif di kelas ini
	Belum   []string `json:"belum"` // nama santri yang belum lunas
}

// RekapTunggakanSPP = kondisi pembayaran SPP satu bulan untuk seluruh madrasah.
type RekapTunggakanSPP struct {
	Tahun       int              `json:"tahun"` // tahun kalender (bukan tahun ajaran)
	Bulan       int              `json:"bulan"` // 1..12
	NamaBulan   string           `json:"nama_bulan"`
	TotalSantri int              `json:"total_santri"`
	Lunas       int              `json:"lunas"`
	Belum       int              `json:"belum"`
	Kelas       []TunggakanKelas `json:"kelas"`
}

// HitungTunggakanSPP mencari santri aktif yang belum melunasi SPP pada satu bulan.
//
// Memakai LEFT JOIN, bukan filter `lunas = 0`: santri yang belum punya baris SPP
// sama sekali juga terhitung belum bayar. Ini yang membedakannya dari sekadar
// membaca tabel spp — di produksi banyak santri memang belum punya barisnya.
//
// `k.aktif = 1` WAJIB ada: kelas "Alumni" dan "Waqof / Berhenti" berisi santri yang
// baris santri-nya masih is_active = 1, padahal mereka sudah tidak ditagih SPP.
// Tanpa filter ini mereka ikut masuk daftar tunggakan. Konvensi sama dengan
// StatusAbsensiHarian.
func HitungTunggakanSPP(db *sql.DB, tahun, bulan int) (RekapTunggakanSPP, error) {
	out := RekapTunggakanSPP{Tahun: tahun, Bulan: bulan}
	if bulan >= 1 && bulan <= 12 {
		out.NamaBulan = bulanIndo[bulan-1]
	}

	rows, err := db.Query(`
		SELECT k.id, k.nama, s.nama, COALESCE(sp.lunas, 0) AS lunas
		FROM santri s
		JOIN kelas k ON k.id = s.kelas_id
		LEFT JOIN spp sp ON sp.santri_id = s.id AND sp.tahun = ? AND sp.bulan = ?
		WHERE s.is_active = 1 AND k.aktif = 1
		ORDER BY k.nama, s.nama`, tahun, bulan)
	if err != nil {
		return out, err
	}
	defer rows.Close()

	idx := map[int64]int{}
	for rows.Next() {
		var kelasID int64
		var kelas, nama string
		var lunas bool
		if err := rows.Scan(&kelasID, &kelas, &nama, &lunas); err != nil {
			continue
		}
		i, ada := idx[kelasID]
		if !ada {
			i = len(out.Kelas)
			idx[kelasID] = i
			out.Kelas = append(out.Kelas, TunggakanKelas{KelasID: kelasID, Kelas: kelas, Belum: []string{}})
		}
		out.Kelas[i].Total++
		out.TotalSantri++
		if lunas {
			out.Lunas++
		} else {
			out.Kelas[i].Belum = append(out.Kelas[i].Belum, nama)
			out.Belum++
		}
	}
	if out.Kelas == nil {
		out.Kelas = []TunggakanKelas{}
	}
	return out, rows.Err()
}

// PesanTunggakanSPP menyusun teks pengingat.
//
// Daftar nama adalah bagian yang paling berguna, jadi dicoba dulu selengkapnya.
// Bila melewati batas satu pesan Telegram, disusun ulang tanpa nama (hanya jumlah
// per kelas) — lebih baik ringkas tapi utuh daripada terpotong di tengah nama.
func PesanTunggakanSPP(r RekapTunggakanSPP) string {
	pesan := susunTunggakan(r, true)
	if len(pesan) > batasPesanTelegram {
		pesan = susunTunggakan(r, false)
	}
	return pesan
}

func susunTunggakan(r RekapTunggakanSPP, denganNama bool) string {
	var b strings.Builder
	b.WriteString("Assalamu'alaikum warahmatullahi wabarakatuh.\n\n")
	b.WriteString(fmt.Sprintf("Pengingat *SPP %s %d*.\n\n", r.NamaBulan, r.Tahun))
	b.WriteString(fmt.Sprintf("Belum membayar: *%d dari %d santri* (sudah lunas %d).\n",
		r.Belum, r.TotalSantri, r.Lunas))

	for _, k := range r.Kelas {
		if len(k.Belum) == 0 {
			continue
		}
		b.WriteString(fmt.Sprintf("\n*%s* — %d dari %d belum bayar\n", k.Kelas, len(k.Belum), k.Total))
		if denganNama {
			for _, nama := range k.Belum {
				b.WriteString("• " + nama + "\n")
			}
		}
	}

	if !denganNama {
		b.WriteString("\n_Daftar nama terlalu panjang untuk satu pesan — buka halaman SPP untuk rinciannya._\n")
	}
	b.WriteString("\nMohon ditindaklanjuti. Jazakumullahu khairan.\n\n_SIM Madrasah Al Fath_")
	return b.String()
}

// AntreNotifikasi memasukkan satu pesan ke antrean pengiriman. Worker Telegram
// yang mengambilnya pada polling berikutnya.
func AntreNotifikasi(db *sql.DB, jenis, refTanggal, tujuan, pesan string) error {
	_, err := db.Exec(`
		INSERT INTO notifikasi (santri_id, jenis, ref_tanggal, tujuan, pesan, status)
		VALUES (NULL, ?, ?, ?, ?, 'pending')`, jenis, refTanggal, tujuan, pesan)
	return err
}

// ===================== PENGATURAN =====================

// PengaturanPengingatSPP — jadwal pengingat bulanan.
type PengaturanPengingatSPP struct {
	Aktif   bool `json:"aktif"`
	Tanggal int  `json:"tanggal"` // tanggal berapa tiap bulan (1..28)
	Jam     int  `json:"jam"`
	Menit   int  `json:"menit"`
}

// BacaPengaturanPengingatSPP mengembalikan jadwal; bila tabel/baris belum ada
// (migrasi tertinggal) dipakai nilai bawaan yang sama dengan migrasi 019.
func BacaPengaturanPengingatSPP(db *sql.DB) PengaturanPengingatSPP {
	p := PengaturanPengingatSPP{Aktif: true, Tanggal: 10, Jam: 19, Menit: 0}
	_ = db.QueryRow(`SELECT aktif, tanggal, jam, menit FROM pengingat_spp_pengaturan WHERE id = 1`).
		Scan(&p.Aktif, &p.Tanggal, &p.Jam, &p.Menit)
	return p
}

// bulanDiminta membaca ?tahun= & ?bulan= dari query, bawaan = bulan berjalan.
func bulanDiminta(r *http.Request) (int, int) {
	now := time.Now()
	tahun, bulan := now.Year(), int(now.Month())
	if v := r.URL.Query().Get("tahun"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 2000 && n <= 2100 {
			tahun = n
		}
	}
	if v := r.URL.Query().Get("bulan"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 12 {
			bulan = n
		}
	}
	return tahun, bulan
}

// GET /pengingat-spp?tahun=&bulan= — jadwal + rekap tunggakan bulan tersebut.
func (h *Handler) GetPengingatSPP(w http.ResponseWriter, r *http.Request) {
	tahun, bulan := bulanDiminta(r)
	rekap, err := HitungTunggakanSPP(h.DB, tahun, bulan)
	if err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"pengaturan": BacaPengaturanPengingatSPP(h.DB),
		"rekap":      rekap,
	})
}

// POST /pengingat-spp — ubah nyala/mati, tanggal, dan jam pengingat.
func (h *Handler) SetPengingatSPP(w http.ResponseWriter, r *http.Request) {
	var req PengaturanPengingatSPP
	if !decode(w, r, &req) {
		return
	}
	// dibatasi 28 supaya tanggalnya selalu ada — Februari tidak punya 29-31
	if req.Tanggal < 1 || req.Tanggal > 28 {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST",
			"Tanggal harus 1–28 agar selalu ada di setiap bulan (termasuk Februari)")
		return
	}
	if req.Jam < 0 || req.Jam > 23 || req.Menit < 0 || req.Menit > 59 {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Jam harus 0–23 dan menit 0–59")
		return
	}
	if _, err := h.DB.Exec(`
		INSERT INTO pengingat_spp_pengaturan (id, aktif, tanggal, jam, menit) VALUES (1, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE aktif = VALUES(aktif), tanggal = VALUES(tanggal),
		                        jam = VALUES(jam), menit = VALUES(menit)`,
		req.Aktif, req.Tanggal, req.Jam, req.Menit); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, req)
}

// POST /pengingat-spp/kirim?tahun=&bulan= — kirim sekarang tanpa menunggu jadwal.
// Sengaja TIDAK mengubah `terakhir_kirim`: pengiriman manual tidak boleh membatalkan
// pengingat terjadwal bulan itu.
func (h *Handler) KirimPengingatSPP(w http.ResponseWriter, r *http.Request) {
	var chatID sql.NullString
	_ = h.DB.QueryRow(`SELECT chat_id FROM telegram_pengaturan WHERE id = 1`).Scan(&chatID)
	tujuan := strings.TrimSpace(chatID.String)
	if tujuan == "" {
		httpx.Error(w, http.StatusBadRequest, "CHAT_KOSONG", "Tujuan chat Telegram belum diatur")
		return
	}

	tahun, bulan := bulanDiminta(r)
	rekap, err := HitungTunggakanSPP(h.DB, tahun, bulan)
	if err != nil {
		dbErr(w, err)
		return
	}
	if rekap.Belum == 0 {
		httpx.Error(w, http.StatusBadRequest, "TIDAK_ADA_TUNGGAKAN",
			fmt.Sprintf("Seluruh santri sudah lunas SPP %s %d — tidak ada yang perlu diingatkan.",
				rekap.NamaBulan, rekap.Tahun))
		return
	}

	ref := fmt.Sprintf("%04d-%02d-01", tahun, bulan)
	if err := AntreNotifikasi(h.DB, JenisPengingatSPP, ref, tujuan, PesanTunggakanSPP(rekap)); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("Pengingat SPP %s %d diantrekan (%d santri belum bayar).",
			rekap.NamaBulan, rekap.Tahun, rekap.Belum),
		"belum": rekap.Belum,
	})
}
