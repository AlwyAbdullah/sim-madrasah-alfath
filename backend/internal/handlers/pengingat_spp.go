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
// ruang untuk kepala & penutup pesan.
const batasPesanTelegram = 3500

// BulanTagih = satu bulan yang ditagih, dalam penanggalan biasa.
type BulanTagih struct {
	Tahun int `json:"tahun"`
	Bulan int `json:"bulan"`
}

func (b BulanTagih) kunci() int { return b.Tahun*100 + b.Bulan }

// bulanTagihan menyusun daftar bulan yang sudah jatuh tempo pada satu tahun
// ajaran: dari Juli sampai bulan yang diminta.
//
// Sengaja BERHENTI di tahun ajaran berjalan dan tidak menengok ke belakang.
// Tahun ajaran sebelumnya punya kelas dan catatan yang berbeda, dan datanya
// jauh lebih jarang — menagihnya di sini hanya akan memunculkan tunggakan
// karangan.
func bulanTagihan(tahun, bulan int) []BulanTagih {
	startYear := tahun
	if bulan < 7 {
		startYear = tahun - 1
	}
	out := []BulanTagih{}
	for _, m := range urutanBulanTA {
		t := calTahun(startYear, m)
		out = append(out, BulanTagih{Tahun: t, Bulan: m})
		if t == tahun && m == bulan {
			break
		}
	}
	return out
}

// SantriTunggakan = satu santri berikut bulan-bulan yang belum ia bayar.
type SantriTunggakan struct {
	SantriID int64        `json:"santri_id"`
	Nama     string       `json:"nama"`
	Bulan    []BulanTagih `json:"bulan"`
	// Teks siap baca, mis. "Agustus–September (2 bln)".
	Teks string `json:"teks"`
}

// TunggakanKelas = daftar santri satu kelas yang masih punya tunggakan SPP.
type TunggakanKelas struct {
	KelasID int64             `json:"kelas_id"`
	Kelas   string            `json:"kelas"`
	Total   int               `json:"total"` // seluruh santri aktif di kelas ini
	Belum   []SantriTunggakan `json:"belum"`
}

// RekapTunggakanSPP = kondisi pembayaran SPP satu tahun ajaran sampai bulan tertentu.
type RekapTunggakanSPP struct {
	Tahun       int              `json:"tahun"` // tahun kalender bulan terakhir yang ditagih
	Bulan       int              `json:"bulan"` // 1..12
	NamaBulan   string           `json:"nama_bulan"`
	TahunAjaran string           `json:"tahun_ajaran"` // mis. "2026/2027"
	Ditagih     []BulanTagih     `json:"ditagih"`      // bulan yang ikut dihitung
	TotalSantri int              `json:"total_santri"`
	Lunas       int              `json:"lunas"` // tidak punya tunggakan sama sekali
	Belum       int              `json:"belum"` // punya minimal satu bulan tertunggak
	Kelas       []TunggakanKelas `json:"kelas"`
}

