"use client";

import { useEffect, useState } from "react";
import { api, exportUrl } from "@/lib/api";

type Opt = { id: number; nama: string; is_active?: boolean };
type Mapel = { id: number; nama: string };
type Row = {
  santri_id: number;
  nis: string;
  nama: string;
  nilai: Record<string, number | null>;
  rata_rata: number | null;
  peringkat: number;
};

export default function LegerPage() {
  const [kelas, setKelas] = useState<Opt[]>([]);
  const [periode, setPeriode] = useState<Opt[]>([]);
  const [kelasId, setKelasId] = useState("");
  const [periodeId, setPeriodeId] = useState("");
  const [mapel, setMapel] = useState<Mapel[]>([]);
  const [rows, setRows] = useState<Row[]>([]);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    api("/kelas?aktif=1").then(setKelas).catch(() => {});
    api("/periode").then((p: Opt[]) => {
      setPeriode(p);
      const aktif = p.find((x) => x.is_active);
      if (aktif) setPeriodeId(String(aktif.id));
    }).catch(() => {});
  }, []);

  const ready = kelasId && periodeId;

  useEffect(() => {
    if (!ready) { setRows([]); setMapel([]); setLoaded(false); return; }
    api(`/nilai/leger?kelas_id=${kelasId}&periode_id=${periodeId}`).then((d) => {
      setMapel(d.mapel || []);
      setRows([...(d.rows || [])].sort((a: Row, b: Row) => a.peringkat - b.peringkat));
      setLoaded(true);
    }).catch(() => {});
  }, [kelasId, periodeId]);

  function exportExcel() {
    window.open(exportUrl(`/nilai/leger/export?kelas_id=${kelasId}&periode_id=${periodeId}`), "_blank");
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
      <h1 style={{ margin: 0 }}>Leger Nilai &amp; Peringkat</h1>

      <div className="row">
        <select className="input" value={kelasId} onChange={(e) => setKelasId(e.target.value)}>
          <option value="">— kelas —</option>
          {kelas.map((k) => <option key={k.id} value={k.id}>{k.nama}</option>)}
        </select>
        <select className="input" value={periodeId} onChange={(e) => setPeriodeId(e.target.value)}>
          <option value="">— periode —</option>
          {periode.map((p) => <option key={p.id} value={p.id}>{p.nama}</option>)}
        </select>
        <button className="btn secondary" onClick={exportExcel} disabled={!ready || rows.length === 0}>⬇ Ekspor Excel</button>
      </div>

      {ready && loaded && rows.length === 0 && (
        <p className="muted">Belum ada nilai untuk kelas &amp; periode ini.</p>
      )}

      {rows.length > 0 && (
        <div className="card" style={{ padding: 0, overflow: "auto" }}>
          <table>
            <thead>
              <tr>
                <th style={{ width: 40 }}>Rank</th>
                <th>Nama</th>
                {mapel.map((m) => <th key={m.id} style={{ whiteSpace: "nowrap" }}>{m.nama}</th>)}
                <th style={{ width: 90 }}>Rata-rata</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.santri_id}>
                  <td><strong>{r.peringkat}</strong></td>
                  <td>{r.nama}<div className="muted" style={{ fontSize: 12 }}>{r.nis}</div></td>
                  {mapel.map((m) => <td key={m.id}>{r.nilai[m.id] ?? "-"}</td>)}
                  <td><strong>{r.rata_rata ?? "-"}</strong></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!ready && <p className="muted">Pilih kelas dan periode untuk menampilkan leger.</p>}
    </div>
  );
}
