// Package gembok membatasi percobaan login yang GAGAL, dan memberi admin cara
// membukanya kembali.
//
// Kenapa bukan sekadar pembatas laju per IP seperti sebelumnya:
//
//   - Seluruh madrasah memakai satu jaringan. Dari sisi server, 20 guru yang
//     login dari WiFi yang sama terlihat sebagai SATU alamat IP. Jatah "5
//     percobaan per IP" berarti orang keenam yang login pagi itu ditolak
//     walaupun passwordnya benar.
//   - Percobaan yang BERHASIL dulu ikut dihitung. Login normal pun bisa
//     menghabiskan jatah, sehingga yang terkunci justru pengguna yang sah.
//
// Karena itu yang dihitung sekarang hanya percobaan gagal, dengan dua
// hitungan terpisah: per nama akun (ketat — inilah yang menghalangi penebakan
// password satu orang) dan per IP (longgar — jaring pengaman untuk pemindai
// yang mencoba banyak akun sekaligus). Login yang berhasil menghapus keduanya.
//
// Konsekuensi yang disengaja: orang lain bisa membuat sebuah akun terkunci
// dengan sengaja salah password 5 kali. Itu melekat pada semua penguncian
// berbasis akun. Ruginya dibatasi dua hal — kuncinya hilang sendiri setelah
// jendela waktu lewat, dan admin bisa membukanya seketika dari halaman User.
//
// Seluruh data disimpan di memori: mati listrik / restart layanan =
// semua kunci terbuka. Untuk satu server madrasah ini justru sifat yang
// diinginkan; tidak ada yang perlu dibersihkan dari database.
package gembok

import (
	"strings"
	"sync"
	"time"
)

// Nilai bawaan. Per akun sengaja jauh lebih ketat daripada per IP: satu orang
// yang lupa password paling banter salah 3–4 kali, sedangkan satu IP menaungi
// seluruh guru madrasah.
const (
	MaksAkun = 5
	MaksIP   = 20
	Jendela  = 15 * time.Minute
)

type catatan struct {
	waktu []time.Time
	// IP yang pernah gagal untuk akun ini. Dipakai saat admin membuka kunci:
	// membuka nama akunnya saja tidak cukup bila IP-nya ikut terkunci —
	// orangnya akan tetap tertolak dan mengira tombolnya tidak berfungsi.
	ip map[string]bool
}

type Gembok struct {
	mu       sync.Mutex
	akun     map[string]*catatan
	ip       map[string]*catatan
	maksAkun int
	maksIP   int
	jendela  time.Duration
}

func Baru(maksAkun, maksIP int, jendela time.Duration) *Gembok {
	return &Gembok{
		akun:     map[string]*catatan{},
		ip:       map[string]*catatan{},
		maksAkun: maksAkun,
		maksIP:   maksIP,
		jendela:  jendela,
	}
}

// Bawaan membuat gembok dengan nilai yang dipakai aplikasi.
func Bawaan() *Gembok { return Baru(MaksAkun, MaksIP, Jendela) }

// Kunci menormalkan nama akun agar "Romli" dan " romli " dihitung sama —
// kalau tidak, penguncian per akun gampang dihindari dengan mengubah huruf.
func Kunci(namaAkun string) string { return strings.ToLower(strings.TrimSpace(namaAkun)) }

// Terkunci menggambarkan satu penguncian yang sedang berlaku.
type Terkunci struct {
	Jenis     string `json:"jenis"` // "akun" | "ip"
	Kunci     string `json:"kunci"`
	Gagal     int    `json:"gagal"`
	SisaDetik int    `json:"sisa_detik"`
}

// Periksa menjawab apakah percobaan login boleh diteruskan. Bila tidak, sisa
// berisi berapa lama lagi kuncinya berlaku.
func (g *Gembok) Periksa(namaAkun, ip string) (sisa time.Duration, boleh bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()

	if s := g.sisaKunci(g.akun[Kunci(namaAkun)], g.maksAkun, now); s > 0 {
		return s, false
	}
	if s := g.sisaKunci(g.ip[ip], g.maksIP, now); s > 0 {
		return s, false
	}
	return 0, true
}

// Gagal mencatat satu percobaan gagal dan mengembalikan sisa kesempatan
// sebelum akun terkunci (0 berarti barusan terkunci).
func (g *Gembok) Gagal(namaAkun, ip string) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()

	ka := Kunci(namaAkun)
	c := g.catat(g.akun, ka, now)
	if ip != "" {
		c.ip[ip] = true
		g.catat(g.ip, ip, now)
	}

	sisa := g.maksAkun - len(c.waktu)
	if sisa < 0 {
		sisa = 0
	}
	g.sapu(now)
	return sisa
}