// HitungTunggakanSPP mencari santri aktif yang masih punya tunggakan SPP pada
// tahun ajaran berjalan, sampai bulan yang diminta.
//
// Dulu hanya memeriksa SATU bulan. Akibatnya santri yang membayar bulan ini
// tapi menunggak bulan lalu tidak pernah muncul di pengingat — padahal justru
// dia yang perlu ditagih. Sekarang seluruh bulan yang sudah jatuh tempo pada
// tahun ajaran ini diperiksa, dan tiap santri dilaporkan menunggak bulan apa
// sampai apa.
//
// Memakai daftar bulan LUNAS, bukan filter `lunas = 0`: santri yang belum punya
// baris SPP sama sekali juga terhitung belum bayar. Ini yang membedakannya dari
// sekadar membaca tabel spp — di produksi banyak santri memang belum punya barisnya.
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
	out.Ditagih = bulanTagihan(tahun, bulan)
	if len(out.Ditagih) > 0 {
		mulai := out.Ditagih[0].Tahun
		out.TahunAjaran = fmt.Sprintf("%d/%d", mulai, mulai+1)
	}

	// Bulan yang sudah lunas, per santri.
	tanda := make([]string, 0, len(out.Ditagih))
	args := make([]interface{}, 0, len(out.Ditagih))
	for _, b := range out.Ditagih {
		tanda = append(tanda, "?")
		args = append(args, b.kunci())
	}
	lunasnya := map[int64]map[int]bool{}
	if len(args) > 0 {
		lrows, err := db.Query(`
			SELECT santri_id, tahun, bulan FROM spp
			WHERE lunas = 1 AND (tahun * 100 + bulan) IN (`+strings.Join(tanda, ",")+`)`, args...)
		if err != nil {
			return out, err
		}
		for lrows.Next() {
			var sid int64
			var t, m int
			if lrows.Scan(&sid, &t, &m) != nil {
				continue
			}
			if lunasnya[sid] == nil {
				lunasnya[sid] = map[int]bool{}
			}
			lunasnya[sid][t*100+m] = true
		}
		lrows.Close()
		if err := lrows.Err(); err != nil {
			return out, err
		}
	}

	rows, err := db.Query(`
		SELECT k.id, k.nama, s.id, s.nama
		FROM santri s
		JOIN kelas k ON k.id = s.kelas_id
		WHERE s.is_active = 1 AND k.aktif = 1
		ORDER BY k.nama, s.nama`)
	if err != nil {
		return out, err
	}
	defer rows.Close()

	idx := map[int64]int{}
	for rows.Next() {
		var kelasID, santriID int64
		var kelas, nama string
		if err := rows.Scan(&kelasID, &kelas, &santriID, &nama); err != nil {
			continue
		}
		i, ada := idx[kelasID]
		if !ada {
			i = len(out.Kelas)
			idx[kelasID] = i
			out.Kelas = append(out.Kelas, TunggakanKelas{KelasID: kelasID, Kelas: kelas, Belum: []SantriTunggakan{}})
		}
		out.Kelas[i].Total++
		out.TotalSantri++

		var nunggak []BulanTagih
		for _, b := range out.Ditagih {
			if !lunasnya[santriID][b.kunci()] {
				nunggak = append(nunggak, b)
			}
		}
		if len(nunggak) == 0 {
			out.Lunas++
			continue
		}
		out.Kelas[i].Belum = append(out.Kelas[i].Belum, SantriTunggakan{
			SantriID: santriID, Nama: nama, Bulan: nunggak, Teks: TeksBulanTunggakan(nunggak),
		})
		out.Belum++
	}
	if out.Kelas == nil {
		out.Kelas = []TunggakanKelas{}
	}
	return out, rows.Err()
}

// TeksBulanTunggakan merangkum bulan-bulan tertunggak jadi satu frasa pendek.
//
// Bulan yang berurutan digabung jadi rentang ("Juli–September") karena itulah
// bentuk yang dipakai orang saat menagih. Bulan yang terputus tetap disebut
// terpisah ("Juli, September") — kalau dipaksa jadi satu rentang, bulan yang
// sudah dibayar ikut tertagih dan pengingatnya jadi tidak dipercaya.
func TeksBulanTunggakan(b []BulanTagih) string {
	if len(b) == 0 {
		return ""
	}
	// posisi dalam urutan tahun ajaran, untuk mengetahui mana yang berurutan
	posisi := map[int]int{}
	for i, m := range urutanBulanTA {
		posisi[m] = i
	}

	bagian := []string{}
	awal, akhir := b[0], b[0]
	tutup := func() {
		if awal == akhir {
			bagian = append(bagian, bulanIndo[awal.Bulan-1])
			return
		}
		bagian = append(bagian, bulanIndo[awal.Bulan-1]+"–"+bulanIndo[akhir.Bulan-1])
	}
	for _, m := range b[1:] {
		if posisi[m.Bulan] == posisi[akhir.Bulan]+1 {
			akhir = m
			continue
		}
		tutup()
		awal, akhir = m, m
	}
	tutup()

	teks := strings.Join(bagian, ", ")
	if len(b) > 1 {
		teks += fmt.Sprintf(" (%d bln)", len(b))
	}
	return teks
}

