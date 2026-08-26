package middleware

import (
	"net/http"
	"testing"
)

func permintaan(remote string, header map[string]string) *http.Request {
	r, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	r.RemoteAddr = remote
	for k, v := range header {
		r.Header.Set(k, v)
	}
	return r
}

// Port harus dibuang. Tanpa ini setiap sambungan TCP baru menghasilkan kunci
// berbeda dan pembatas percobaan login tidak pernah tercapai.
func TestPortDibuangDariRemoteAddr(t *testing.T) {
	a := clientIP(permintaan("203.0.113.9:51000", nil))
	b := clientIP(permintaan("203.0.113.9:51001", nil))
	if a != "203.0.113.9" || a != b {
		t.Fatalf("kunci harus sama tanpa port: %q vs %q", a, b)
	}
}

// Permintaan langsung ke backend (bukan lewat nginx) tidak boleh bisa mengarang
// identitasnya sendiri lewat header.
func TestHeaderDariLuarTidakDipercaya(t *testing.T) {
	kunci := clientIP(permintaan("203.0.113.9:51000", map[string]string{
		"X-Forwarded-For": "1.2.3.4",
		"X-Real-IP":       "5.6.7.8",
	}))
	if kunci != "203.0.113.9" {
		t.Fatalf("header dari klien luar tidak boleh dipakai, dapat %q", kunci)
	}
}

// Dari nginx (localhost), X-Real-IP dipercaya karena nginx menimpanya sendiri.
func TestXRealIPDipercayaDariLocalhost(t *testing.T) {
	kunci := clientIP(permintaan("127.0.0.1:40000", map[string]string{
		"X-Real-IP": "203.0.113.9",
	}))
	if kunci != "203.0.113.9" {
		t.Fatalf("mau 203.0.113.9, dapat %q", kunci)
	}
}

// nginx memakai $proxy_add_x_forwarded_for yang MENAMBAHKAN IP asli di akhir.
// Awalan kiriman klien harus diabaikan — kalau tidak, penyerang tinggal
// mengganti-ganti awalan untuk mendapat jatah baru tiap permintaan.
func TestAwalanXFFKiromanKlienDiabaikan(t *testing.T) {
	k1 := clientIP(permintaan("127.0.0.1:40000", map[string]string{
		"X-Forwarded-For": "9.9.9.1, 203.0.113.9",
	}))
	k2 := clientIP(permintaan("127.0.0.1:40001", map[string]string{
		"X-Forwarded-For": "9.9.9.2, 203.0.113.9",
	}))
	if k1 != "203.0.113.9" || k1 != k2 {
		t.Fatalf("awalan palsu tidak boleh membuat kunci baru: %q vs %q", k1, k2)
	}
}

func TestIPv6Loopback(t *testing.T) {
	kunci := clientIP(permintaan("[::1]:40000", map[string]string{"X-Real-IP": "203.0.113.9"}))
	if kunci != "203.0.113.9" {
		t.Fatalf("::1 harus dianggap localhost, dapat %q", kunci)
	}
}
