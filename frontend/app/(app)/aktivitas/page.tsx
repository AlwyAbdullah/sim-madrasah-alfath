"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";

type Entri = {
  id: number;
  user_id: number | null;
  username: string;
  nama: string;
  aksi: string;
  entitas: string | null;
  entitas_id: string | null;
  ringkasan: string;
  rincian: string | null;
  ip: string | null;
  created_at: string;
};

type Pelaku = { user_id: number; nama: string };

const AKSI_LABEL: Record<string, string> = {
  login: "Login",
  logout: "Logout",
  simpan_absensi: "Absensi",
  simpan_nilai: "Nilai",
  simpan_tugas: "Tugas",
  simpan_spp: "SPP",
  kembalikan_spp: "SPP dikembalikan",
  tambah_santri: "Santri ditambah",
  ubah_santri: "Santri diubah",
  hapus_santri: "Santri dinonaktifkan",
  naik_kelas: "Naik kelas",
  ubah_master: "Master data",
  buat_akun: "Akun dibuat",
  ubah_akun: "Akun diubah",
  hapus_akun: "Akun dihapus",
  reset_password: "Reset password",
  ganti_password: "Ganti password",
  ubah_wali_kelas: "Wali kelas",
  ubah_pengaturan: "Pengaturan",
};

// aksi yang perlu menonjol saat disapu mata
const WARNA: Record<string, string> = {
  reset_password: "alpha",
  hapus_akun: "alpha",
  naik_kelas: "alpha",
  buat_akun: "sakit",
  ubah_akun: "sakit",
  ganti_password: "sakit",
  login: "izin",
  logout: "izin",
};

const HALAMAN = 100;

