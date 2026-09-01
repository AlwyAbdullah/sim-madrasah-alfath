package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"sim-madrasah/backend/internal/audit"
	"sim-madrasah/backend/internal/auth"
	"sim-madrasah/backend/internal/httpx"
	"sim-madrasah/backend/internal/middleware"
	"sim-madrasah/backend/internal/username"
)

// PasswordDefault dipakai saat akun dibuat dari master guru dan saat direset.
const PasswordDefault = "guru123"

// statusPassword memberi tahu apakah sebuah akun masih memakai password bawaan.
//
// Password TIDAK disimpan — yang tersimpan hash bcrypt yang tidak bisa dibalik.
// Tapi bcrypt bisa MEMERIKSA tebakan, jadi kita masih bisa membedakan "masih
// guru123" dari "sudah diganti sendiri" tanpa menyimpan apa pun dalam bentuk
// terbaca. Inilah pengganti yang aman untuk permintaan "superadmin bisa melihat
// password".
func statusPassword(hash string) string {
	if auth.CheckPassword(hash, PasswordDefault) {
		return "default"
	}
	return "diganti"
}

type userOut struct {
	ID            int64    `json:"id"`
	Username      string   `json:"username"`
	Nama          string   `json:"nama"`
	Role          string   `json:"role"`
	IsActive      bool     `json:"is_active"`
	GuruID        *int64   `json:"guru_id"`
	NamaGuru      *string  `json:"nama_guru"`
	StatusPass    string   `json:"status_password"` // "default" | "diganti"
	TerakhirLogin *string  `json:"terakhir_login"`
	WaliKelas     []string `json:"wali_kelas"`
	// Terkunci karena salah password berkali-kali. Bukan dari database —
	// hitungannya ada di memori dan hilang saat layanan restart.
	LoginTerkunci  bool `json:"login_terkunci"`
	LoginGagal     int  `json:"login_gagal"`
	LoginSisaDetik int  `json:"login_sisa_detik"`
}

// GET /users
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`
		SELECT u.id, u.username, u.nama, u.role, u.is_active, u.guru_id, g.nama,
		       u.password_hash, u.terakhir_login
		FROM users u
		LEFT JOIN guru g ON g.id = u.guru_id
		ORDER BY FIELD(u.role,'superadmin','admin','kepala','guru'), u.username`)
	if err != nil {
		dbErr(w, err)
		return
	}
	defer rows.Close()

	out := []userOut{}
	idx := map[int64]int{}
	for rows.Next() {
		var x userOut
		var guruID sql.NullInt64
		var namaGuru, terakhir sql.NullString
		var hash string
		if err := rows.Scan(&x.ID, &x.Username, &x.Nama, &x.Role, &x.IsActive,
			&guruID, &namaGuru, &hash, &terakhir); err != nil {
			continue
		}
		if guruID.Valid {
			v := guruID.Int64
			x.GuruID = &v
		}
		if namaGuru.Valid {
			v := strings.TrimSpace(namaGuru.String)
			x.NamaGuru = &v
		}
		if terakhir.Valid {
			x.TerakhirLogin = &terakhir.String
		}
		x.StatusPass = statusPassword(hash)
		x.LoginTerkunci, x.LoginGagal, x.LoginSisaDetik = h.Gembok.Status(x.Username)
		x.WaliKelas = []string{}
		idx[x.ID] = len(out)
		out = append(out, x)
	}

	// kelas yang diampu tiap akun — ditampilkan agar terlihat siapa wali apa
	wrows, err := h.DB.Query(`
		SELECT kw.user_id, k.nama FROM kelas_wali kw
		JOIN kelas k ON k.id = kw.kelas_id
		ORDER BY kw.urutan, k.nama`)
	if err == nil {
		for wrows.Next() {
			var uid int64
			var kelas string
			if wrows.Scan(&uid, &kelas) == nil {
				if i, ok := idx[uid]; ok {
					out[i].WaliKelas = append(out[i].WaliKelas, kelas)
				}
			}
		}
		wrows.Close()
	}

	httpx.JSON(w, http.StatusOK, out)
}

type userReq struct {
	Username string `json:"username"`
	Nama     string `json:"nama"`
	Role     string `json:"role"`
	Password string `json:"password"`
	IsActive *bool  `json:"is_active"`
	GuruID   *int64 `json:"guru_id"`
}

func validRole(r string) bool {
	return r == "superadmin" || r == "admin" || r == "guru" || r == "kepala"
}

