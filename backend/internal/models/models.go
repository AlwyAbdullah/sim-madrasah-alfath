package models

// Bobot nilai akhir: Tugas 30% + UTS 30% + UAS 40%
const (
	BobotTugas = 0.30
	BobotUTS   = 0.30
	BobotUAS   = 0.40
)

// HitungNilaiAkhir mengembalikan nilai akhir terbobot.
// Komponen yang nil dianggap 0.
func HitungNilaiAkhir(tugas, uts, uas *float64) float64 {
	val := func(p *float64) float64 {
		if p == nil {
			return 0
		}
		return *p
	}
	akhir := val(tugas)*BobotTugas + val(uts)*BobotUTS + val(uas)*BobotUAS
	// bulatkan 2 desimal
	return float64(int(akhir*100+0.5)) / 100
}

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Nama     string `json:"nama"`
	Role     string `json:"role"`
}

type Santri struct {
	ID           int64  `json:"id"`
	NIS          string `json:"nis"`
	Nama         string `json:"nama"`
	JenisKelamin string `json:"jenis_kelamin"`
	KelasID      int64  `json:"kelas_id"`
	KelasNama    string `json:"kelas_nama,omitempty"`
}

type AbsensiItem struct {
	SantriID   int64   `json:"santri_id"`
	Nama       string  `json:"nama,omitempty"`
	NIS        string  `json:"nis,omitempty"`
	Status     string  `json:"status"` // hadir|izin|sakit|alpha
	Keterangan *string `json:"keterangan,omitempty"`
}

type AbsensiBatch struct {
	KelasID int64         `json:"kelas_id"`
	Tanggal string        `json:"tanggal"` // YYYY-MM-DD
	Items   []AbsensiItem `json:"items"`
}

type NilaiItem struct {
	SantriID   int64    `json:"santri_id"`
	Nama       string   `json:"nama,omitempty"`
	NIS        string   `json:"nis,omitempty"`
	Tugas      *float64 `json:"tugas"`
	UTS        *float64 `json:"uts"`
	UAS        *float64 `json:"uas"`
	NilaiAkhir *float64 `json:"nilai_akhir,omitempty"`
}

type NilaiBatch struct {
	KelasID         int64       `json:"kelas_id"`
	MataPelajaranID int64       `json:"mata_pelajaran_id"`
	PeriodeID       int64       `json:"periode_id"`
	Items           []NilaiItem `json:"items"`
}
