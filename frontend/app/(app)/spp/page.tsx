"use client";

import { useEffect, useState } from "react";
import { api, exportUrl } from "@/lib/api";

type Kelas = { id: number; nama: string };
type Item = {
  santri_id: number;
  nis: string;
  nama: string;
  bulan: Record<string, boolean>; // key = bulan kalender (1..12)
  lunas: number;
};

const BULAN = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];
const URUTAN = [7, 8, 9, 10, 11, 12, 1, 2, 3, 4, 5, 6]; // Juli..Juni

function taStartNow() {
  const d = new Date();
  return d.getMonth() + 1 >= 7 ? d.getFullYear() : d.getFullYear() - 1;
}

export default function SppPage() {
  const [kelas, setKelas] = useState<Kelas[]>([]);
  const [kelasId, setKelasId] = useState("");
  const [tahun, setTahun] = useState(taStartNow()); // tahun ajaran mulai
  const [items, setItems] = useState<Item[]>([]);
  const [msg, setMsg] = useState("");

  useEffect(() => { api("/kelas?aktif=1").then(setKelas).catch(() => {}); }, []);

  async function load() {
    if (!kelasId) { setItems([]); return; }
    const d = await api(`/spp?kelas_id=${kelasId}&tahun=${tahun}`);
    setItems((d.items || []).map((it: Item) => ({ ...it, bulan: it.bulan || {} })));
  }
  useEffect(() => { load(); /* eslint-disable-next-line */ }, [kelasId, tahun]);

  async function toggle(it: Item, bulan: number) {
    const baru = !it.bulan[bulan];
    setItems((prev) => prev.map((x) => {
      if (x.santri_id !== it.santri_id) return x;
      const bln = { ...x.bulan, [bulan]: baru };
      const lunas = Object.values(bln).filter(Boolean).length;
      return { ...x, bulan: bln, lunas };
    }));
    try {
      await api("/spp/toggle", { method: "POST", body: { santri_id: it.santri_id, tahun, bulan, lunas: baru } });
    } catch (e: any) { setMsg(e.message); load(); }
  }

  const years = [taStartNow() - 1, taStartNow(), taStartNow() + 1, taStartNow() + 2];
  const totalLunas = items.reduce((a, x) => a + x.lunas, 0);

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
      <h1 style={{ margin: 0 }}>Pembayaran SPP</h1>
      <p className="muted" style={{ margin: 0, fontSize: 13 }}>Tahun ajaran Juli–Juni. Centang = lunas (tersimpan otomatis).</p>

      <div className="row">
        <select className="input" value={kelasId} onChange={(e) => setKelasId(e.target.value)}>
          <option value="">— pilih kelas —</option>
          {kelas.map((k) => <option key={k.id} value={k.id}>{k.nama}</option>)}
        </select>
        <select className="input" value={tahun} onChange={(e) => setTahun(Number(e.target.value))}>
          {years.map((y) => <option key={y} value={y}>{y}/{y + 1}</option>)}
        </select>
        <button className="btn secondary" onClick={() => window.open(exportUrl(`/spp/export?kelas_id=${kelasId}&tahun=${tahun}`), "_blank")} disabled={!kelasId || items.length === 0}>
          ⬇ Ekspor Excel
        </button>
      </div>

      {items.length > 0 && (
        <p className="muted" style={{ margin: 0, fontSize: 13 }}>Total lunas: <strong>{totalLunas}</strong> / {items.length * 12} slot.</p>
      )}
      {msg && <div className="card" style={{ padding: 12 }}>{msg}</div>}

      {items.length > 0 && (
        <div className="card table-wrap" style={{ padding: 0 }}>
          <table>
            <thead>
              <tr>
                <th style={{ position: "sticky", left: 0, background: "#f8fafc", minWidth: 150 }}>Nama</th>
                {URUTAN.map((b) => <th key={b} style={{ textAlign: "center", padding: "10px 6px" }}>{BULAN[b - 1]}</th>)}
                <th style={{ textAlign: "center" }}>Lunas</th>
              </tr>
            </thead>
            <tbody>
              {items.map((it) => (
                <tr key={it.santri_id}>
                  <td style={{ position: "sticky", left: 0, background: "#fff" }}>
                    {it.nama}<div className="muted" style={{ fontSize: 12 }}>{it.nis}</div>
                  </td>
                  {URUTAN.map((b) => (
                    <td key={b} style={{ textAlign: "center", padding: "6px" }}>
                      <input type="checkbox" checked={!!it.bulan[b]} onChange={() => toggle(it, b)} style={{ width: 18, height: 18, cursor: "pointer" }} />
                    </td>
                  ))}
                  <td style={{ textAlign: "center" }}>
                    <span className={`badge ${it.lunas === 12 ? "hadir" : it.lunas === 0 ? "alpha" : "sakit"}`}>{it.lunas}/12</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!kelasId && <p className="muted">Pilih kelas untuk menandai pembayaran SPP.</p>}
    </div>
  );
}