// hanya superadmin yang boleh membuat/mengubah akun setingkat admin ke atas
func bolehKelolaRole(pelaku, target string) bool {
	if target == "superadmin" || target == "admin" {
		return pelaku == "superadmin"
	}
	return true
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req userReq
	if !decode(w, r, &req) {
		return
	}
	if req.Username == "" || req.Nama == "" || req.Password == "" || !validRole(req.Role) {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST",
			"Username, nama, password, dan role (superadmin/admin/guru/kepala) wajib")
		return
	}
	c := middleware.ClaimsFrom(r)
	if c == nil || !bolehKelolaRole(c.Role, req.Role) {
		httpx.Error(w, http.StatusForbidden, "FORBIDDEN",
			"Hanya superadmin yang boleh membuat akun admin atau superadmin")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "HASH_ERROR", "Gagal memproses password")
		return
	}
	res, err := h.DB.Exec(
		`INSERT INTO users (username, password_hash, nama, role, guru_id) VALUES (?, ?, ?, ?, ?)`,
		req.Username, hash, req.Nama, req.Role, req.GuruID)
	if err != nil {
		dbErr(w, err)
		return
	}
	id, _ := res.LastInsertId()
	audit.Catat(h.DB, r, audit.Entri{
		Aksi: audit.BuatAkun, Entitas: "users", EntitasID: strconv.FormatInt(id, 10),
		Ringkasan: fmt.Sprintf("Membuat akun %s (%s) untuk %s", req.Username, req.Role, req.Nama),
	})
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

// UpdateUser: ubah nama/role/status/penautan guru. Password hanya diubah bila diisi.
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req userReq
	if !decode(w, r, &req) {
		return
	}
	if req.Nama == "" || !validRole(req.Role) {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Nama dan role wajib")
		return
	}

	c := middleware.ClaimsFrom(r)
	roleLama, _, err := h.roleDanAktif(id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "Akun tidak ditemukan")
		return
	}
	// baik role lama maupun role baru harus boleh dikelola oleh pelaku
	if c == nil || !bolehKelolaRole(c.Role, roleLama) || !bolehKelolaRole(c.Role, req.Role) {
		httpx.Error(w, http.StatusForbidden, "FORBIDDEN",
			"Hanya superadmin yang boleh mengubah akun admin atau superadmin")
		return
	}

	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	// menonaktifkan / menurunkan derajat superadmin terakhir akan mengunci semua
	// orang di luar sistem — ditolak sebelum sempat terjadi
	if roleLama == "superadmin" && (!active || req.Role != "superadmin") {
		if err := h.pastikanBukanSuperadminTerakhir(id); err != nil {
			httpx.Error(w, http.StatusBadRequest, "SUPERADMIN_TERAKHIR", err.Error())
			return
		}
	}

	if _, err := h.DB.Exec(
		`UPDATE users SET nama = ?, role = ?, is_active = ?, guru_id = ? WHERE id = ?`,
		req.Nama, req.Role, active, req.GuruID, id); err != nil {
		dbErr(w, err)
		return
	}
	if req.Password != "" {
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "HASH_ERROR", "Gagal memproses password")
			return
		}
		if _, err := h.DB.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, hash, id); err != nil {
			dbErr(w, err)
			return
		}
	}
	audit.Catat(h.DB, r, audit.Entri{
		Aksi: audit.UbahAkun, Entitas: "users", EntitasID: id,
		Ringkasan: fmt.Sprintf("Mengubah akun %s — peran %s, %s", req.Nama, req.Role,
			map[bool]string{true: "aktif", false: "nonaktif"}[active]),
		Rincian: map[string]interface{}{"role_lama": roleLama, "role_baru": req.Role, "aktif": active},
	})
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

func (h *Handler) roleDanAktif(id string) (string, bool, error) {
	var role string
	var aktif bool
	err := h.DB.QueryRow(`SELECT role, is_active FROM users WHERE id = ?`, id).Scan(&role, &aktif)
	return role, aktif, err
}

// pastikanBukanSuperadminTerakhir menolak tindakan yang menyisakan nol superadmin aktif.
func (h *Handler) pastikanBukanSuperadminTerakhir(id string) error {
	var sisa int
	if err := h.DB.QueryRow(`
		SELECT COUNT(*) FROM users
		WHERE role = 'superadmin' AND is_active = 1 AND id <> ?`, id).Scan(&sisa); err != nil {
		return nil // gagal memeriksa -> jangan menghalangi
	}
	if sisa == 0 {
		return fmt.Errorf("Ini superadmin aktif terakhir. Angkat superadmin lain dulu, " +
			"kalau tidak tidak akan ada yang bisa mengelola akun lagi.")
	}
	return nil
}