export default function AktivitasPage() {
  const [items, setItems] = useState<Entri[]>([]);
  const [aksiTersedia, setAksi] = useState<string[]>([]);
  const [pelaku, setPelaku] = useState<Pelaku[]>([]);
  const [msg, setMsg] = useState("");
  const [muat, setMuat] = useState(false);
  const [buka, setBuka] = useState<number | null>(null);

  const [fUser, setFUser] = useState("");
  const [fAksi, setFAksi] = useState("");
  const [fDari, setFDari] = useState("");
  const [fSampai, setFSampai] = useState("");
  const [fCari, setFCari] = useState("");
  const [offset, setOffset] = useState(0);

  const load = useCallback(async () => {
    setMuat(true);
    try {
      const p = new URLSearchParams();
      if (fUser) p.set("user_id", fUser);
      if (fAksi) p.set("aksi", fAksi);
      if (fDari) p.set("dari", fDari);
      if (fSampai) p.set("sampai", fSampai);
      if (fCari) p.set("cari", fCari);
      p.set("limit", String(HALAMAN));
      p.set("offset", String(offset));
      const d = await api(`/aktivitas?${p.toString()}`);
      setItems(d.items || []);
      setAksi(d.aksi || []);
      setPelaku(d.pelaku || []);
    } catch (e: any) { setMsg(e.message); }
    finally { setMuat(false); }
  }, [fUser, fAksi, fDari, fSampai, fCari, offset]);

  useEffect(() => { load(); }, [load]);

  // ganti saringan berarti kembali ke halaman pertama
  function saring(set: (v: string) => void) {
    return (v: string) => { setOffset(0); set(v); };
  }

  function reset() {
    setOffset(0); setFUser(""); setFAksi(""); setFDari(""); setFSampai(""); setFCari("");
  }

  const adaSaringan = fUser || fAksi || fDari || fSampai || fCari;

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
      <h1 style={{ margin: 0 }}>Aktivitas</h1>
      <p className="muted" style={{ margin: 0, fontSize: 13 }}>
        Lini masa perubahan sistem. Dicatat <strong>per tindakan</strong>, bukan per baris data —
        menyimpan absensi satu kelas muncul sebagai satu entri, bukan sepuluh.
      </p>

      <div className="card" style={{ padding: 12, display: "flex", gap: 8, flexWrap: "wrap", alignItems: "flex-end" }}>
        <label style={{ display: "flex", flexDirection: "column", gap: 3 }}>
          <span style={{ fontSize: 12 }} className="muted">Pelaku</span>
          <select className="input" style={{ width: 190 }} value={fUser}
            onChange={(e) => saring(setFUser)(e.target.value)}>
            <option value="">— semua —</option>
            {pelaku.map((p) => <option key={p.user_id} value={p.user_id}>{p.nama}</option>)}
          </select>
        </label>
        <label style={{ display: "flex", flexDirection: "column", gap: 3 }}>
          <span style={{ fontSize: 12 }} className="muted">Jenis</span>
          <select className="input" style={{ width: 170 }} value={fAksi}
            onChange={(e) => saring(setFAksi)(e.target.value)}>
            <option value="">— semua —</option>
            {aksiTersedia.map((a) => <option key={a} value={a}>{AKSI_LABEL[a] || a}</option>)}
          </select>
        </label>
        <label style={{ display: "flex", flexDirection: "column", gap: 3 }}>
          <span style={{ fontSize: 12 }} className="muted">Dari</span>
          <input className="input" type="date" style={{ width: 150 }} value={fDari}
            onChange={(e) => saring(setFDari)(e.target.value)} />
        </label>
        <label style={{ display: "flex", flexDirection: "column", gap: 3 }}>
          <span style={{ fontSize: 12 }} className="muted">Sampai</span>
          <input className="input" type="date" style={{ width: 150 }} value={fSampai}
            onChange={(e) => saring(setFSampai)(e.target.value)} />
        </label>
        <label style={{ display: "flex", flexDirection: "column", gap: 3 }}>
          <span style={{ fontSize: 12 }} className="muted">Cari</span>
          <input className="input" style={{ width: 190 }} placeholder="kata dalam ringkasan"
            value={fCari} onChange={(e) => saring(setFCari)(e.target.value)} />
        </label>
        {adaSaringan && <button className="btn secondary" onClick={reset}>Bersihkan</button>}
        <button className="btn secondary" onClick={() => load()}>↻ Muat ulang</button>
      </div>

      {msg && <div className="card" style={{ padding: 12 }}>{msg}</div>}

      {muat ? (
        <p className="muted">Memuat…</p>
      ) : items.length === 0 ? (
        <p className="muted">
          {adaSaringan ? "Tidak ada aktivitas yang cocok dengan saringan." : "Belum ada aktivitas tercatat."}
        </p>
      ) : (
        <div className="card table-wrap" style={{ padding: 0 }}>
          <table>
            <thead>
              <tr>
                <th style={{ width: 145 }}>Waktu</th>
                <th style={{ width: 160 }}>Pelaku</th>
                <th style={{ width: 150 }}>Jenis</th>
                <th>Keterangan</th>
                <th style={{ width: 70 }}></th>
              </tr>
            </thead>
            <tbody>
              {items.map((e) => (
                <tr key={e.id}>
                  <td style={{ fontSize: 12 }}>{e.created_at.slice(0, 16).replace("T", " ")}</td>
                  <td style={{ fontSize: 13 }}>
                    {e.nama}
                    <div className="muted" style={{ fontSize: 11, fontFamily: "monospace" }}>{e.username}</div>
                  </td>
                  <td>
                    <span className={`badge ${WARNA[e.aksi] || "hadir"}`} style={{ fontSize: 11 }}>
                      {AKSI_LABEL[e.aksi] || e.aksi}
                    </span>
                  </td>
                  <td style={{ fontSize: 13 }}>{e.ringkasan}</td>
                  <td>
                    {(e.rincian || e.ip) && (
                      <button className="btn secondary" style={{ padding: "2px 8px", fontSize: 12 }}
                        onClick={() => setBuka(buka === e.id ? null : e.id)}>
                        {buka === e.id ? "Tutup" : "Rincian"}
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {buka !== null && (() => {
        const e = items.find((x) => x.id === buka);
        if (!e) return null;
        return (
          <div className="card" style={{ padding: 14 }}>
            <div className="row" style={{ justifyContent: "space-between" }}>
              <strong>{e.ringkasan}</strong>
              <button className="btn secondary" style={{ padding: "4px 10px" }} onClick={() => setBuka(null)}>Tutup</button>
            </div>
            <div className="muted" style={{ fontSize: 12, marginTop: 4 }}>
              {e.nama} · {e.created_at.replace("T", " ")}{e.ip ? ` · dari ${e.ip}` : ""}
              {e.entitas ? ` · ${e.entitas}${e.entitas_id ? ` #${e.entitas_id}` : ""}` : ""}
            </div>
            {e.rincian && (
              <pre style={{ whiteSpace: "pre-wrap", fontSize: 12, background: "#f8fafc", padding: 10, borderRadius: 8, marginTop: 10 }}>
                {(() => { try { return JSON.stringify(JSON.parse(e.rincian), null, 2); } catch { return e.rincian; } })()}
              </pre>
            )}
          </div>
        );
      })()}

      <div className="row" style={{ gap: 8, alignItems: "center" }}>
        <button className="btn secondary" disabled={offset === 0}
          onClick={() => setOffset(Math.max(0, offset - HALAMAN))}>← Sebelumnya</button>
        <span className="muted" style={{ fontSize: 12 }}>
          {offset + 1}–{offset + items.length}
        </span>
        <button className="btn secondary" disabled={items.length < HALAMAN}
          onClick={() => setOffset(offset + HALAMAN)}>Berikutnya →</button>
      </div>
    </div>
  );
}
