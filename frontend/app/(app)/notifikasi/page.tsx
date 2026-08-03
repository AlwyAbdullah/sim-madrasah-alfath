"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";

type Item = {
  id: number;
  nama: string;
  jenis: string;
  ref_tanggal: string | null;
  tujuan: string;
  pesan: string;
  status: string;
  percobaan: number;
  catatan: string | null;
  dikirim_at: string | null;
  created_at: string;
};

const WARNA: Record<string, string> = {
  pending: "sakit",     // kuning
  terkirim: "hadir",    // hijau
  gagal: "alpha",       // merah
  batal: "izin",        // biru
};

export default function NotifikasiPage() {
  const [items, setItems] = useState<Item[]>([]);
  const [ringkasan, setRingkasan] = useState<Record<string, number>>({});
  const [status, setStatus] = useState("");
  const [msg, setMsg] = useState("");
  const [lihat, setLihat] = useState<Item | null>(null);

  const load = useCallback(async () => {
    const d = await api(`/notifikasi${status ? `?status=${status}` : ""}`);
    setItems(d.items || []);
    setRingkasan(d.ringkasan || {});
  }, [status]);

  useEffect(() => { load().catch((e) => setMsg(e.message)); }, [load]);

  async function aksi(id: number, jenis: "ulang" | "batal") {
    try {
      await api(`/notifikasi/${id}/${jenis}`, { method: "POST" });
      setMsg(jenis === "ulang" ? "Dikembalikan ke antrean." : "Dibatalkan.");
      await load();
    } catch (e: any) { setMsg(e.message); }
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
      <h1 style={{ margin: 0 }}>Notifikasi WhatsApp</h1>
      <p className="muted" style={{ margin: 0, fontSize: 13 }}>
        Pesan otomatis ke orang tua saat santri <strong>alpha</strong>. Backend menyusun antrean di sini,
        lalu bot WhatsApp mengambil &amp; mengirimnya. Jika status absensi dikoreksi menjadi bukan alpha,
        pesan yang belum terkirim otomatis dibatalkan.
      </p>

      <div className="row" style={{ fontSize: 13 }}>
        <span className="badge hadir">Terkirim {ringkasan.terkirim || 0}</span>
        <span className="badge sakit">Menunggu {ringkasan.pending || 0}</span>
        <span className="badge alpha">Gagal {ringkasan.gagal || 0}</span>
        <span className="badge izin">Batal {ringkasan.batal || 0}</span>
        <select className="input" value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value="">— semua status —</option>
          <option value="pending">Menunggu</option>
          <option value="terkirim">Terkirim</option>
          <option value="gagal">Gagal</option>
          <option value="batal">Batal</option>
        </select>
        <button className="btn secondary" onClick={() => load().catch(() => {})}>↻ Muat ulang</button>
      </div>

      {msg && <div className="card" style={{ padding: 12 }}>{msg}</div>}

      {items.length === 0 ? (
        <p className="muted">Belum ada notifikasi. Pesan akan muncul otomatis saat ada santri ditandai <strong>alpha</strong> (dan nomor orang tuanya terisi).</p>
      ) : (
        <div className="card table-wrap" style={{ padding: 0 }}>
          <table>
            <thead>
              <tr>
                <th style={{ width: 150 }}>Santri</th>
                <th style={{ width: 100 }}>Tanggal</th>
                <th style={{ width: 120 }}>Tujuan</th>
                <th>Pesan</th>
                <th style={{ width: 90 }}>Status</th>
                <th style={{ width: 150 }}>Aksi</th>
              </tr>
            </thead>
            <tbody>
              {items.map((it) => (
                <tr key={it.id}>
                  <td>{it.nama}</td>
                  <td>{it.ref_tanggal || "-"}</td>
                  <td style={{ fontSize: 13 }}>{it.tujuan}</td>
                  <td>
                    <div style={{ fontSize: 12, color: "var(--muted)", maxWidth: 380, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                      {it.pesan.replace(/\n+/g, " ").slice(0, 90)}…
                    </div>
                    {it.catatan && <div style={{ fontSize: 12, color: "var(--danger)" }}>⚠ {it.catatan}</div>}
                  </td>
                  <td>
                    <span className={`badge ${WARNA[it.status] || ""}`}>{it.status}</span>
                    {it.percobaan > 0 && <div className="muted" style={{ fontSize: 11 }}>{it.percobaan}× coba</div>}
                  </td>
                  <td>
                    <div className="row" style={{ gap: 6 }}>
                      <button className="btn secondary" style={{ padding: "4px 8px" }} onClick={() => setLihat(it)}>Lihat</button>
                      {it.status !== "terkirim" && it.status !== "pending" && (
                        <button className="btn secondary" style={{ padding: "4px 8px" }} onClick={() => aksi(it.id, "ulang")}>Kirim ulang</button>
                      )}
                      {it.status === "pending" && (
                        <button className="btn secondary" style={{ padding: "4px 8px", color: "var(--danger)" }} onClick={() => aksi(it.id, "batal")}>Batal</button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {lihat && (
        <div className="card" style={{ padding: 16 }}>
          <div className="row" style={{ justifyContent: "space-between" }}>
            <strong>Isi pesan — {lihat.nama}</strong>
            <button className="btn secondary" style={{ padding: "4px 10px" }} onClick={() => setLihat(null)}>Tutup</button>
          </div>
          <pre style={{ whiteSpace: "pre-wrap", fontFamily: "inherit", fontSize: 14, background: "#f8fafc", padding: 12, borderRadius: 8, marginTop: 10 }}>
            {lihat.pesan}
          </pre>
          <div className="muted" style={{ fontSize: 12 }}>
            Ke: {lihat.tujuan} · dibuat {lihat.created_at}{lihat.dikirim_at ? ` · terkirim ${lihat.dikirim_at}` : ""}
          </div>
        </div>
      )}
    </div>
  );
}