// jejakData menghitung berapa baris data yang dibuat sebuah akun. Dipakai untuk
// memutuskan boleh-tidaknya hapus permanen: seluruh FK created_by memakai
// ON DELETE SET NULL, sehingga menghapus akun akan MENGOSONGKAN jejak pembuat
// pada data yang pernah dibuatnya — diam-diam, tanpa peringatan.
func (h *Handler) jejakData(id string) map[string]int {
	jejak := map[string]int{}
	tabel := []string{"absensi", "nilai", "spp", "absensi_guru", "catatan", "tugas", "riwayat_tugas"}
	for _, t := range tabel {
		var n int
		if err := h.DB.QueryRow(
			fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE created_by = ?`, t), id).Scan(&n); err == nil && n > 0 {
			jejak[t] = n
		}
	}
	return jejak
}

// DELETE /users/{id}?permanen=1
// Bawaannya menonaktifkan (soft delete) agar riwayat tetap utuh.
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c := middleware.ClaimsFrom(r)
	if c == nil {
		httpx.Error(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak ditemukan")
		return
	}
	if strconv.FormatInt(c.UserID, 10) == id {
		httpx.Error(w, http.StatusBadRequest, "AKUN_SENDIRI",
			"Akun tidak boleh menghapus dirinya sendiri.")
		return
	}
	role, _, err := h.roleDanAktif(id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "Akun tidak ditemukan")
		return
	}
	if !bolehKelolaRole(c.Role, role) {
		httpx.Error(w, http.StatusForbidden, "FORBIDDEN",
			"Hanya superadmin yang boleh menghapus akun admin atau superadmin")
		return
	}
	if role == "superadmin" {
		if err := h.pastikanBukanSuperadminTerakhir(id); err != nil {
			httpx.Error(w, http.StatusBadRequest, "SUPERADMIN_TERAKHIR", err.Error())
			return
		}
	}

	if r.URL.Query().Get("permanen") == "1" {
		if jejak := h.jejakData(id); len(jejak) > 0 {
			bagian := make([]string, 0, len(jejak))
			for t, n := range jejak {
				bagian = append(bagian, fmt.Sprintf("%d %s", n, t))
			}
			httpx.Error(w, http.StatusBadRequest, "PUNYA_JEJAK", fmt.Sprintf(
				"Akun ini sudah membuat %s. Menghapusnya permanen akan menghilangkan "+
					"jejak siapa pembuat data tersebut. Nonaktifkan saja.", strings.Join(bagian, ", ")))
			return
		}
		if _, err := h.DB.Exec(`DELETE FROM users WHERE id = ?`, id); err != nil {
			dbErr(w, err)
			return
		}
		audit.Catat(h.DB, r, audit.Entri{
			Aksi: audit.HapusAkun, Entitas: "users", EntitasID: id,
			Ringkasan: fmt.Sprintf("Menghapus PERMANEN akun id %s (peran %s)", id, role),
		})
		httpx.JSON(w, http.StatusOK, map[string]string{"message": "Akun dihapus permanen"})
		return
	}

	if _, err := h.DB.Exec(`UPDATE users SET is_active = 0 WHERE id = ?`, id); err != nil {
		dbErr(w, err)
		return
	}
	// tendang keluar sekarang juga, jangan menunggu tokennya kedaluwarsa
	h.akhiriSemuaSesiUser(id, c.UserID)
	audit.Catat(h.DB, r, audit.Entri{
		Aksi: audit.HapusAkun, Entitas: "users", EntitasID: id,
		Ringkasan: fmt.Sprintf("Menonaktifkan akun id %s (peran %s)", id, role),
	})
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Akun dinonaktifkan"})
}

// ===================== AKUN DARI MASTER GURU =====================

type calonAkun struct {
	GuruID    int64  `json:"guru_id"`
	Nama      string `json:"nama"`
	Username  string `json:"username"`
	SudahAda  bool   `json:"sudah_ada"`
	AkunNama  string `json:"akun_username,omitempty"`
}

// susunCalon menyiapkan usulan username untuk seluruh guru, menghindari bentrok
// dengan username yang sudah ada maupun antar-usulan.
func (h *Handler) susunCalon() ([]calonAkun, error) {
	terpakai := map[string]bool{}
	urows, err := h.DB.Query(`SELECT LOWER(username) FROM users`)
	if err != nil {
		return nil, err
	}
	for urows.Next() {
		var u string
		if urows.Scan(&u) == nil {
			terpakai[u] = true
		}
	}
	urows.Close()

	punyaAkun := map[int64]string{}
	arows, err := h.DB.Query(`SELECT guru_id, username FROM users WHERE guru_id IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	for arows.Next() {
		var gid int64
		var un string
		if arows.Scan(&gid, &un) == nil {
			punyaAkun[gid] = un
		}
	}
	arows.Close()

	rows, err := h.DB.Query(`SELECT id, nama FROM guru ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []calonAkun{}
	for rows.Next() {
		var c calonAkun
		if rows.Scan(&c.GuruID, &c.Nama) != nil {
			continue
		}
		c.Nama = strings.TrimSpace(c.Nama)
		if un, ada := punyaAkun[c.GuruID]; ada {
			c.SudahAda = true
			c.AkunNama = un
			c.Username = un
		} else {
			c.Username = username.Pilih(c.Nama, terpakai)
			if c.Username != "" {
				terpakai[c.Username] = true
			}
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GET /users/dari-guru — pratinjau: guru mana yang belum punya akun & usulan usernamenya.
func (h *Handler) PreviewAkunGuru(w http.ResponseWriter, r *http.Request) {
	calon, err := h.susunCalon()
	if err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"password_default": PasswordDefault,
		"items":            calon,
	})
}

type buatAkunReq struct {
	// Username boleh ditimpa admin lewat pratinjau. Kunci = guru_id.
	Username map[string]string `json:"username"`
	// Role per guru (bawaan "guru"). Kunci = guru_id.
	Role map[string]string `json:"role"`
}

// POST /users/dari-guru — buat akun untuk guru yang belum punya.
// Idempoten: guru yang sudah punya akun dilewati, jadi aman dipanggil ulang
// setiap ada guru baru.
func (h *Handler) BuatAkunDariGuru(w http.ResponseWriter, r *http.Request) {
	var req buatAkunReq
	if r.ContentLength > 0 && !decode(w, r, &req) {
		return
	}
	c := middleware.ClaimsFrom(r)
	if c == nil {
		httpx.Error(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak ditemukan")
		return
	}

	calon, err := h.susunCalon()
	if err != nil {
		dbErr(w, err)
		return
	}

	hash, err := auth.HashPassword(PasswordDefault)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "HASH_ERROR", "Gagal memproses password")
		return
	}

	type hasil struct {
		GuruID   int64  `json:"guru_id"`
		Nama     string `json:"nama"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}

	// Susun dulu SELURUH rencana dan periksa wewenangnya sebelum satu baris pun
	// ditulis. Kalau divalidasi sambil jalan, permintaan yang ditolak di tengah
	// meninggalkan sebagian akun terbuat — membingungkan dan sulit dibereskan.
	rencana := []hasil{}
	dilewati := 0
	for _, k := range calon {
		if k.SudahAda {
			dilewati++
			continue
		}
		kunci := strconv.FormatInt(k.GuruID, 10)

		un := k.Username
		if v, ok := req.Username[kunci]; ok && strings.TrimSpace(v) != "" {
			un = strings.ToLower(strings.TrimSpace(v))
		}
		if un == "" {
			continue // nama guru kosong -> tidak bisa disusun
		}

		role := "guru"
		if v, ok := req.Role[kunci]; ok && validRole(v) {
			role = v
		}
		if !bolehKelolaRole(c.Role, role) {
			httpx.Error(w, http.StatusForbidden, "FORBIDDEN", fmt.Sprintf(
				"Hanya superadmin yang boleh membuat akun %s (diminta untuk %s). "+
					"Tidak ada akun yang dibuat.", role, strings.TrimSpace(k.Nama)))
			return
		}
		rencana = append(rencana, hasil{GuruID: k.GuruID, Nama: k.Nama, Username: un, Role: role})
	}

	tx, err := h.DB.Begin()
	if err != nil {
		dbErr(w, err)
		return
	}
	defer tx.Rollback()

	dibuat := []hasil{}
	for _, p := range rencana {
		if _, err := tx.Exec(`
			INSERT INTO users (username, password_hash, nama, role, guru_id)
			VALUES (?, ?, ?, ?, ?)`, p.Username, hash, p.Nama, p.Role, p.GuruID); err != nil {
			httpx.Error(w, http.StatusBadRequest, "GAGAL_BUAT", fmt.Sprintf(
				"Gagal membuat akun %q untuk %s: %v. Tidak ada akun yang dibuat.",
				p.Username, strings.TrimSpace(p.Nama), err))
			return
		}
		dibuat = append(dibuat, p)
	}
	if err := tx.Commit(); err != nil {
		dbErr(w, err)
		return
	}

	if len(dibuat) > 0 {
		nama := make([]string, 0, len(dibuat))
		for _, d := range dibuat {
			nama = append(nama, d.Username)
		}
		audit.Catat(h.DB, r, audit.Entri{
			Aksi: audit.BuatAkun, Entitas: "users",
			Ringkasan: fmt.Sprintf("Membuat %d akun dari master guru", len(dibuat)),
			Rincian:   map[string]interface{}{"username": nama},
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"dibuat":           dibuat,
		"jumlah_dibuat":    len(dibuat),
		"jumlah_dilewati":  dilewati,
		"password_default": PasswordDefault,
	})
}

// ===================== PASSWORD =====================

type resetPassReq struct {
	Password string `json:"password"` // kosong = kembali ke PasswordDefault
}

// POST /users/{id}/reset-password — hanya superadmin.
// Password baru dikembalikan SEKALI di respons agar bisa diberitahukan ke
// pemiliknya; setelah itu tidak bisa dilihat lagi dari mana pun.
func (h *Handler) ResetPasswordUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req resetPassReq
	if r.ContentLength > 0 && !decode(w, r, &req) {
		return
	}
	baru := strings.TrimSpace(req.Password)
	if baru == "" {
		baru = PasswordDefault
	}
	if len(baru) < 6 {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Password minimal 6 karakter")
		return
	}
	var namaAkun string
	if err := h.DB.QueryRow(`SELECT username FROM users WHERE id = ?`, id).Scan(&namaAkun); err != nil {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "Akun tidak ditemukan")
		return
	}
	hash, err := auth.HashPassword(baru)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "HASH_ERROR", "Gagal memproses password")
		return
	}
	if _, err := h.DB.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, hash, id); err != nil {
		dbErr(w, err)
		return
	}
	// Password lama tidak berlaku lagi, jadi sesi yang masih memakainya juga
	// tidak boleh berlaku — kalau resetnya karena password bocor, membiarkan
	// sesi lama hidup membuat resetnya sia-sia.
	var pemutus interface{}
	if c := middleware.ClaimsFrom(r); c != nil {
		pemutus = c.UserID
	}
	h.akhiriSemuaSesiUser(id, pemutus)
	// Reset password paling sering diminta justru karena orangnya sudah salah
	// berkali-kali. Tanpa ini passwordnya baru tapi pintunya masih terkunci.
	h.Gembok.BukaAkun(namaAkun)

	audit.Catat(h.DB, r, audit.Entri{
		Aksi: audit.ResetPassword, Entitas: "users", EntitasID: id,
		Ringkasan: fmt.Sprintf("Mereset password akun id %s", id),
	})
	httpx.JSON(w, http.StatusOK, map[string]string{
		"password": baru,
		"message":  "Password direset. Catat sekarang — nilainya tidak bisa dilihat lagi setelah layar ini ditutup.",
	})
}

