"use client";

import { useEffect, useMemo, useState } from "react";
import { api, exportUrl } from "@/lib/api";

type Item = { guru_id: number; nama: string; status: string; keterangan?: string | null };
type RekapRow = { guru_id: number; nama: string; hadir: number; izin: number; sakit: number; alpha: number; total: number };

const STATUSES = ["hadir", "izin", "sakit", "alpha"];
const pad = (n: number) => String(n).padStart(2, "0");

// Hitung rentang tanggal dari mode (bulan / semester / tahun).
function rentang(mode: string, bulan: string, tahun: number, semester: string): { from: string; to: string; label: string } {
  if (mode === "bulan") {
    const [yy, mm] = bulan.split("-").map(Number);
    const last = new Date(yy, mm, 0).getDate();
    const namaBulan = new Date(yy, mm - 1, 1).toLocaleDateString("id-ID", { month: "long", year: "numeric" });
    return { from: `${yy}-${pad(mm)}-01`, to: `${yy}-${pad(mm)}-${pad(last)}`, label: namaBulan };
  }
  if (mode === "semester") {
    if (semester === "ganjil") return { from: `${tahun}-07-01`, to: `${tahun}-12-31`, label: `Semester Ganjil ${tahun} (Jul–Des ${tahun})` };
    return { from: `${tahun}-01-01`, to: `${tahun}-06-30`, label: `Semester Genap ${tahun} (Jan–Jun ${tahun})` };
  }
  return { from: `${tahun}-01-01`, to: `${tahun}-12-31`, label: `Tahun ${tahun}` };
}