// PesanTunggakanSPP menyusun teks pengingat, dipecah menjadi beberapa pesan
// bila tidak muat dalam satu kiriman Telegram.
//
// Dulu daftar namanya DIBUANG begitu pesan kepanjangan, dan yang tersisa hanya
// jumlah per kelas. Justru nama dan bulan tertunggak itulah isi yang berguna —
// jadi sekarang pesannya yang dipecah, bukan isinya yang dikurangi.
func PesanTunggakanSPP(r RekapTunggakanSPP) []string {
	penutup := "\nMohon ditindaklanjuti. Jazakumullahu khairan.\n\n_SIM Madrasah Al Fath_"
	kepala := func(bagian, dari int) string {
		var b strings.Builder
		b.WriteString("Assalamu'alaikum warahmatullahi wabarakatuh.\n\n")
		b.WriteString(fmt.Sprintf("Pengingat *SPP* — tahun ajaran %s, sampai *%s %d*.\n",
			r.TahunAjaran, r.NamaBulan, r.Tahun))
		b.WriteString(fmt.Sprintf("Belum lunas: *%d dari %d santri* (lunas semua bulan: %d).\n",
			r.Belum, r.TotalSantri, r.Lunas))
		if dari > 1 {
			b.WriteString(fmt.Sprintf("_Bagian %d dari %d._\n", bagian, dari))
		}
		return b.String()
	}

	// Satu blok = satu kelas. Kelas yang terlalu besar untuk satu pesan dipecah
	// lagi, dengan judulnya diulang supaya potongan berikutnya tetap jelas
	// menyebut kelas apa.
	ruang := batasPesanTelegram - len(kepala(9, 9)) - len(penutup)
	blok := []string{}
	for _, k := range r.Kelas {
		if len(k.Belum) == 0 {
			continue
		}
		judul := fmt.Sprintf("*%s* — %d dari %d belum lunas\n", k.Kelas, len(k.Belum), k.Total)
		isi := judul
		for _, s := range k.Belum {
			baris := fmt.Sprintf("• %s — %s\n", strings.TrimSpace(s.Nama), s.Teks)
			if len(isi)+len(baris) > ruang {
				blok = append(blok, isi)
				isi = judul
			}
			isi += baris
		}
		blok = append(blok, isi)
	}
	if len(blok) == 0 {
		return []string{kepala(1, 1) + penutup}
	}

	// Dikemas dua kali: putaran pertama untuk mengetahui ada berapa bagian,
	// putaran kedua supaya tiap pesan bisa menyebut "bagian 2 dari 3".
	kemas := func(dari int) []string {
		pesan := []string{}
		isi := ""
		for _, bl := range blok {
			calon := isi
			if calon != "" {
				calon += "\n"
			}
			calon += bl
			if isi != "" && len(kepala(len(pesan)+1, dari))+len(calon)+len(penutup) > batasPesanTelegram {
				pesan = append(pesan, kepala(len(pesan)+1, dari)+"\n"+isi+penutup)
				isi = bl
				continue
			}
			isi = calon
		}
		if isi != "" {
			pesan = append(pesan, kepala(len(pesan)+1, dari)+"\n"+isi+penutup)
		}
		return pesan
	}
	return kemas(len(kemas(1)))
}

// AntreNotifikasi memasukkan satu pesan ke antrean pengiriman. Worker Telegram
// yang mengambilnya pada polling berikutnya.
func AntreNotifikasi(db *sql.DB, jenis, refTanggal, tujuan, pesan string) error {
	_, err := db.Exec(`
		INSERT INTO notifikasi (santri_id, jenis, ref_tanggal, tujuan, pesan, status)
		VALUES (NULL, ?, ?, ?, ?, 'pending')`, jenis, refTanggal, tujuan, pesan)
	return err
}

// AntreBanyakNotifikasi mengantrekan pesan yang sudah dipecah, berurutan.
// Bila satu bagian gagal, sisanya tetap dicoba: pengingat yang tiba sebagian
// masih lebih berguna daripada tidak ada sama sekali.
func AntreBanyakNotifikasi(db *sql.DB, jenis, refTanggal, tujuan string, pesan []string) error {
	var pertama error
	for _, p := range pesan {
		if err := AntreNotifikasi(db, jenis, refTanggal, tujuan, p); err != nil && pertama == nil {
			pertama = err
		}
	}
	return pertama
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

// GET /pengingat-spp?tahun=&bulan= — jadwal + rekap tunggakan sampai bulan tersebut.
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
			fmt.Sprintf("Tidak ada tunggakan sampai %s %d — tidak ada yang perlu diingatkan.",
				rekap.NamaBulan, rekap.Tahun))
		return
	}

	ref := fmt.Sprintf("%04d-%02d-01", tahun, bulan)
	pesan := PesanTunggakanSPP(rekap)
	if err := AntreBanyakNotifikasi(h.DB, JenisPengingatSPP, ref, tujuan, pesan); err != nil {
		dbErr(w, err)
		return
	}
	catatan := ""
	if len(pesan) > 1 {
		catatan = fmt.Sprintf(" Daftarnya panjang, jadi dikirim dalam %d pesan.", len(pesan))
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("Pengingat SPP sampai %s %d diantrekan (%d santri punya tunggakan).%s",
			rekap.NamaBulan, rekap.Tahun, rekap.Belum, catatan),
		"belum": rekap.Belum,
		"pesan": len(pesan),
	})
}
