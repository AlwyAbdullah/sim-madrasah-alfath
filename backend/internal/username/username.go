// Package username menyusun username login dari nama guru.
//
// Dipisahkan jadi paket sendiri karena aturannya khas nama Arab-Indonesia dan
// mudah salah: ada gelar di depan ("U" = Ustadz), kata sambung nasab ("bin"),
// dan artikel "al" yang menyatu dengan kata sesudahnya ("AL KAFF" -> "alkaff").
// Salah satu saja membuat hasilnya terasa asing bagi pemiliknya.
package username

import (
	"fmt"
	"strings"
	"unicode"
)

// gelar di depan nama yang dibuang
var gelar = map[string]bool{"u": true, "ust": true, "ustad": true, "ustadz": true}

// kata sambung nasab — bukan bagian nama yang dipakai untuk username
var sambung = map[string]bool{"bin": true, "binti": true, "ibn": true}

// bersihkan menyisakan huruf & angka saja, huruf kecil semua.
func bersihkan(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Bagian memecah nama menjadi kata-kata yang dipakai untuk username:
// gelar depan dibuang, "bin"/"binti" dibuang, dan "al" disatukan dengan kata
// sesudahnya sehingga "ALI AL KAFF" menjadi ["ali", "alkaff"].
func Bagian(nama string) []string {
	kata := strings.Fields(strings.TrimSpace(nama))
	var out []string
	for i := 0; i < len(kata); i++ {
		k := bersihkan(kata[i])
		if k == "" {
			continue
		}
		// gelar hanya dibuang bila berada di posisi paling depan
		if len(out) == 0 && gelar[k] {
			continue
		}
		if sambung[k] {
			continue
		}
		// "al" menyatu dengan kata berikutnya
		if k == "al" && i+1 < len(kata) {
			lanjut := bersihkan(kata[i+1])
			if lanjut != "" && !sambung[lanjut] {
				out = append(out, k+lanjut)
				i++
				continue
			}
		}
		out = append(out, k)
	}
	return out
}

// Calon mengembalikan daftar username usulan dari yang paling disukai ke yang
// paling akhir. Pemanggil memilih calon pertama yang belum dipakai.
//
// Urutannya: "depan.belakang", lalu "depan.tengah" (bila ada nama tengah — ini
// yang memisahkan dua orang bernama mirip seperti "Idrus bin Agil" dan
// "Idrus Tsani bin Agil"), lalu berakhiran angka.
func Calon(nama string) []string {
	p := Bagian(nama)
	if len(p) == 0 {
		return nil
	}

	var dasar []string
	tambah := func(s string) {
		for _, a := range dasar {
			if a == s {
				return
			}
		}
		dasar = append(dasar, s)
	}

	if len(p) == 1 {
		tambah(p[0])
	} else {
		tambah(p[0] + "." + p[len(p)-1])
		for i := 1; i < len(p)-1; i++ {
			tambah(p[0] + "." + p[i])
		}
	}

	out := append([]string{}, dasar...)
	// cadangan berakhiran angka bila semua bentuk di atas sudah terpakai
	for n := 2; n <= 9; n++ {
		out = append(out, fmt.Sprintf("%s%d", dasar[0], n))
	}
	return out
}

// Pilih mengembalikan calon pertama yang belum ada di `terpakai`.
// Kunci pada `terpakai` harus sudah huruf kecil.
func Pilih(nama string, terpakai map[string]bool) string {
	for _, c := range Calon(nama) {
		if !terpakai[c] {
			return c
		}
	}
	return ""
}