export default function AbsensiGuruPage() {
  const today = new Date();
  const [tanggal, setTanggal] = useState(() => today.toISOString().slice(0, 10));
  const [items, setItems] = useState<Item[]>([]);
  const [msg, setMsg] = useState("");
  const [saving, setSaving] = useState(false);

  // rekap
  const [mode, setMode] = useState("bulan");
  const [bulan, setBulan] = useState(() => today.toISOString().slice(0, 7));
  const [tahun, setTahun] = useState(today.getFullYear());
  const [semester, setSemester] = useState("ganjil");
  const [rekap, setRekap] = useState<RekapRow[]>([]);

  const range = useMemo(() => rentang(mode, bulan, tahun, semester), [mode, bulan, tahun, semester]);

  async function load() {
    const d = await api(`/absensi-guru?tanggal=${tanggal}`);
    setItems(d.items.map((it: Item) => ({ ...it, status: it.status || "" })));
    setMsg("");
  }
  useEffect(() => { load().catch(() => {}); /* eslint-disable-next-line */ }, [tanggal]);

  async function loadRekap() {
    const d = await api(`/absensi-guru/rekap?from=${range.from}&to=${range.to}`);
    setRekap(d.rows || []);
  }
  useEffect(() => { loadRekap().catch(() => {}); /* eslint-disable-next-line */ }, [range.from, range.to]);

  function setStatus(id: number, status: string) {
    setItems((prev) => prev.map((it) => (it.guru_id === id ? { ...it, status } : it)));
  }
  function setKet(id: number, ket: string) {
    setItems((prev) => prev.map((it) => (it.guru_id === id ? { ...it, keterangan: ket } : it)));
  }
  function semuaHadir() {
    setItems((prev) => prev.map((it) => ({ ...it, status: "hadir" })));
  }

  const counts = STATUSES.reduce((acc, s) => {
    acc[s] = items.filter((i) => i.status === s).length;
    return acc;
  }, {} as Record<string, number>);

  async function simpan() {
    const belum = items.filter((i) => !i.status);
    if (belum.length > 0) { setMsg(`${belum.length} guru belum ditandai.`); return; }
    setSaving(true);
    setMsg("");
    try {
      const d = await api("/absensi-guru/batch", {
        method: "POST",
        body: {
          tanggal,
          items: items.map((i) => ({ guru_id: i.guru_id, status: i.status, keterangan: i.keterangan || null })),
        },
      });
      setMsg(`Tersimpan: ${d.saved} guru.`);
      loadRekap().catch(() => {});
    } catch (e: any) {
      setMsg(e.message);
    } finally {
      setSaving(false);
    }
  }

  function exportExcel() {
    const url = exportUrl(`/absensi-guru/export?from=${range.from}&to=${range.to}&label=${encodeURIComponent(range.label)}`);
    window.open(url, "_blank");
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
      <h1 style={{ margin: 0 }}>Absensi Guru</h1>

      {/* ===== Input harian ===== */}
      <div className="row">
        <input className="input" type="date" value={tanggal} onChange={(e) => setTanggal(e.target.value)} />
        <button className="btn secondary" onClick={semuaHadir} disabled={!items.length}>✓ Tandai Semua Hadir</button>
        <button className="btn" onClick={simpan} disabled={saving || !items.length}>{saving ? "Menyimpan..." : "Simpan"}</button>
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

      {items.length > 0 ? (
        <div className="card table-wrap" style={{ padding: 0 }}>
          <table>
            <thead>
              <tr>
                <th style={{ width: 40 }}>No</th>
                <th>Nama Guru</th>
                <th style={{ width: 320 }}>Status</th>
                <th>Keterangan (opsional)</th>
              </tr>
            </thead>
            <tbody>
              {items.map((it, idx) => (
                <tr key={it.guru_id}>
                  <td>{idx + 1}</td>
                  <td>{it.nama}</td>
                  <td>
                    <div className="row" style={{ gap: 6 }}>
                      {STATUSES.map((s) => (
                        <label key={s} style={{ fontSize: 13, cursor: "pointer" }}>
                          <input type="radio" name={`st-${it.guru_id}`} checked={it.status === s} onChange={() => setStatus(it.guru_id, s)} /> {s}
                        </label>
                      ))}
                    </div>
                  </td>
                  <td>
                    <input className="input" style={{ width: "100%" }} placeholder="—"
                      value={it.keterangan || ""} onChange={(e) => setKet(it.guru_id, e.target.value)} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p className="muted">Belum ada data guru. Tambahkan dulu di <strong>Master Data → Guru</strong>.</p>
      )}

      {/* ===== Rekap ===== */}
      <h3 style={{ margin: "8px 0 0" }}>Rekap Absensi</h3>
      <div className="row" style={{ fontSize: 13 }}>
        <select className="input" value={mode} onChange={(e) => setMode(e.target.value)}>
          <option value="bulan">Per Bulan</option>
          <option value="semester">Per Semester</option>
          <option value="tahun">Per Tahun</option>
        </select>
        {mode === "bulan" && (
          <input className="input" type="month" value={bulan} onChange={(e) => setBulan(e.target.value)} />
        )}
        {mode === "semester" && (
          <>
            <select className="input" value={semester} onChange={(e) => setSemester(e.target.value)}>
              <option value="ganjil">Ganjil (Jul–Des)</option>
              <option value="genap">Genap (Jan–Jun)</option>
            </select>
            <input className="input" type="number" style={{ width: 110 }} value={tahun} onChange={(e) => setTahun(Number(e.target.value))} />
          </>
        )}
        {mode === "tahun" && (
          <input className="input" type="number" style={{ width: 110 }} value={tahun} onChange={(e) => setTahun(Number(e.target.value))} />
        )}
        <button className="btn secondary" onClick={exportExcel} disabled={!rekap.length}>⬇ Ekspor Excel</button>
      </div>
      <p className="muted" style={{ margin: 0, fontSize: 13 }}>Rentang: {range.from} s/d {range.to} ({range.label})</p>

      {rekap.length > 0 && (
        <div className="card table-wrap" style={{ padding: 0 }}>
          <table>
            <thead>
              <tr>
                <th style={{ width: 40 }}>No</th>
                <th>Nama Guru</th>
                <th style={{ width: 70 }}>Hadir</th>
                <th style={{ width: 70 }}>Izin</th>
                <th style={{ width: 70 }}>Sakit</th>
                <th style={{ width: 70 }}>Alpha</th>
                <th style={{ width: 70 }}>Total</th>
              </tr>
            </thead>
            <tbody>
              {rekap.map((r, idx) => (
                <tr key={r.guru_id}>
                  <td>{idx + 1}</td>
                  <td>{r.nama}</td>
                  <td>{r.hadir}</td>
                  <td>{r.izin}</td>
                  <td>{r.sakit}</td>
                  <td>{r.alpha}</td>
                  <td><strong>{r.total}</strong></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
