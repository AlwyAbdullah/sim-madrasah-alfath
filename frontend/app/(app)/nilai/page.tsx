"use client";

import { useEffect, useState } from "react";
import { api, exportUrl } from "@/lib/api";

type Opt = { id: number; nama: string };
type Item = {
  santri_id: number;
  nama: string;
  nis: string;
  tugas: number | null;
  uts: number | null;
  uas: number | null;
};

// Bobot: Tugas 30% + UTS 30% + UAS 40%
function hitungAkhir(t: number | null, u: number | null, a: number | null) {
  const v = (x: number | null) => (x == null ? 0 : x);
  return Math.round((v(t) * 0.3 + v(u) * 0.3 + v(a) * 0.4) * 100) / 100;
}

export default function NilaiPage() {
  const [kelas, setKelas] = useState<Opt[]>([]);
  const [mapel, setMapel] = useState<Opt[]>([]);
  const [periode, setPeriode] = useState<Opt[]>([]);
  const [kelasId, setKelasId] = useState("");
  const [mapelId, setMapelId] = useState("");
  const [periodeId, setPeriodeId] = useState("");
  const [items, setItems] = useState<Item[]>([]);
  const [msg, setMsg] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api("/kelas").then(setKelas).catch(() => {});
    api("/mata-pelajaran").then(setMapel).catch(() => {});
    api("/periode").then(setPeriode).catch(() => {});
  }, []);

  const ready = kelasId && mapelId && periodeId;

  async function load() {
    if (!ready) { setItems([]); return; }
    const d = await api(`/nilai?kelas_id=${kelasId}&mata_pelajaran_id=${mapelId}&periode_id=${periodeId}`);
    setItems(d.items);
    setMsg("");
  }
  useEffect(() => { load(); /* eslint-disable-next-line */ }, [kelasId, mapelId, periodeId]);

  function setVal(id: number, field: "tugas" | "uts" | "uas", val: string) {
    const num = val === "" ? null : Math.max(0, Math.min(100, Number(val)));
    setItems((prev) => prev.map((it) => (it.santri_id === id ? { ...it, [field]: num } : it)));
  }

  async function simpan() {
    setSaving(true);
    setMsg("");
    try {
      const d = await api("/nilai/batch", {
        method: "POST",
        body: {
          kelas_id: Number(kelasId),
          mata_pelajaran_id: Number(mapelId),
          periode_id: Number(periodeId),
          items: items.map((i) => ({ santri_id: i.santri_id, tugas: i.tugas, uts: i.uts, uas: i.uas })),
        },
      });
      setMsg(`Tersimpan: ${d.saved} santri.`);
    } catch (e: any) {
      setMsg(e.message);
    } finally {
      setSaving(false);
    }
  }

  function exportExcel() {
    const url = exportUrl(`/nilai/export?kelas_id=${kelasId}&mata_pelajaran_id=${mapelId}&periode_id=${periodeId}`);
    window.open(url, "_blank");
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
      <h1 style={{ margin: 0 }}>Input Nilai</h1>
      <p className="muted" style={{ margin: 0 }}>Bobot Nilai Akhir: Tugas 30% + UTS 30% + UAS 40%</p>

      <div className="row">
        <select className="input" value={kelasId} onChange={(e) => setKelasId(e.target.value)}>
          <option value="">— kelas —</option>
          {kelas.map((k) => <option key={k.id} value={k.id}>{k.nama}</option>)}
        </select>
        <select className="input" value={mapelId} onChange={(e) => setMapelId(e.target.value)}>
          <option value="">— mata pelajaran —</option>
          {mapel.map((m) => <option key={m.id} value={m.id}>{m.nama}</option>)}
        </select>
        <select className="input" value={periodeId} onChange={(e) => setPeriodeId(e.target.value)}>
          <option value="">— periode —</option>
          {periode.map((p) => <option key={p.id} value={p.id}>{p.nama}</option>)}
        </select>
        <button className="btn" onClick={simpan} disabled={saving || !items.length}>
          {saving ? "Menyimpan..." : "Simpan"}
        </button>
        <button className="btn secondary" onClick={exportExcel} disabled={!ready || !items.length}>
          ⬇ Ekspor Excel
        </button>
      </div>

      {msg && <div className="card" style={{ padding: 12 }}>{msg}</div>}

      {items.length > 0 && (
        <div className="card table-wrap" style={{ padding: 0 }}>
          <table>
            <thead>
              <tr>
                <th style={{ width: 40 }}>No</th>
                <th>Nama</th>
                <th style={{ width: 110 }}>Tugas</th>
                <th style={{ width: 110 }}>UTS</th>
                <th style={{ width: 110 }}>UAS</th>
                <th style={{ width: 110 }}>Nilai Akhir</th>
              </tr>
            </thead>
            <tbody>
              {items.map((it, idx) => (
                <tr key={it.santri_id}>
                  <td>{idx + 1}</td>
                  <td>{it.nama}<div className="muted" style={{ fontSize: 12 }}>{it.nis}</div></td>
                  {(["tugas", "uts", "uas"] as const).map((f) => (
                    <td key={f}>
                      <input className="input" type="number" min={0} max={100} style={{ width: 80 }}
                        value={it[f] ?? ""} onChange={(e) => setVal(it.santri_id, f, e.target.value)} />
                    </td>
                  ))}
                  <td><strong>{hitungAkhir(it.tugas, it.uts, it.uas)}</strong></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!ready && <p className="muted">Pilih kelas, mata pelajaran, dan periode untuk mulai input nilai.</p>}
    </div>
  );
}
