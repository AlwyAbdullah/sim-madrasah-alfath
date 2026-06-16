"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";

type Kelas = { id: number; nama: string; aktif: boolean };
type Mapel = { id: number; nama: string; kode: string };
type Row = { mata_pelajaran_id: number; nama: string; checked: boolean; kitab: string };

export default function PelajaranKelasPage() {
  const [kelas, setKelas] = useState<Kelas[]>([]);
  const [kelasId, setKelasId] = useState("");
  const [rows, setRows] = useState<Row[]>([]);
  const [msg, setMsg] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => { api("/kelas").then(setKelas).catch(() => {}); }, []);

  async function load() {
    if (!kelasId) { setRows([]); return; }
    setMsg("");
    const [allMapel, mapped] = await Promise.all([
      api("/mata-pelajaran") as Promise<Mapel[]>,
      api(`/kelas/${kelasId}/mapel`) as Promise<any[]>,
    ]);
    const map = new Map<number, string>();
    mapped.forEach((m) => map.set(m.mata_pelajaran_id, m.kitab || ""));
    setRows(allMapel.map((m) => ({
      mata_pelajaran_id: m.id,
      nama: m.nama,
      checked: map.has(m.id),
      kitab: map.get(m.id) || "",
    })));
  }
  useEffect(() => { load(); /* eslint-disable-next-line */ }, [kelasId]);

  function toggle(id: number) {
    setRows((p) => p.map((r) => (r.mata_pelajaran_id === id ? { ...r, checked: !r.checked } : r)));
  }
  function setKitab(id: number, v: string) {
    setRows((p) => p.map((r) => (r.mata_pelajaran_id === id ? { ...r, kitab: v } : r)));
  }

  async function simpan() {
    setSaving(true); setMsg("");
    try {
      const items = rows.filter((r) => r.checked).map((r) => ({ mata_pelajaran_id: r.mata_pelajaran_id, kitab: r.kitab || null }));
      const d = await api(`/kelas/${kelasId}/mapel`, { method: "PUT", body: { items } });
      setMsg(`Tersimpan: ${d.saved} pelajaran.`);
    } catch (e: any) { setMsg(e.message); }
    finally { setSaving(false); }
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
      <h1 style={{ margin: 0 }}>Pelajaran per Kelas</h1>
      <p className="muted" style={{ margin: 0, fontSize: 13 }}>
        Centang pelajaran yang diajarkan di kelas ini & isi nama kitabnya. Nilai, Leger, dan Rapor mengikuti daftar ini.
      </p>

      <div className="row">
        <select className="input" value={kelasId} onChange={(e) => setKelasId(e.target.value)}>
          <option value="">— pilih kelas —</option>
          {kelas.map((k) => <option key={k.id} value={k.id}>{k.nama}{k.aktif ? "" : " (non-aktif)"}</option>)}
        </select>
        <button className="btn" onClick={simpan} disabled={saving || !kelasId}>{saving ? "Menyimpan..." : "Simpan"}</button>
      </div>

      {msg && <div className="card" style={{ padding: 12 }}>{msg}</div>}

      {kelasId && rows.length > 0 && (
        <div className="card table-wrap" style={{ padding: 0 }}>
          <table>
            <thead>
              <tr>
                <th style={{ width: 60 }}>Ajar?</th>
                <th>Mata Pelajaran</th>
                <th>Nama Kitab</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.mata_pelajaran_id}>
                  <td style={{ textAlign: "center" }}>
                    <input type="checkbox" checked={r.checked} onChange={() => toggle(r.mata_pelajaran_id)} style={{ width: 18, height: 18, cursor: "pointer" }} />
                  </td>
                  <td>{r.nama}</td>
                  <td>
                    <input className="input" style={{ width: "100%" }} placeholder="mis. MUQODDIMAH"
                      value={r.kitab} disabled={!r.checked}
                      onChange={(e) => setKitab(r.mata_pelajaran_id, e.target.value)} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!kelasId && <p className="muted">Pilih kelas untuk mengatur pelajaran.</p>}
    </div>
  );
}
