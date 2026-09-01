"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";

type User = {
  id: number;
  username: string;
  nama: string;
  role: string;
  is_active: boolean;
  guru_id: number | null;
  nama_guru: string | null;
  status_password: "default" | "diganti";
  terakhir_login: string | null;
  wali_kelas: string[];
  login_terkunci: boolean;
  login_gagal: number;
  login_sisa_detik: number;
};

// Penguncian login karena salah password berkali-kali. Hanya ada di memori
// server, jadi daftarnya kosong lagi setiap layanan direstart.
type Blokir = {
  jenis: "akun" | "ip";
  kunci: string;
  nama: string;
  gagal: number;
  sisa_detik: number;
  sisa_teks: string;
};

type Calon = {
  guru_id: number;
  nama: string;
  username: string;
  sudah_ada: boolean;
  akun_username?: string;
};

const ROLE_LABEL: Record<string, string> = {
  superadmin: "Superadmin",
  admin: "Admin",
  kepala: "Kepala",
  guru: "Guru",
};

const PASS_AWAL = "guru123";

export default function UsersMaster() {
  const [items, setItems] = useState<User[]>([]);
  const [blokir, setBlokir] = useState<Blokir[]>([]);
  const [saya, setSaya] = useState<{ nama: string; role: string } | null>(null);
  const [msg, setMsg] = useState("");

  // pratinjau pembuatan akun dari master guru
  const [calon, setCalon] = useState<Calon[] | null>(null);
  const [override, setOverride] = useState<Record<string, string>>({});
  const [roleBaru, setRoleBaru] = useState<Record<string, string>>({});
  const [sibuk, setSibuk] = useState(false);
  const [passDefault, setPassDefault] = useState(PASS_AWAL);

  // tambah user manual — untuk orang yang tidak ada di master guru
  const [formManual, setFormManual] = useState(false);
  const [manual, setManual] = useState({ username: "", nama: "", role: "guru", password: PASS_AWAL });

  // hasil pembuatan / reset — ditampilkan sekali
  const [hasil, setHasil] = useState<{ username: string; nama: string; role: string }[] | null>(null);
  const [passBaru, setPassBaru] = useState<{ nama: string; password: string } | null>(null);

  const load = useCallback(async () => {
    const [u, b] = await Promise.all([api("/users"), api("/login-blokir")]);
    setItems(u);
    setBlokir(b);
  }, []);

  useEffect(() => {
    load().catch((e) => setMsg(e.message));
    api("/auth/me").then((d) => setSaya(d.user)).catch(() => {});
  }, [load]);

  const superadmin = saya?.role === "superadmin";

  async function bukaPratinjau() {
    setSibuk(true);
    try {
      const d = await api("/users/dari-guru");
      setCalon(d.items || []);
      setPassDefault(d.password_default || "guru123");
      setOverride({});
      setRoleBaru({});
    } catch (e: any) { setMsg(e.message); }
    finally { setSibuk(false); }
  }

  async function buatAkun() {
    const belum = (calon || []).filter((c) => !c.sudah_ada);
    if (belum.length === 0) return;
    if (!confirm(`Buat ${belum.length} akun baru dengan password ${passDefault}?`)) return;
    setSibuk(true);
    try {
      const d = await api("/users/dari-guru", {
        method: "POST",
        body: { username: override, role: roleBaru },
      });
      setHasil(d.dibuat || []);
      setMsg(`${d.jumlah_dibuat} akun dibuat, ${d.jumlah_dilewati} dilewati (sudah punya akun).`);
      setCalon(null);
      await load();
    } catch (e: any) { setMsg("❌ " + e.message); }
    finally { setSibuk(false); }
  }

  async function reset(u: User) {
    if (!confirm(`Reset password ${u.nama} (${u.username}) menjadi ${passDefault}?`)) return;
    try {
      const d = await api(`/users/${u.id}/reset-password`, { method: "POST", body: {} });
      setPassBaru({ nama: u.nama, password: d.password });
      await load();
    } catch (e: any) { setMsg("❌ " + e.message); }
  }

  async function bukaKunci(jenis: "akun" | "ip", kunci: string) {
    try {
      const d = await api("/login-blokir/buka", { method: "POST", body: { jenis, kunci } });
      setMsg(d.message);
      await load();
    } catch (e: any) { setMsg("❌ " + e.message); }
  }

  async function nonaktifkan(u: User) {
    if (!confirm(
      `Nonaktifkan akun ${u.username}?\n\nAkun tidak bisa login lagi, tapi seluruh riwayat ` +
      `data yang pernah dibuatnya tetap utuh.`
    )) return;
    try {
      const d = await api(`/users/${u.id}`, { method: "DELETE" });
      setMsg(d.message);
      await load();
    } catch (e: any) { setMsg("❌ " + e.message); }
  }

  async function aktifkan(u: User) {
    try {
      await api(`/users/${u.id}`, {
        method: "PUT",
        body: { nama: u.nama, role: u.role, is_active: true, guru_id: u.guru_id },
      });
      setMsg(`Akun ${u.username} diaktifkan kembali.`);
      await load();
    } catch (e: any) { setMsg("❌ " + e.message); }
  }

  async function buatManual(e: React.FormEvent) {
    e.preventDefault();
    setSibuk(true);
    try {
      await api("/users", { method: "POST", body: manual });
      setMsg(`Akun ${manual.username} dibuat.`);
      setManual({ username: "", nama: "", role: "guru", password: PASS_AWAL });
      setFormManual(false);
      await load();
    } catch (e: any) { setMsg("❌ " + e.message); }
    finally { setSibuk(false); }
  }

  const belumPunya = (calon || []).filter((c) => !c.sudah_ada);

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
      <div className="row" style={{ justifyContent: "space-between", alignItems: "center", flexWrap: "wrap", gap: 10 }}>
        <h1 style={{ margin: 0 }}>User</h1>
        <div className="row" style={{ gap: 8 }}>
          <button className="btn secondary" onClick={() => setFormManual((v) => !v)}>
            + Tambah manual
          </button>
          <button className="btn" onClick={bukaPratinjau} disabled={sibuk}>
            + Buatkan akun dari master guru
          </button>
        </div>
      </div>

      {/* akun untuk orang yang tidak ada di master guru (mis. bendahara) */}
      {formManual && (
        <form className="card" style={{ padding: 14, display: "flex", gap: 10, flexWrap: "wrap", alignItems: "flex-end" }}
          onSubmit={buatManual}>
          <label style={{ display: "flex", flexDirection: "column", gap: 4 }}>
            <span style={{ fontSize: 12 }}>Username</span>
            <input className="input" style={{ width: 170, fontFamily: "monospace" }} required
              value={manual.username} onChange={(e) => setManual({ ...manual, username: e.target.value })} />
          </label>
          <label style={{ display: "flex", flexDirection: "column", gap: 4 }}>
            <span style={{ fontSize: 12 }}>Nama</span>
            <input className="input" style={{ width: 220 }} required
              value={manual.nama} onChange={(e) => setManual({ ...manual, nama: e.target.value })} />
          </label>
          <label style={{ display: "flex", flexDirection: "column", gap: 4 }}>
            <span style={{ fontSize: 12 }}>Peran</span>
            <select className="input" style={{ width: 140 }}
              value={manual.role} onChange={(e) => setManual({ ...manual, role: e.target.value })}>
              <option value="guru">Guru</option>
              {superadmin && <option value="admin">Admin</option>}
              {superadmin && <option value="superadmin">Superadmin</option>}
            </select>
          </label>
          <label style={{ display: "flex", flexDirection: "column", gap: 4 }}>
            <span style={{ fontSize: 12 }}>Password awal</span>
            <input className="input" style={{ width: 150, fontFamily: "monospace" }} required
              value={manual.password} onChange={(e) => setManual({ ...manual, password: e.target.value })} />
          </label>
          <button className="btn" type="submit" disabled={sibuk}>Buat akun</button>
          <button className="btn secondary" type="button" onClick={() => setFormManual(false)}>Batal</button>
        </form>
      )}

      <p className="muted" style={{ margin: 0, fontSize: 13 }}>
        Setiap guru sebaiknya punya akun sendiri agar terlihat siapa yang mengisi absensi dan
        mengubah nilai. Password bawaan <code>{passDefault}</code>.
      </p>

      {msg && <div className="card" style={{ padding: 12 }}>{msg}</div>}

      {/* ===== login yang sedang terkunci ===== */}
      {blokir.length > 0 && (
        <div className="card" style={{ padding: 16, borderLeft: "4px solid var(--danger)" }}>
          <strong>🔒 Login terkunci sementara ({blokir.length})</strong>
          <div className="muted" style={{ fontSize: 12, margin: "6px 0 10px" }}>
            Terkunci otomatis setelah beberapa kali salah password. Kuncinya lepas sendiri setelah
            waktunya habis — tombol di bawah hanya mempercepat, dan tidak mengubah passwordnya.
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            {blokir.map((b) => (
              <div key={`${b.jenis}-${b.kunci}`} className="row"
                style={{ justifyContent: "space-between", alignItems: "center", gap: 10, flexWrap: "wrap" }}>
                <div style={{ fontSize: 13 }}>
                  {b.jenis === "akun" ? (
                    <>
                      <strong>{b.nama || b.kunci}</strong>{" "}
                      <code style={{ fontSize: 12 }}>{b.kunci}</code>
                    </>
                  ) : (
                    <>
                      Alamat jaringan <code style={{ fontSize: 12 }}>{b.kunci}</code>
                    </>
                  )}
                  <span className="muted"> — {b.gagal}× gagal, terbuka sendiri dalam {b.sisa_teks}</span>
                </div>
                <button className="btn secondary" style={{ padding: "4px 10px" }}
                  onClick={() => bukaKunci(b.jenis, b.kunci)}>Buka kunci</button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* ===== hasil reset password — tampil sekali ===== */}
      {passBaru && (
        <div className="card" style={{ padding: 16, borderLeft: "4px solid var(--danger)" }}>
          <div className="row" style={{ justifyContent: "space-between" }}>
            <strong>Password baru untuk {passBaru.nama}</strong>
            <button className="btn secondary" style={{ padding: "4px 10px" }} onClick={() => setPassBaru(null)}>Tutup</button>
          </div>
          <p style={{ fontSize: 22, fontFamily: "monospace", margin: "10px 0" }}>{passBaru.password}</p>
          <div className="muted" style={{ fontSize: 12 }}>
            Catat sekarang. Setelah kotak ini ditutup, password tidak bisa dilihat lagi dari mana pun —
            yang tersimpan di database hanya hash-nya.
          </div>
        </div>
      )}

      {/* ===== hasil pembuatan akun massal ===== */}
      {hasil && hasil.length > 0 && (
        <div className="card" style={{ padding: 16 }}>
          <div className="row" style={{ justifyContent: "space-between" }}>
            <strong>{hasil.length} akun dibuat — password semuanya <code>{passDefault}</code></strong>
            <div className="row" style={{ gap: 6 }}>
              <button className="btn secondary" style={{ padding: "4px 10px" }} onClick={() => window.print()}>🖨️ Cetak</button>
              <button className="btn secondary" style={{ padding: "4px 10px" }} onClick={() => setHasil(null)}>Tutup</button>
            </div>
          </div>
          <div className="table-wrap" style={{ marginTop: 10 }}>
            <table>
              <thead><tr><th>Nama</th><th>Username</th><th>Password</th><th>Peran</th></tr></thead>
              <tbody>
                {hasil.map((h) => (
                  <tr key={h.username}>
                    <td>{h.nama}</td>
                    <td style={{ fontFamily: "monospace" }}>{h.username}</td>
                    <td style={{ fontFamily: "monospace" }}>{passDefault}</td>
                    <td>{ROLE_LABEL[h.role] || h.role}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* ===== pratinjau sebelum membuat ===== */}
      {calon && (
        <div className="card" style={{ padding: 16 }}>
          <div className="row" style={{ justifyContent: "space-between", flexWrap: "wrap", gap: 8 }}>
            <strong>Pratinjau — {belumPunya.length} guru belum punya akun</strong>
            <div className="row" style={{ gap: 6 }}>
              <button className="btn secondary" onClick={() => setCalon(null)}>Batal</button>
              <button className="btn" onClick={buatAkun} disabled={sibuk || belumPunya.length === 0}>
                {sibuk ? "Membuat…" : `Buat ${belumPunya.length} akun`}
              </button>
            </div>
          </div>
          <div className="muted" style={{ fontSize: 12, marginTop: 6 }}>
            Username boleh diubah sebelum dibuat. Guru yang sudah punya akun otomatis dilewati,
            jadi tombol ini aman ditekan lagi setiap ada guru baru.
          </div>
          <div className="table-wrap" style={{ marginTop: 10 }}>
            <table>
              <thead>
                <tr>
                  <th style={{ width: 40 }}>#</th>
                  <th>Nama guru</th>
                  <th style={{ width: 230 }}>Username</th>
                  <th style={{ width: 150 }}>Peran</th>
                </tr>
              </thead>
              <tbody>
                {calon.map((c) => (
                  <tr key={c.guru_id} style={c.sudah_ada ? { opacity: 0.5 } : undefined}>
                    <td>{c.guru_id}</td>
                    <td>{c.nama}</td>
                    <td>
                      {c.sudah_ada ? (
                        <span className="muted" style={{ fontSize: 13 }}>
                          sudah punya: <code>{c.akun_username}</code>
                        </span>
                      ) : (
                        <input className="input" style={{ width: "100%", fontFamily: "monospace" }}
                          value={override[c.guru_id] ?? c.username}
                          onChange={(e) => setOverride({ ...override, [c.guru_id]: e.target.value })} />
                      )}
                    </td>
                    <td>
                      {c.sudah_ada ? "—" : (
                        <select className="input" style={{ width: "100%" }}
                          value={roleBaru[c.guru_id] ?? "guru"}
                          onChange={(e) => setRoleBaru({ ...roleBaru, [c.guru_id]: e.target.value })}>
                          <option value="guru">Guru</option>
                          {superadmin && <option value="admin">Admin</option>}
                          {superadmin && <option value="superadmin">Superadmin</option>}
                        </select>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* ===== daftar user ===== */}
      <div className="card table-wrap" style={{ padding: 0 }}>
        <table>
          <thead>
            <tr>
              <th>Username</th>
              <th>Nama</th>
              <th style={{ width: 110 }}>Peran</th>
              <th style={{ width: 150 }}>Wali kelas</th>
              <th style={{ width: 130 }}>Password</th>
              <th style={{ width: 130 }}>Terakhir login</th>
              <th style={{ width: 90 }}>Status</th>
              <th style={{ width: 190 }}>Aksi</th>
            </tr>
          </thead>
          <tbody>
            {items.map((u) => (
              <tr key={u.id} style={u.is_active ? undefined : { opacity: 0.55 }}>
                <td style={{ fontFamily: "monospace" }}>{u.username}</td>
                <td>
                  {u.nama}
                  {u.nama_guru && <div className="muted" style={{ fontSize: 11 }}>↳ {u.nama_guru}</div>}
                </td>
                <td>
                  <span className={`badge ${u.role === "superadmin" ? "alpha" : u.role === "admin" ? "sakit" : "izin"}`}>
                    {ROLE_LABEL[u.role] || u.role}
                  </span>
                </td>
                <td style={{ fontSize: 12 }}>
                  {u.wali_kelas.length > 0 ? u.wali_kelas.join(", ") : <span className="muted">—</span>}
                </td>
                <td>
                  {u.status_password === "default" ? (
                    <span style={{ fontFamily: "monospace", fontSize: 13 }}>{passDefault}</span>
                  ) : (
                    <span className="muted" style={{ fontSize: 12 }}>Sudah diganti</span>
                  )}
                </td>
                <td style={{ fontSize: 12 }}>
                  {u.terakhir_login ? u.terakhir_login.slice(0, 16).replace("T", " ")
                    : <span className="muted">belum pernah</span>}
                </td>
                <td>
                  <span className={`badge ${u.is_active ? "hadir" : "alpha"}`}>
                    {u.is_active ? "Aktif" : "Nonaktif"}
                  </span>
                  {u.login_terkunci && (
                    <div className="badge alpha" style={{ marginTop: 4 }} title={`${u.login_gagal}× salah password`}>
                      🔒 Terkunci
                    </div>
                  )}
                </td>
                <td>
                  <div className="row" style={{ gap: 6, flexWrap: "wrap" }}>
                    {u.login_terkunci && (
                      <button className="btn" style={{ padding: "4px 8px" }}
                        onClick={() => bukaKunci("akun", u.username)}>Buka kunci</button>
                    )}
                    {superadmin && (
                      <button className="btn secondary" style={{ padding: "4px 8px" }} onClick={() => reset(u)}>
                        Reset pw
                      </button>
                    )}
                    {u.is_active ? (
                      <button className="btn secondary" style={{ padding: "4px 8px", color: "var(--danger)" }}
                        onClick={() => nonaktifkan(u)}>Nonaktifkan</button>
                    ) : (
                      <button className="btn secondary" style={{ padding: "4px 8px" }}
                        onClick={() => aktifkan(u)}>Aktifkan</button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <p className="muted" style={{ fontSize: 12, margin: 0 }}>
        Kolom <strong>Password</strong> hanya bisa menunjukkan apakah password masih bawaan atau
        sudah diganti — isinya sendiri tidak tersimpan dalam bentuk terbaca, bahkan di backup
        sekalipun. Kalau seseorang lupa passwordnya, pakai <strong>Reset pw</strong>. Kalau
        passwordnya benar tapi tertolak karena tadi salah berkali-kali, cukup{" "}
        <strong>Buka kunci</strong> — passwordnya tidak perlu diganti.
      </p>
    </div>
  );
}
