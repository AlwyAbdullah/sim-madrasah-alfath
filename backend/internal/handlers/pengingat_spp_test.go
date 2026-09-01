package handlers

import (
	"fmt"
	"strings"
	"testing"
)

func bulan(pasangan ...int) []BulanTagih {
	out := []BulanTagih{}
	for i := 0; i+1 < len(pasangan); i += 2 {
		out = append(out, BulanTagih{Tahun: pasangan[i], Bulan: pasangan[i+1]})
	}
	return out
}

func TestTeksBulanTunggakan(t *testing.T) {
	kasus := []struct {
		nama string
		in   []BulanTagih
		mau  string
	}{
		{"satu bulan", bulan(2026, 9), "September"},
		{"berurutan", bulan(2026, 7, 2026, 8, 2026, 9), "Juli–September (3 bln)"},
		{"terputus", bulan(2026, 7, 2026, 9), "Juli, September (2 bln)"},
		{"lintas tahun kalender", bulan(2026, 11, 2026, 12, 2027, 1), "November–Januari (3 bln)"},
		{"dua rentang", bulan(2026, 7, 2026, 8, 2026, 10, 2026, 11), "Juli–Agustus, Oktober–November (4 bln)"},
		{"kosong", nil, ""},
	}
	for _, k := range kasus {
		if dapat := TeksBulanTunggakan(k.in); dapat != k.mau {
			t.Errorf("%s: mau %q, dapat %q", k.nama, k.mau, dapat)
		}
	}
}

// Juli–Juni tetap urut menurut tahun ajaran, bukan menurut nomor bulan:
// Januari datang SESUDAH Desember, jadi keduanya satu rentang.
func TestBulanTagihanUrutTahunAjaran(t *testing.T) {
	b := bulanTagihan(2027, 2) // Februari 2027 => TA 2026/2027
	if len(b) != 8 {
		t.Fatalf("Juli..Februari seharusnya 8 bulan, dapat %d", len(b))
	}
	if b[0] != (BulanTagih{2026, 7}) || b[len(b)-1] != (BulanTagih{2027, 2}) {
		t.Fatalf("rentang salah: %v .. %v", b[0], b[len(b)-1])
	}
	b = bulanTagihan(2026, 9) // September 2026 => TA 2026/2027
	if len(b) != 3 || b[0] != (BulanTagih{2026, 7}) {
		t.Fatalf("Juli..September seharusnya 3 bulan mulai Juli 2026, dapat %v", b)
	}
}

func rekapUji(jumlahKelas, perKelas int) RekapTunggakanSPP {
	r := RekapTunggakanSPP{
		Tahun: 2026, Bulan: 9, NamaBulan: "September", TahunAjaran: "2026/2027",
		Ditagih: bulanTagihan(2026, 9),
	}
	for k := 1; k <= jumlahKelas; k++ {
		kelas := TunggakanKelas{KelasID: int64(k), Kelas: fmt.Sprintf("Kelas %d Pagi", k), Total: perKelas}
		for i := 0; i < perKelas; i++ {
			b := bulan(2026, 8, 2026, 9)
			kelas.Belum = append(kelas.Belum, SantriTunggakan{
				SantriID: int64(k*100 + i),
				Nama:     fmt.Sprintf("MUHAMMAD SANTRI CONTOH KE %d-%d", k, i),
				Bulan:    b, Teks: TeksBulanTunggakan(b),
			})
		}
		r.Kelas = append(r.Kelas, kelas)
		r.Belum += perKelas
		r.TotalSantri += perKelas
	}
	return r
}

// Daftar panjang harus DIPECAH jadi beberapa pesan, bukan dibuang namanya:
// nama dan bulan tertunggak justru satu-satunya isi yang berguna.
func TestPesanPanjangDipecahBukanDipangkas(t *testing.T) {
	pesan := PesanTunggakanSPP(rekapUji(13, 20))
	if len(pesan) < 2 {
		t.Fatalf("260 santri seharusnya butuh lebih dari satu pesan, dapat %d", len(pesan))
	}
	gabung := strings.Join(pesan, "\n")
	for _, p := range pesan {
		if len(p) > batasPesanTelegram {
			t.Fatalf("ada pesan %d karakter, melewati batas %d", len(p), batasPesanTelegram)
		}
		if !strings.Contains(p, "Pengingat *SPP*") {
			t.Fatal("tiap bagian harus berdiri sendiri dengan kepala pesannya")
		}
	}
	// tiap santri tetap tersebut, lengkap dengan bulan tertunggaknya
	for _, k := range rekapUji(13, 20).Kelas {
		for _, s := range k.Belum {
			if !strings.Contains(gabung, s.Nama) {
				t.Fatalf("nama %q hilang dari pesan", s.Nama)
			}
		}
	}
	if !strings.Contains(pesan[0], fmt.Sprintf("Bagian 1 dari %d", len(pesan))) {
		t.Fatalf("penanda bagian tidak cocok dengan jumlah pesan (%d)", len(pesan))
	}
	if !strings.Contains(gabung, "Agustus–September (2 bln)") {
		t.Fatal("bulan tertunggak harus ikut tertulis di tiap baris santri")
	}
}

// Daftar pendek tetap satu pesan, tanpa penanda "Bagian 1 dari 1" yang mengganggu.
func TestPesanPendekTetapSatu(t *testing.T) {
	pesan := PesanTunggakanSPP(rekapUji(2, 3))
	if len(pesan) != 1 {
		t.Fatalf("6 santri seharusnya cukup satu pesan, dapat %d", len(pesan))
	}
	if strings.Contains(pesan[0], "Bagian") {
		t.Fatal("pesan tunggal tidak perlu penanda bagian")
	}
}

// Satu kelas yang sendirian sudah kepanjangan harus ikut dipecah, dengan judul
// kelas diulang supaya potongan berikutnya tidak jadi daftar nama tanpa konteks.
func TestKelasSangatBesarDipecahDenganJudulDiulang(t *testing.T) {
	pesan := PesanTunggakanSPP(rekapUji(1, 120))
	if len(pesan) < 2 {
		t.Fatalf("satu kelas 120 santri seharusnya lebih dari satu pesan, dapat %d", len(pesan))
	}
	for i, p := range pesan {
		if !strings.Contains(p, "*Kelas 1 Pagi*") {
			t.Fatalf("bagian %d tidak menyebut kelasnya", i+1)
		}
	}
}