type gantiPassReq struct {
	PasswordLama string `json:"password_lama"`
	PasswordBaru string `json:"password_baru"`
}

// POST /auth/ganti-password — mengganti password sendiri (semua peran).
func (h *Handler) GantiPasswordSaya(w http.ResponseWriter, r *http.Request) {
	var req gantiPassReq
	if !decode(w, r, &req) {
		return
	}
	c := middleware.ClaimsFrom(r)
	if c == nil {
		httpx.Error(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak ditemukan")
		return
	}
	if len(req.PasswordBaru) < 6 {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Password baru minimal 6 karakter")
		return
	}

	var hash string
	if err := h.DB.QueryRow(`SELECT password_hash FROM users WHERE id = ?`, c.UserID).Scan(&hash); err != nil {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "Akun tidak ditemukan")
		return
	}
	if !auth.CheckPassword(hash, req.PasswordLama) {
		httpx.Error(w, http.StatusBadRequest, "PASSWORD_SALAH", "Password lama salah")
		return
	}
	baru, err := auth.HashPassword(req.PasswordBaru)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "HASH_ERROR", "Gagal memproses password")
		return
	}
	if _, err := h.DB.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, baru, c.UserID); err != nil {
		dbErr(w, err)
		return
	}
	audit.Catat(h.DB, r, audit.Entri{
		Aksi: audit.GantiPassword, Entitas: "users", EntitasID: strconv.FormatInt(c.UserID, 10),
		Ringkasan: fmt.Sprintf("%s mengganti passwordnya sendiri", c.Username),
	})
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Password berhasil diganti"})
}