// Berhasil menghapus hitungan gagal setelah login yang sah.
//
// Hitungan IP ikut dihapus karena satu login yang benar dari jaringan itu
// membuktikan yang memakainya orang madrasah, bukan pemindai. Tanpa ini,
// beberapa guru yang salah ketik di pagi yang sama bisa menutup jalan masuk
// bagi seluruh sekolah sampai jendela waktunya lewat.
func (g *Gembok) Berhasil(namaAkun, ip string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.akun, Kunci(namaAkun))
	if ip != "" {
		delete(g.ip, ip)
	}
}

// Status dipakai daftar user: apakah akun ini sedang terkunci dan berapa kali gagal.
func (g *Gembok) Status(namaAkun string) (terkunci bool, gagal, sisaDetik int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	c := g.akun[Kunci(namaAkun)]
	if c == nil {
		return false, 0, 0
	}
	hidup := saring(c.waktu, now, g.jendela)
	if len(hidup) == 0 {
		return false, 0, 0
	}
	if len(hidup) < g.maksAkun {
		return false, len(hidup), 0
	}
	return true, len(hidup), int((g.jendela - now.Sub(hidup[0])).Seconds())
}

// Daftar mengembalikan semua penguncian yang sedang berlaku.
func (g *Gembok) Daftar() []Terkunci {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()

	out := []Terkunci{}
	kumpul := func(jenis string, m map[string]*catatan, maks int) {
		for k, c := range m {
			if s := g.sisaKunci(c, maks, now); s > 0 {
				out = append(out, Terkunci{
					Jenis: jenis, Kunci: k,
					Gagal:     len(saring(c.waktu, now, g.jendela)),
					SisaDetik: int(s.Seconds()),
				})
			}
		}
	}
	kumpul("akun", g.akun, g.maksAkun)
	kumpul("ip", g.ip, g.maksIP)
	return out
}

// BukaAkun membuka kunci sebuah akun berikut IP yang tercatat gagal untuknya.
// Mengembalikan false bila akun itu memang tidak sedang terkunci.
func (g *Gembok) BukaAkun(namaAkun string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	ka := Kunci(namaAkun)
	c := g.akun[ka]
	if c == nil {
		return false
	}
	for ip := range c.ip {
		delete(g.ip, ip)
	}
	delete(g.akun, ka)
	return true
}

// BukaIP membuka kunci satu alamat IP.
func (g *Gembok) BukaIP(ip string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ada := g.ip[ip]; !ada {
		return false
	}
	delete(g.ip, ip)
	return true
}

// ===== bagian dalam (pemanggil sudah memegang mu) =====

func (g *Gembok) catat(m map[string]*catatan, kunci string, now time.Time) *catatan {
	c := m[kunci]
	if c == nil {
		c = &catatan{ip: map[string]bool{}}
		m[kunci] = c
	}
	c.waktu = append(saring(c.waktu, now, g.jendela), now)
	return c
}

// sisaKunci: >0 berarti terkunci, nilainya sisa waktu sampai percobaan
// terlama keluar dari jendela.
func (g *Gembok) sisaKunci(c *catatan, maks int, now time.Time) time.Duration {
	if c == nil {
		return 0
	}
	hidup := saring(c.waktu, now, g.jendela)
	if len(hidup) < maks {
		return 0
	}
	sisa := g.jendela - now.Sub(hidup[0])
	if sisa < 0 {
		return 0
	}
	return sisa
}

// sapu membuang catatan yang sudah tidak berguna. Tanpa ini map terus tumbuh
// oleh nama akun karangan — dan nama akun berasal dari luar, jadi jumlahnya
// tidak terbatas.
func (g *Gembok) sapu(now time.Time) {
	if len(g.akun)+len(g.ip) < 200 {
		return
	}
	for _, m := range []map[string]*catatan{g.akun, g.ip} {
		for k, c := range m {
			if len(saring(c.waktu, now, g.jendela)) == 0 {
				delete(m, k)
			}
		}
	}
}

// saring menyisakan percobaan yang masih di dalam jendela waktu.
func saring(waktu []time.Time, now time.Time, jendela time.Duration) []time.Time {
	hidup := waktu[:0:0]
	for _, t := range waktu {
		if now.Sub(t) < jendela {
			hidup = append(hidup, t)
		}
	}
	return hidup
}
