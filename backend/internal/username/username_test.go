package username

import "testing"

// Nama-nama di bawah diambil dari master guru sungguhan; kalau aturannya berubah
// dan hasilnya bergeser, guru akan menerima username yang tidak dikenalinya.
func TestPilihUsernameDariMasterGuru(t *testing.T) {
	// urutan sama dengan urutan id di master, karena calon pertama diberikan
	// kepada yang lebih dulu diproses
	nama := []string{
		"ACHMAD BIN BIN ALI ASSEGAF ",
		"IDRUS TSANI BIN AGIL",
		"U IDRUS BIN AGIL ",
		"U SHOLEH ASSEGAF ",
		"U HUSIN BAAGIL ",
		"U ALWI ASSEGAF ",
		"U ZAMRONI ",
		"U ARIF RAHMAN ",
		"U ALI AL KAFF",
		"U ALWI AL KAFF",
		"U IRHAS SOLTHONI",
		"U ISMAIL AL HASNI ",
		"U SALIM BSA ",
		"U MUHAMMAD AL HAMID ",
		"ABUBAKAR MAULADAWILAH ",
		"NIZAR",
		"ALWY AL IDRUS",
		"MUHAMMAD MASHUR",
		"MUHAMMAD AL HADDAD",
		"ROMLI",
	}
	mau := []string{
		"achmad.assegaf",
		"idrus.agil", // diproses lebih dulu -> dapat bentuk dasar
		"idrus.agil2",
		"sholeh.assegaf",
		"husin.baagil",
		"alwi.assegaf",
		"zamroni",
		"arif.rahman",
		"ali.alkaff",
		"alwi.alkaff",
		"irhas.solthoni",
		"ismail.alhasni",
		"salim.bsa",
		"muhammad.alhamid",
		"abubakar.mauladawilah",
		"nizar",
		"alwy.alidrus",
		"muhammad.mashur",
		"muhammad.alhaddad",
		"romli",
	}

	terpakai := map[string]bool{}
	for i, n := range nama {
		got := Pilih(n, terpakai)
		if got != mau[i] {
			t.Errorf("Pilih(%q) = %q, mau %q", n, got, mau[i])
		}
		terpakai[got] = true
	}
}

// "Idrus Tsani bin Agil" dan "Idrus bin Agil" sama-sama menghasilkan
// "idrus.agil". Yang punya nama tengah harus bisa memakai nama tengah itu
// sebagai pembeda, bukan sekadar ditempeli angka.
func TestNamaTengahJadiPembeda(t *testing.T) {
	terpakai := map[string]bool{"idrus.agil": true}
	if got := Pilih("IDRUS TSANI BIN AGIL", terpakai); got != "idrus.tsani" {
		t.Errorf("mau idrus.tsani, dapat %q", got)
	}
	// tanpa nama tengah tidak ada pilihan lain selain berakhiran angka
	if got := Pilih("U IDRUS BIN AGIL", terpakai); got != "idrus.agil2" {
		t.Errorf("mau idrus.agil2, dapat %q", got)
	}
}

func TestBagian(t *testing.T) {
	kasus := []struct {
		nama string
		mau  []string
	}{
		{"U ALI AL KAFF", []string{"ali", "alkaff"}},              // "al" menyatu
		{"ACHMAD BIN BIN ALI ASSEGAF", []string{"achmad", "ali", "assegaf"}}, // "bin" ganda
		{"U ZAMRONI", []string{"zamroni"}},                        // gelar dibuang
		{"ALWY AL IDRUS", []string{"alwy", "alidrus"}},
		{"  ROMLI  ", []string{"romli"}},
	}
	for _, k := range kasus {
		got := Bagian(k.nama)
		if len(got) != len(k.mau) {
			t.Errorf("Bagian(%q) = %v, mau %v", k.nama, got, k.mau)
			continue
		}
		for i := range got {
			if got[i] != k.mau[i] {
				t.Errorf("Bagian(%q) = %v, mau %v", k.nama, got, k.mau)
				break
			}
		}
	}
}

// Gelar "U" hanya boleh dibuang di depan — kalau muncul di tengah, itu bagian nama.
func TestGelarHanyaDiDepan(t *testing.T) {
	got := Bagian("AHMAD U SALIM")
	mau := []string{"ahmad", "u", "salim"}
	for i := range mau {
		if i >= len(got) || got[i] != mau[i] {
			t.Fatalf("Bagian = %v, mau %v", got, mau)
		}
	}
}

func TestNamaKosong(t *testing.T) {
	if got := Pilih("   ", map[string]bool{}); got != "" {
		t.Errorf("nama kosong harus menghasilkan string kosong, dapat %q", got)
	}
}
