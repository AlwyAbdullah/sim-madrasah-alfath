"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";

type Kelas = { id: number; nama: string; aktif?: boolean };
type Santri = { id: number; nama: string; nis?: string };

// Saran kelas tujuan dari nama kelas asal (Sifr A→Kelas 1, Sifr B→Kelas 2,
// Kelas N→Kelas N+1, Kelas 6→Alumni).
function suggestDestId(sourceName: string, list: Kelas[]): string {
  const n = (sourceName || "").trim().toLowerCase();
  let target = "";
  if (n === "sifr a") target = "kelas 1";
  else if (n === "sifr b") target = "kelas 2";
  else {
    const m = n.match(/kelas\s*(\d+)/);
    if (m) {
      const k = parseInt(m[1], 10);
      target = k >= 6 ? "alumni" : `kelas ${k + 1}`;
    }
  }
  if (!target) return "";
  const found = list.find((k) => k.nama.trim().toLowerCase() === target);
  return found ? String(found.id) : "";
}

export default function NaikKelasPage() {
  const [kelas, setKelas] = useState<Kelas[]>([]);
  const [fromId, setFromId] = useState("");
  const [toId, setToId] = useState("");
  const [santri, setSantri] = useState<Santri[]>([]);
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => { api("/kelas").then(setKelas).catch(() => {}); }, []);

  useEffect(() => {
    setMsg("");
    if (!fromId) { setSantri([]); setToId(""); return; }
    api(`/santri?kelas_id=${fromId}`).then(setSantri).catch(() => setSantri([]));
    const src = kelas.find((k) => String(k.id) === fromId);
    setToId(src ? suggestDestId(src.nama, kelas) : "");
    // eslint-disable-next-line
  }, [fromId]);

  const fromNama = kelas.find((k) => String(k.id) === fromId)?.nama || "";
  const toNama = kelas.find((k) => String(k.id) === toId)?.nama || "";

  async function jalankan() {
    if (!fromId || !toId) { setMsg("Pilih kelas asal dan tujuan."); return; }
    if (fromId === toId) { setMsg("Kelas asal dan tujuan tidak boleh sama."); return; }
    if (!confirm(`Naikkan ${santri.length} santri dari "${fromNama}" ke "${toNama}"?\n\nNilai, absensi, dan rapor periode lama TETAP aman — hanya kelas santri sekarang yang berubah.`)) return;
    setBusy(true);
    setMsg("");
    try {
      const d = await api("/santri/naik-kelas", {
        method: "POST",
        body: { from_kelas_id: Number(fromId), to_kelas_id: Number(toId) },
      });
      setMsg(`Berhasil: ${d.moved} santri dipindah dari "${fromNama}" ke "${toNama}".`);
      setSantri([]); setFromId(""); setToId("");
    } catch (e: any) {
      setMsg(e.message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
      <h1 style={{ margin: 0 }}>Naik Kelas</h1>
      <p className="muted" style={{ margin: 0 }}>
        Pindahkan seluruh santri satu kelas ke kelas berikutnya. <strong>Nilai, absensi, dan rapor periode lama tetap aman</strong> — hanya kelas santri saat ini yang berubah. Sebaiknya dijalankan setelah nilai semester selesai &amp; periode/tahun ajaran baru dibuat.
      </p>

      <div className="row" style={{ alignItems: "flex-end" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          <label style={{ fontSize: 13, fontWeight: 600 }}>Kelas asal</label>
          <select className="input" value={fromId} onChange={(e) => setFromId(e.target.value)}>
            <option value="">— pilih —</option>
            {kelas.map((k) => <option key={k.id} value={k.id}>{k.nama}</option>)}
          </select>
        </div>
        <div style={{ fontSize: 22, paddingBottom: 6 }}>→</div>
        <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          <label style={{ fontSize: 13, fontWeight: 600 }}>Kelas tujuan</label>
          <select className="input" value={toId} onChange={(e) => setToId(e.target.value)}>
            <option value="">— pilih —</option>
            {kelas.map((k) => <option key={k.id} value={k.id}>{k.nama}</option>)}
          </select>
        </div>
        <button className="btn" onClick={jalankan} disabled={busy || !fromId || !toId || !santri.length}>
          {busy ? "Memproses..." : "⬆ Naikkan Kelas"}
        </button>
      </div>

      {msg && <div className="card" style={{ padding: 12 }}>{msg}</div>}

      {fromId && (
        <div className="card" style={{ padding: 0 }}>
          <div style={{ padding: 12, borderBottom: "1px solid var(--border)", fontSize: 14 }}>
            <strong>{santri.length}</strong> santri di <strong>{fromNama}</strong> akan dipindah ke <strong>{toNama || "—"}</strong>
          </div>
          <div className="table-wrap">
            <table>
              <thead><tr><th style={{ width: 40 }}>No</th><th>Nama</th></tr></thead>
              <tbody>
                {santri.map((s, i) => <tr key={s.id}><td>{i + 1}</td><td>{s.nama}</td></tr>)}
                {santri.length === 0 && <tr><td colSpan={2} className="muted" style={{ padding: 14 }}>Tidak ada santri aktif di kelas ini.</td></tr>}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
