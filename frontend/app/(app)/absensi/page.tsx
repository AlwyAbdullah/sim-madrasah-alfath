"use client";

import { useEffect, useState } from "react";
import { api, exportUrl } from "@/lib/api";

type Kelas = { id: number; nama: string };
type Item = {
  santri_id: number;
  nama: string;
  nis: string;
  status: string;
  keterangan?: string | null;
};

const STATUSES = ["hadir", "izin", "sakit", "alpha"];

export default function AbsensiPage() {
  const [kelas, setKelas] = useState<Kelas[]>([]);
  const [kelasId, setKelasId] = useState("");
  const [tanggal, setTanggal] = useState(() => new Date().toISOString().slice(0, 10));
  const [bulan, setBulan] = useState(() => new Date().toISOString().slice(0, 7));
  const [items, setItems] = useState<Item[]>([]);
  const [msg, setMsg] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => { api("/kelas?aktif=1").then(setKelas).catch(() => {}); }, []);

  async function load() {
    if (!kelasId) { setItems([]); return; }
    const d = await api(`/absensi?kelas_id=${kelasId}&tanggal=${tanggal}`);
    setItems(d.items.map((it: Item) => ({ ...it, status: it.status || "" })));
    setMsg("");
  }
  useEffect(() => { load(); /* eslint-disable-next-line */ }, [kelasId, tanggal]);

  function setStatus(id: number, status: string) {
    setItems((prev) => prev.map((it) => (it.santri_id === id ? { ...it, status } : it)));
  }
  function setKet(id: number, ket: string) {
    setItems((prev) => prev.map((it) => (it.santri_id === id ? { ...it, keterangan: ket } : it)));
  }
  function semuaHadir() {
    setItems((prev) => prev.map((it) => ({ ...it, status: "hadir" })));
  }

  const counts = STATUSES.reduce((acc, s) => {
    acc[s] = items.filter((i) => i.status === s).length;
    return acc;
  }, {} as Record<string, number>);

  function exportRekap() {
    if (!kelasId) return;
    window.open(exportUrl(`/absensi/export?kelas_id=${kelasId}&bulan=${bulan}`), "_blank");
  }

  async function simpan() {
    const belum = items.filter((i) => !i.status);
    if (belum.length > 0) { setMsg(`${belum.length} santri belum ditandai.`); return; }
    setSaving(true);
    setMsg("");
    try {
      const d = await api("/absensi/batch", {
        method: "POST",
        body: {
          kelas_id: Number(kelasId),
          tanggal,
          items: items.map((i) => ({
            santri_id: i.santri_id,
            status: i.status,
            keterangan: i.keterangan || null,
          })),
        },
      });
      setMsg(`Tersimpan: ${d.saved} santri.`);
    } catch (e: any) {
      setMsg(e.message);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
      <h1 style={{ margin: 0 }}>Absensi Harian</h1>

      <div className="row">
        <select className="input" value={kelasId} onChange={(e) => setKelasId(e.target.value)}>
          <option value="">— pilih kelas —</option>
          {kelas.map((k) => <option key={k.id} value={k.id}>{k.nama}</option>)}
        </select>
        <input className="input" type="date" value={tanggal} onChange={(e) => setTanggal(e.target.value)} />
        <button className="btn secondary" onClick={semuaHadir} disabled={!items.length}>
          ✓ Tandai Semua Hadir
        </button>
        <button className="btn" onClick={simpan} disabled={saving || !items.length}>
          {saving ? "Menyimpan..." : "Simpan"}
        </button>
      </div>

      <div className="row" style={{ fontSize: 13 }}>
        <span className="muted">Rekap bulanan:</span>
        <input className="input" type="month" value={bulan} onChange={(e) => setBulan(e.target.value)} />
        <button className="btn secondary" onClick={exportRekap} disabled={!kelasId}>
          ⬇ Ekspor Rekap Absensi
        </button>
      </div>

      {items.length > 0 && (
        <div className="row" style={{ fontSize: 13 }}>
          <span className="badge hadir">Hadir {counts.hadir}</span>
          <span className="badge izin">Izin {counts.izin}</span>
          <span className="badge sakit">Sakit {counts.sakit}</span>
          <span className="badge alpha">Alpha {counts.alpha}</span>
        </div>
      )}

      {msg && <div className="card" style={{ padding: 12 }}>{msg}</div>}

      {items.length > 0 && (
        <div className="card table-wrap" style={{ padding: 0 }}>
          <table>
            <thead>
              <tr>
                <th style={{ width: 40 }}>No</th>
                <th>Nama</th>
                <th style={{ width: 320 }}>Status</th>
                <th>Keterangan (opsional)</th>
              </tr>
            </thead>
            <tbody>
              {items.map((it, idx) => (
                <tr key={it.santri_id}>
                  <td>{idx + 1}</td>
                  <td>{it.nama}<div className="muted" style={{ fontSize: 12 }}>{it.nis}</div></td>
                  <td>
                    <div className="row" style={{ gap: 6 }}>
                      {STATUSES.map((s) => (
                        <label key={s} style={{ fontSize: 13, cursor: "pointer" }}>
                          <input type="radio" name={`st-${it.santri_id}`}
                            checked={it.status === s} onChange={() => setStatus(it.santri_id, s)} /> {s}
                        </label>
                      ))}
                    </div>
                  </td>
                  <td>
                    <input className="input" style={{ width: "100%" }} placeholder="—"
                      value={it.keterangan || ""} onChange={(e) => setKet(it.santri_id, e.target.value)} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!kelasId && <p className="muted">Pilih kelas untuk mulai absensi.</p>}
    </div>
  );
}
