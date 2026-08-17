"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { api, exportUrl } from "@/lib/api";

type Kelas = { id: number; nama: string };
type Angka = { hadir: number; izin: number; sakit: number; alpha: number; total: number; persen: number };
type PerKelas = Angka & { kelas_id: number; kelas: string; santri: number };
type PerBulan = Angka & { bulan: string };
type PerSantri = Angka & { santri_id: number; nama: string; kelas: string };
type Rekap = {
  from: string; to: string; hari_efektif: number;
  ringkasan: Angka; per_kelas: PerKelas[]; per_bulan: PerBulan[]; perhatian: PerSantri[];
};

const NAMA_BULAN = ["Januari", "Februari", "Maret", "April", "Mei", "Juni",
  "Juli", "Agustus", "September", "Oktober", "November", "Desember"];
const pad = (n: number) => String(n).padStart(2, "0");

function rentang(mode: string, bulan: string, tahun: number, semester: string, dari: string, sampai: string) {
  if (mode === "bulan") {
    const [y, m] = bulan.split("-").map(Number);
    const akhir = new Date(y, m, 0).getDate();
    return { from: `${y}-${pad(m)}-01`, to: `${y}-${pad(m)}-${pad(akhir)}`, label: `${NAMA_BULAN[m - 1]} ${y}` };
  }
  if (mode === "semester") {
    return semester === "ganjil"
      ? { from: `${tahun}-07-01`, to: `${tahun}-12-31`, label: `Semester Ganjil ${tahun}` }
      : { from: `${tahun}-01-01`, to: `${tahun}-06-30`, label: `Semester Genap ${tahun}` };
  }
  if (mode === "tahun") {
    return { from: `${tahun}-01-01`, to: `${tahun}-12-31`, label: `Tahun ${tahun}` };
  }
  return { from: dari, to: sampai, label: `${dari} s/d ${sampai}` };
}

/** Bar persentase sederhana — hijau bila baik, merah bila rendah. */
function Bar({ persen }: { persen: number }) {
  const warna = persen >= 90 ? "#16a34a" : persen >= 75 ? "#d97706" : "#dc2626";
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 8, minWidth: 130 }}>
      <div style={{ flex: 1, height: 8, background: "#e5e7eb", borderRadius: 999, overflow: "hidden" }}>
        <div style={{ width: `${Math.min(100, persen)}%`, height: "100%", background: warna }} />
      </div>
      <span style={{ fontVariantNumeric: "tabular-nums", fontWeight: 600, fontSize: 13, color: warna, width: 46, textAlign: "right" }}>
        {persen.toFixed(1)}%
      </span>
    </div>
  );
}

