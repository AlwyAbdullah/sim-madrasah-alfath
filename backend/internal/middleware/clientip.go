package middleware

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP mengembalikan IP klien yang bisa dipercaya. Diekspor karena
// pencatatan aktivitas (internal/audit), daftar sesi, dan penguncian login
// (internal/gembok) harus memakai penentuan yang sama persis — kalau berbeda,
// salah satunya pasti keliru.
func ClientIP(r *http.Request) string { return clientIP(r) }

// clientIP menentukan identitas pemanggil.
//
// Header X-Forwarded-For / X-Real-IP HANYA dipercaya bila sambungannya datang
// dari localhost, yaitu dari nginx di mesin yang sama. Alasannya:
//
//   - nginx memakai $proxy_add_x_forwarded_for, yang MENAMBAHKAN IP asli ke
//     nilai kiriman klien. Kalau seluruh isi header dipakai sebagai kunci,
//     penyerang tinggal mengganti-ganti awalannya untuk mendapat jatah baru
//     tiap permintaan — pembatasnya jadi tidak ada gunanya.
//   - backend juga bisa dihubungi langsung (tanpa lewat nginx), dan dalam
//     keadaan itu header tersebut sepenuhnya karangan klien.
//
// Untuk sambungan dari nginx, dipakai entri TERAKHIR pada X-Forwarded-For —
// itulah yang ditambahkan nginx sendiri dan tidak bisa dipalsukan klien.
func clientIP(r *http.Request) string {
	host := hostSaja(r.RemoteAddr)

	if isLoopback(host) {
		if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
			return xri
		}
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			bagian := strings.Split(xff, ",")
			return strings.TrimSpace(bagian[len(bagian)-1])
		}
	}
	// port dibuang: tanpa ini tiap sambungan TCP baru punya identitas berbeda,
	// sehingga hitungan percobaan gagal selalu mulai dari nol
	return host
}

func hostSaja(addr string) string {
	if h, _, err := net.SplitHostPort(addr); err == nil {
		return h
	}
	return addr
}

func isLoopback(host string) bool {
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return host == "localhost"
}
