package gembok

import (
	"fmt"
	"testing"
	"time"
)

func gagalBerkali(g *Gembok, akun, ip string, n int) {
	for i := 0; i < n; i++ {
		g.Gagal(akun, ip)
	}
}

// Inti fiturnya: 5 kali salah -> terkunci, dan percobaan berikutnya ditolak
// sebelum password sempat diperiksa.
func TestAkunTerkunciSetelahBatas(t *testing.T) {
	g := Bawaan()
	gagalBerkali(g, "romli", "10.0.0.1", MaksAkun-1)
	if _, boleh := g.Periksa("romli", "10.0.0.1"); !boleh {
		t.Fatal("belum mencapai batas, seharusnya masih boleh mencoba")
	}
	g.Gagal("romli", "10.0.0.1")
	sisa, boleh := g.Periksa("romli", "10.0.0.1")
	if boleh {
		t.Fatal("setelah 5 kali gagal seharusnya terkunci")
	}
	if sisa <= 0 || sisa > Jendela {
		t.Fatalf("sisa waktu kunci tidak masuk akal: %v", sisa)
	}
}

// Login yang BERHASIL tidak boleh menghabiskan jatah — ini persis bug pembatas
// lama, yang menghitung semua percobaan termasuk yang benar.
func TestBerhasilMenghapusHitungan(t *testing.T) {
	g := Bawaan()
	gagalBerkali(g, "romli", "10.0.0.1", MaksAkun-1)
	g.Berhasil("romli", "10.0.0.1")
	if _, gagal, _ := g.Status("romli"); gagal != 0 {
		t.Fatalf("hitungan gagal harus nol setelah login berhasil, dapat %d", gagal)
	}
	gagalBerkali(g, "romli", "10.0.0.1", MaksAkun-1)
	if _, boleh := g.Periksa("romli", "10.0.0.1"); !boleh {
		t.Fatal("jatahnya harus dihitung dari nol lagi")
	}
}

// Satu WiFi madrasah = satu IP untuk semua guru. Beberapa orang salah ketik
// tidak boleh menutup jalan masuk bagi yang lain.
func TestSatuIPBersamaTidakLangsungMengunciSemua(t *testing.T) {
	g := Bawaan()
	const ip = "103.175.219.47"
	for i := 0; i < 4; i++ {
		gagalBerkali(g, fmt.Sprintf("guru%d", i), ip, MaksAkun-1)
	}
	if _, boleh := g.Periksa("orang.lain", ip); !boleh {
		t.Fatal("guru lain dari IP yang sama masih harus bisa mencoba login")
	}
}

// Tapi satu IP yang mencoba banyak akun sekaligus (pemindai) tetap harus berhenti.
func TestIPTeruskunciSetelahBanyakAkunGagal(t *testing.T) {
	g := Bawaan()
	const ip = "198.51.100.7"
	for i := 0; i < MaksIP; i++ {
		g.Gagal(fmt.Sprintf("tebakan%d", i), ip)
	}
	if _, boleh := g.Periksa("akun.baru", ip); boleh {
		t.Fatal("IP yang mencoba banyak akun seharusnya terkunci")
	}
}

// Tombol admin harus benar-benar membuka: bukan hanya nama akunnya, tapi juga
// IP yang ikut terkunci karena percobaan gagal akun tersebut.
func TestBukaAkunIkutMembukaIPnya(t *testing.T) {
	g := Baru(3, 5, Jendela)
	for i := 0; i < 5; i++ {
		g.Gagal("romli", "10.0.0.9")
	}
	if _, boleh := g.Periksa("romli", "10.0.0.9"); boleh {
		t.Fatal("prasyarat gagal: akun dan IP seharusnya terkunci")
	}
	if !g.BukaAkun("romli") {
		t.Fatal("BukaAkun harus melaporkan bahwa ada yang dibuka")
	}
	if _, boleh := g.Periksa("romli", "10.0.0.9"); !boleh {
		t.Fatal("setelah dibuka admin, login harus bisa dicoba lagi")
	}
}

// Huruf besar/kecil tidak boleh jadi celah menghindari penguncian.
func TestNamaAkunTidakPekaHurufBesar(t *testing.T) {
	g := Bawaan()
	gagalBerkali(g, "Romli", "10.0.0.1", MaksAkun)
	if _, boleh := g.Periksa("  romli  ", "10.0.0.1"); boleh {
		t.Fatal("ROMLI/romli harus dihitung sebagai akun yang sama")
	}
}

// Kunci hilang sendiri setelah jendela waktu lewat, tanpa perlu admin.
func TestKunciLepasSetelahJendelaLewat(t *testing.T) {
	g := Baru(2, 10, 40*time.Millisecond)
	g.Gagal("romli", "10.0.0.1")
	g.Gagal("romli", "10.0.0.1")
	if _, boleh := g.Periksa("romli", "10.0.0.1"); boleh {
		t.Fatal("prasyarat gagal: seharusnya terkunci")
	}
	time.Sleep(60 * time.Millisecond)
	if _, boleh := g.Periksa("romli", "10.0.0.1"); !boleh {
		t.Fatal("kunci harus lepas sendiri setelah jendela waktu lewat")
	}
}

func TestDaftarHanyaBerisiYangTerkunci(t *testing.T) {
	g := Bawaan()
	gagalBerkali(g, "romli", "10.0.0.1", MaksAkun)
	gagalBerkali(g, "nizar", "10.0.0.2", MaksAkun-2) // belum kena
	d := g.Daftar()
	if len(d) != 1 || d[0].Jenis != "akun" || d[0].Kunci != "romli" {
		t.Fatalf("daftar seharusnya hanya berisi akun romli, dapat %+v", d)
	}
}