export default function RekapAbsensiPage() {
  const kini = new Date();
  const [kelas, setKelas] = useState<Kelas[]>([]);
  const [kelasId, setKelasId] = useState("");
  const [mode, setMode] = useState("bulan");
  const [bulan, setBulan] = useState(() => kini.toISOString().slice(0, 7));
  const [tahun, setTahun] = useState(kini.getFullYear());
  const [semester, setSemester] = useState(kini.getMonth() + 1 >= 7 ? "ganjil" : "genap");
  const [dari, setDari] = useState(() => kini.toISOString().slice(0, 10));
  const [sampai, setSampai] = useState(() => kini.toISOString().slice(0, 10));
  const [ambangAlpha, setAmbangAlpha] = useState(1);

  const [data, setData] = useState<Rekap | null>(null);
  const [memuat, setMemuat] = useState(false);
  const [msg, setMsg] = useState("");

  const rg = useMemo(() => rentang(mode, bulan, tahun, semester, dari, sampai),
    [mode, bulan, tahun, semester, dari, sampai]);

  useEffect(() => { api("/kelas").then(setKelas).catch(() => {}); }, []);

  const muat = useCallback(async () => {
    setMemuat(true); setMsg("");
    try {
      const d = await api(`/absensi/rekap?from=${rg.from}&to=${rg.to}${kelasId ? `&kelas_id=${kelasId}` : ""}`);
      setData(d);
    } catch (e: any) { setMsg(e.message); }
    finally { setMemuat(false); }
  }, [rg.from, rg.to, kelasId]);

  useEffect(() => { muat(); }, [muat]);

  const perhatian = useMemo(
    () => (data?.perhatian || []).filter((s) => s.alpha >= ambangAlpha),
    [data, ambangAlpha]);

  const r = data?.ringkasan;

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
      <h1 style={{ margin: 0 }}>Rekap Kehadiran</h1>
      <p className="muted" style={{ margin: 0, fontSize: 13 }}>
        Hari efektif dihitung <strong>Sabtu–Rabu</strong> dan mengecualikan hari libur yang terdaftar,
        sama seperti perhitungan di Dashboard.
      </p>

      {/* ===== penyaring ===== */}
      <div className="card" style={{ padding: 12, display: "flex", gap: 10, alignItems: "center", flexWrap: "wrap" }}>
        <select className="input" value={mode} onChange={(e) => setMode(e.target.value)}>
          <option value="bulan">Per Bulan</option>
          <option value="semester">Per Semester</option>
          <option value="tahun">Per Tahun</option>
          <option value="custom">Rentang Sendiri</option>
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
            <input className="input" type="number" style={{ width: 100 }} value={tahun}
              onChange={(e) => setTahun(Number(e.target.value))} />
          </>
        )}
        {mode === "tahun" && (
          <input className="input" type="number" style={{ width: 100 }} value={tahun}
            onChange={(e) => setTahun(Number(e.target.value))} />
        )}
        {mode === "custom" && (
          <>
            <input className="input" type="date" value={dari} onChange={(e) => setDari(e.target.value)} />
            <span className="muted">s/d</span>
            <input className="input" type="date" value={sampai} onChange={(e) => setSampai(e.target.value)} />
          </>
        )}

        <select className="input" value={kelasId} onChange={(e) => setKelasId(e.target.value)}>
          <option value="">Semua kelas</option>
          {kelas.map((k) => <option key={k.id} value={k.id}>{k.nama}</option>)}
        </select>

        <div style={{ flex: 1 }} />
        <button className="btn secondary" onClick={() => muat()}>↻ Muat ulang</button>
        <button className="btn secondary" disabled={!data || data.ringkasan.total === 0}
          onClick={() => window.open(exportUrl(
            `/absensi/rekap/export?from=${rg.from}&to=${rg.to}${kelasId ? `&kelas_id=${kelasId}` : ""}&label=${encodeURIComponent(rg.label)}`), "_blank")}>
          ⬇ Ekspor Excel
        </button>
      </div>

      <p className="muted" style={{ margin: 0, fontSize: 13 }}>
        Periode: <strong>{rg.label}</strong> ({rg.from} s/d {rg.to})
        {data && <> · <strong>{data.hari_efektif}</strong> hari efektif tercatat</>}
      </p>

      {msg && <div className="card" style={{ padding: 12 }}>{msg}</div>}

      {memuat ? (
        <p className="muted">Memuat…</p>
      ) : !r || r.total === 0 ? (
        <p className="muted">Belum ada catatan absensi pada periode ini.</p>
      ) : (
        <>
          {/* ===== kartu ringkasan ===== */}
          <div className="grid-kpi">
            <div className="card kpi" style={{ ["--accent" as any]: r.persen >= 90 ? "var(--primary)" : r.persen >= 75 ? "var(--warn)" : "var(--danger)" }}>
              <div className="kpi-top"><span className="kpi-label">Persentase Kehadiran</span><span className="kpi-icon">📈</span></div>
              <div className="kpi-value">{r.persen.toFixed(1)}%</div>
              <div className="muted" style={{ fontSize: 12 }}>{r.hadir} dari {r.total} catatan</div>
            </div>
            <div className="card kpi" style={{ ["--accent" as any]: "var(--primary)" }}>
              <div className="kpi-top"><span className="kpi-label">Hadir</span><span className="kpi-icon">✅</span></div>
              <div className="kpi-value">{r.hadir}</div>
            </div>
            <div className="card kpi" style={{ ["--accent" as any]: "var(--info)" }}>
              <div className="kpi-top"><span className="kpi-label">Izin</span><span className="kpi-icon">📄</span></div>
              <div className="kpi-value">{r.izin}</div>
            </div>
            <div className="card kpi" style={{ ["--accent" as any]: "var(--warn)" }}>
              <div className="kpi-top"><span className="kpi-label">Sakit</span><span className="kpi-icon">🤒</span></div>
              <div className="kpi-value">{r.sakit}</div>
            </div>
            <div className="card kpi" style={{ ["--accent" as any]: "var(--danger)" }}>
              <div className="kpi-top"><span className="kpi-label">Alpha</span><span className="kpi-icon">⚠️</span></div>
              <div className="kpi-value">{r.alpha}</div>
            </div>
          </div>

          {/* ===== per kelas ===== */}
          <div className="card" style={{ padding: 0 }}>
            <div style={{ padding: "10px 14px", borderBottom: "1px solid var(--border)", fontWeight: 600 }}>
              Perbandingan Antar Kelas
            </div>
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Kelas</th><th style={{ width: 70 }}>Santri</th>
                    <th style={{ width: 70 }}>Hadir</th><th style={{ width: 60 }}>Izin</th>
                    <th style={{ width: 60 }}>Sakit</th><th style={{ width: 70 }}>Alpha</th>
                    <th style={{ width: 180 }}>Kehadiran</th>
                  </tr>
                </thead>
                <tbody>
                  {[...(data?.per_kelas || [])].sort((a, b) => a.persen - b.persen).map((k) => (
                    <tr key={k.kelas_id}>
                      <td><strong>{k.kelas}</strong></td>
                      <td>{k.santri}</td>
                      <td>{k.hadir}</td>
                      <td>{k.izin}</td>
                      <td>{k.sakit}</td>
                      <td style={{ color: k.alpha > 0 ? "var(--danger)" : undefined, fontWeight: k.alpha > 0 ? 700 : 400 }}>{k.alpha}</td>
                      <td><Bar persen={k.persen} /></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <div className="muted" style={{ padding: "8px 14px", fontSize: 12 }}>Diurutkan dari kehadiran terendah.</div>
          </div>

          {/* ===== tren bulanan ===== */}
          {(data?.per_bulan.length || 0) > 1 && (
            <div className="card" style={{ padding: 0 }}>
              <div style={{ padding: "10px 14px", borderBottom: "1px solid var(--border)", fontWeight: 600 }}>
                Tren Bulanan
              </div>
              <div className="table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>Bulan</th><th style={{ width: 70 }}>Hadir</th><th style={{ width: 60 }}>Izin</th>
                      <th style={{ width: 60 }}>Sakit</th><th style={{ width: 70 }}>Alpha</th>
                      <th style={{ width: 180 }}>Kehadiran</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data?.per_bulan.map((b) => {
                      const [y, m] = b.bulan.split("-").map(Number);
                      return (
                        <tr key={b.bulan}>
                          <td>{NAMA_BULAN[m - 1]} {y}</td>
                          <td>{b.hadir}</td><td>{b.izin}</td><td>{b.sakit}</td>
                          <td style={{ color: b.alpha > 0 ? "var(--danger)" : undefined }}>{b.alpha}</td>
                          <td><Bar persen={b.persen} /></td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* ===== perlu perhatian ===== */}
          <div className="card" style={{ padding: 0 }}>
            <div style={{ padding: "10px 14px", borderBottom: "1px solid var(--border)", display: "flex", justifyContent: "space-between", alignItems: "center", flexWrap: "wrap", gap: 8 }}>
              <strong>Santri Perlu Perhatian</strong>
              <div className="row" style={{ fontSize: 13 }}>
                <span className="muted">Tampilkan bila alpha ≥</span>
                <input className="input" type="number" min={1} style={{ width: 70 }}
                  value={ambangAlpha} onChange={(e) => setAmbangAlpha(Math.max(1, Number(e.target.value)))} />
              </div>
            </div>
            {perhatian.length === 0 ? (
              <p className="muted" style={{ padding: 14, margin: 0 }}>
                Tidak ada santri dengan alpha ≥ {ambangAlpha} pada periode ini. 👍
              </p>
            ) : (
              <div className="table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th style={{ width: 40 }}>#</th><th>Nama</th><th style={{ width: 110 }}>Kelas</th>
                      <th style={{ width: 70 }}>Alpha</th><th style={{ width: 60 }}>Izin</th>
                      <th style={{ width: 60 }}>Sakit</th><th style={{ width: 180 }}>Kehadiran</th>
                    </tr>
                  </thead>
                  <tbody>
                    {perhatian.map((s, i) => (
                      <tr key={s.santri_id}>
                        <td>{i + 1}</td>
                        <td>{s.nama}</td>
                        <td>{s.kelas}</td>
                        <td><span className="badge alpha">{s.alpha}</span></td>
                        <td>{s.izin}</td>
                        <td>{s.sakit}</td>
                        <td><Bar persen={s.persen} /></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
