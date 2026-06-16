"use client";

import { useEffect, useState } from "react";
import { api, exportUrl } from "@/lib/api";

type Kelas = { id: number; nama: string };
type Item = {
  santri_id: number;
  nis: string;
  nama: string;
  bulan: Record<string, boolean>;
  lunas: number;
};

const BULAN = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];

export default function SppPage() {
  const [kelas, setKelas] = useState<Kelas[]>([]);
  const [kelasId, setKelasId] = useState("");
  const nowYear = new Date().getFullYear();
  const [tahun, setTahun] = useState(nowYear);
  const [items, setItems] = useState<Item[]>([]);
  const [msg, setMsg] = useState("");

  useEffect(() => { api("/kelas").then(setKelas).catch(() => {}); }, []);

  async function load() {
    if (!kelasId) { setItems([]); return; }
    const d = await api(`/spp?kelas_id=${kelasId}&tahun=${tahun}`);
    setItems((d.items || []).map((it: Item) => ({ ...it, bulan: it.bulan || {} })));
  }
  useEffect(() => { load(); /* eslint-disable-next-line */ }, [kelasId, tahun]);

  async function toggle(it: Item, bulan: number) {
    const baru = !it.bulan[bulan];
    // optimistic update
    setItems((prev) => prev.map((x) => {
      if (x.santri_id !== it.santri_id) return x;
      const bln = { ...x.bulan, [bulan]: baru };
      const lunas = Object.values(bln).filter(Boolean).length;
      return { ...x, bulan: bln, lunas };
    }));
    try {
      await api("/spp/toggle", { method: "POST", body: { santri_id: it.santri_id, tahun, bulan, lunas: baru } });
    } catch (e: any) {
      setMsg(e.message);
      load(); // rollback dari server
    }
  }

  const years = [nowYear - 2, nowYear - 1, nowYear, nowYear + 1];
  const totalLunas = items.reduce((a, x) => a + x.lunas, 0);
  const totalSlot = items.length * 12;

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
      <h1 style={{ margin: 0 }}>Pembayaran SPP</h1>

      <div className="row">
        <select className="input" value={kelasId} onChange={(e) => setKelasId(e.target.value)}>
          <option value="">— pilih kelas —</option>
          {kelas.map((k) => <option key={k.id} value={k.id}>{k.nama}</option>)}
        </select>
        <select className="input" value={tahun} onChange={(e) => setTahun(Number(e.target.value))}>
          {years.map((y) => <option key={y} value={y}>{y}</option>)}
        </select>
        <button className="btn secondary" onClick={() => window.open(exportUrl(`/spp/export?kelas_id=${kelasId}&tahun=${tahun}`), "_blank")} disabled={!kelasId || items.length === 0}>
          ⬇ Ekspor Excel
        </button>
      </div>

      {items.length > 0 && (
        <p className="muted" style={{ margin: 0, fontSize: 13 }}>
          Total lunas: <strong>{totalLunas}</strong> / {totalSlot} slot · centang kotak untuk menandai lunas (tersimpan otomatis).
        </p>
      )}
      {msg && <div className="card" style={{ padding: 12 }}>{msg}</div>}

      {items.length > 0 && (
        <div className="card table-wrap" style={{ padding: 0 }}>
          <table>
            <thead>
              <tr>
                <th style={{ position: "sticky", left: 0, background: "#fff", minWidth: 150 }}>Nama</th>
                {BULAN.map((b) => <th key={b} style={{ textAlign: "center", padding: "10px 6px" }}>{b}</th>)}
                <th style={{ textAlign: "center" }}>Lunas</th>
              </tr>
            </thead>
            <tbody>
              {items.map((it) => (
                <tr key={it.santri_id}>
                  <td style={{ position: "sticky", left: 0, background: "#fff" }}>
                    {it.nama}<div className="muted" style={{ fontSize: 12 }}>{it.nis}</div>
                  </td>
                  {BULAN.map((_, i) => {
                    const bulan = i + 1;
                    return (
                      <td key={bulan} style={{ textAlign: "center", padding: "6px" }}>
                        <input
                          type="checkbox"
                          checked={!!it.bulan[bulan]}
                          onChange={() => toggle(it, bulan)}
                          style={{ width: 18, height: 18, cursor: "pointer" }}
                        />
                      </td>
                    );
                  })}
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
