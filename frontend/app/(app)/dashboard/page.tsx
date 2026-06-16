"use client";

import { useEffect, useRef, useState } from "react";
import { api } from "@/lib/api";

type Kelas = { id: number; nama: string };
type Santri = { id: number; nis: string; nama: string; kelas_nama?: string };

export default function DashboardPage() {
  const [kelas, setKelas] = useState<Kelas[]>([]);
  const [kelasId, setKelasId] = useState("");
  const [tanggal, setTanggal] = useState(() => new Date().toISOString().slice(0, 10));
  const [summary, setSummary] = useState<any>(null);
  const [santriList, setSantriList] = useState<Santri[]>([]);
  const [selected, setSelected] = useState<number | "">("");
  const [detail, setDetail] = useState<any>(null);
  const [range, setRange] = useState("tahun"); // mingguan|bulanan|semester|tahun

  // pencarian santri (combobox)
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const blurTimer = useRef<any>(null);

  useEffect(() => { api("/kelas?aktif=1").then(setKelas).catch(() => {}); }, []);

  useEffect(() => {
    const q = new URLSearchParams({ tanggal });
    if (kelasId) q.set("kelas_id", kelasId);
    api(`/dashboard/summary?${q}`).then(setSummary).catch(() => {});
    const sq = kelasId ? `?kelas_id=${kelasId}&aktif=1` : "?aktif=1";
    api(`/santri${sq}`).then(setSantriList).catch(() => {});
    // reset pilihan saat ganti kelas
    setSelected(""); setQuery(""); setDetail(null);
  }, [kelasId, tanggal]);

  useEffect(() => {
    if (selected === "") { setDetail(null); return; }
    api(`/santri/${selected}/detail?range=${range}`).then(setDetail).catch(() => {});
  }, [selected, range]);

  const rangeLabel: Record<string, string> = {
    mingguan: "Pekan ini (7 hari)", bulanan: "Bulan ini", semester: "Semester berjalan", tahun: "Tahun ajaran",
  };

  const filtered = santriList
    .filter((s) => {
      const q = query.toLowerCase().trim();
      if (!q) return true;
      return s.nama.toLowerCase().includes(q) || (s.nis || "").toLowerCase().includes(q);
    })
    .slice(0, 50);

  function pilih(s: Santri) {
    setSelected(s.id);
    setQuery(`${s.nama}${s.kelas_nama ? ` (${s.kelas_nama})` : ""}`);
    setOpen(false);
  }
  function clearPilih() {
    setSelected(""); setQuery(""); setDetail(null); setOpen(false);
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
      <h1 style={{ margin: 0 }}>Dashboard</h1>

      <div className="row">
        <select className="input" value={kelasId} onChange={(e) => setKelasId(e.target.value)}>
          <option value="">Semua Kelas</option>
          {kelas.map((k) => <option key={k.id} value={k.id}>{k.nama}</option>)}
        </select>
        <input className="input" type="date" value={tanggal} onChange={(e) => setTanggal(e.target.value)} />
      </div>

      {summary && (
        <div className="grid-kpi">
          <Kpi label="Total Santri" value={summary.total_santri} icon="🧑‍🎓" color="#7c3aed" />
          <Kpi label="Hadir Hari Ini" value={summary.hadir} icon="✅" color="#16a34a" />
          <Kpi label="Izin" value={summary.izin} icon="✉️" color="#2563eb" />
          <Kpi label="Sakit" value={summary.sakit} icon="🤒" color="#d97706" />
          <Kpi label="Alpha" value={summary.alpha} icon="⛔" color="#dc2626" />
          <Kpi label="% Kehadiran" value={`${summary.persentase_kehadiran}%`} icon="📈" color="#0891b2" />
        </div>
      )}

      <div className="detail-grid">
        <div className="card">
          <h3 style={{ marginTop: 0 }}>Cari Santri</h3>
          <div style={{ position: "relative" }}>
            <input
              className="input"
              style={{ width: "100%" }}
              placeholder="Ketik nama atau NIS…"
              value={query}
              onChange={(e) => { setQuery(e.target.value); setOpen(true); if (selected !== "") setSelected(""); }}
              onFocus={() => setOpen(true)}
              onBlur={() => { blurTimer.current = setTimeout(() => setOpen(false), 150); }}
            />
            {query && (
              <button onClick={clearPilih} title="Hapus"
                style={{ position: "absolute", right: 8, top: 8, border: "none", background: "transparent", cursor: "pointer", color: "var(--muted)", fontSize: 16 }}>
                ×
              </button>
            )}

            {open && (
              <div
                onMouseDown={() => { if (blurTimer.current) clearTimeout(blurTimer.current); }}
                style={{
                  position: "absolute", top: "100%", left: 0, right: 0, zIndex: 20,
                  background: "#fff", border: "1px solid var(--border)", borderRadius: 8,
                  marginTop: 4, maxHeight: 280, overflowY: "auto", boxShadow: "0 6px 18px rgba(0,0,0,.08)",
                }}>
                {filtered.length === 0 && (
                  <div className="muted" style={{ padding: 10, fontSize: 14 }}>Tidak ada santri cocok.</div>
                )}
                {filtered.map((s) => (
                  <div key={s.id} onClick={() => pilih(s)}
                    style={{ padding: "8px 12px", cursor: "pointer", fontSize: 14, borderBottom: "1px solid #f1f5f9" }}
                    onMouseEnter={(e) => (e.currentTarget.style.background = "#f8fafc")}
                    onMouseLeave={(e) => (e.currentTarget.style.background = "#fff")}>
                    {s.nama}
                    <span className="muted" style={{ fontSize: 12 }}> · {s.nis || "-"}{s.kelas_nama ? ` · ${s.kelas_nama}` : ""}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
          <p className="muted" style={{ fontSize: 12, marginBottom: 0 }}>
            {santriList.length} santri{kelasId ? " di kelas ini" : ""}. Ketik untuk menyaring.
          </p>
        </div>

        <div className="card">
          <h3 style={{ marginTop: 0 }}>Detail Santri</h3>
          {!detail && <p className="muted">Cari & pilih santri untuk melihat kehadiran, ketidakhadiran & nilai.</p>}
          {detail && (
            <>
              <p style={{ margin: "0 0 12px" }}>
                <strong>{detail.santri.nama}</strong> · NIS {detail.santri.nis || "-"} · {detail.santri.kelas}
              </p>

              {/* SPP terlambat — hanya muncul untuk admin (backend yang menentukan) */}
              {detail.spp_terlambat && (
                <div style={{
                  background: detail.spp_terlambat.length ? "#fef2f2" : "#f0fdf4",
                  border: `1px solid ${detail.spp_terlambat.length ? "#fecaca" : "#bbf7d0"}`,
                  borderRadius: 10, padding: "10px 12px", marginBottom: 14,
                }}>
                  <strong style={{ fontSize: 13 }}>💳 SPP Terlambat</strong>
                  {detail.spp_terlambat.length === 0 ? (
                    <span className="muted" style={{ fontSize: 13 }}> — tidak ada tunggakan. ✅</span>
                  ) : (
                    <div style={{ marginTop: 6, display: "flex", flexWrap: "wrap", gap: 6 }}>
                      {detail.spp_terlambat.map((s: any, i: number) => (
                        <span key={i} className="badge alpha">{s.label}</span>
                      ))}
                    </div>
                  )}
                </div>
              )}

              <div className="row" style={{ justifyContent: "space-between", marginBottom: 10 }}>
                <select className="input" value={range} onChange={(e) => setRange(e.target.value)} style={{ fontSize: 13, padding: "6px 10px" }}>
                  <option value="mingguan">Mingguan</option>
                  <option value="bulanan">Bulanan</option>
                  <option value="semester">Semester</option>
                  <option value="tahun">1 Tahun (TA)</option>
                </select>
                <span className="muted" style={{ fontSize: 12 }}>
                  Kehadiran: <strong>{detail.persentase_kehadiran}%</strong> · {rangeLabel[range]}
                </span>
              </div>

              <div className="row" style={{ marginBottom: 16 }}>
                <span className="badge hadir">Hadir {detail.kehadiran.hadir}</span>
                <span className="badge izin">Izin {detail.kehadiran.izin}</span>
                <span className="badge sakit">Sakit {detail.kehadiran.sakit}</span>
                <span className="badge alpha">Alpha {detail.kehadiran.alpha}</span>
              </div>

              <h4 style={{ margin: "0 0 8px" }}>Riwayat Ketidakhadiran</h4>
              <table style={{ marginBottom: 18 }}>
                <thead>
                  <tr><th>Tanggal</th><th>Status</th><th>Alasan / Keterangan</th></tr>
                </thead>
                <tbody>
                  {(!detail.ketidakhadiran || detail.ketidakhadiran.length === 0) && (
                    <tr><td colSpan={3} className="muted">Tidak ada catatan ketidakhadiran.</td></tr>
                  )}
                  {(detail.ketidakhadiran || []).map((r: any, i: number) => (
                    <tr key={i}>
                      <td>{formatTgl(r.tanggal)}</td>
                      <td><span className={`badge ${r.status}`}>{r.status}</span></td>
                      <td>{r.keterangan || <span className="muted">—</span>}</td>
                    </tr>
                  ))}
                </tbody>
              </table>

              <h4 style={{ margin: "0 0 8px" }}>Nilai</h4>
              <table>
                <thead>
                  <tr><th>Mapel</th><th>Tugas</th><th>UTS</th><th>UAS</th><th>Akhir</th></tr>
                </thead>
                <tbody>
                  {detail.nilai.length === 0 && (
                    <tr><td colSpan={5} className="muted">Belum ada nilai.</td></tr>
                  )}
                  {detail.nilai.map((n: any, i: number) => (
                    <tr key={i}>
                      <td>{n.mata_pelajaran}</td>
                      <td>{n.tugas ?? "-"}</td>
                      <td>{n.uts ?? "-"}</td>
                      <td>{n.uas ?? "-"}</td>
                      <td><strong>{n.nilai_akhir ?? "-"}</strong></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function formatTgl(s: string) {
  // "2026-06-15" -> "15 Jun 2026"
  const [y, m, d] = s.split("-");
  const bulan = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];
  return `${d} ${bulan[Number(m) - 1] || m} ${y}`;
}

function Kpi({ label, value, color, icon }: { label: string; value: any; color?: string; icon?: string }) {
  return (
    <div className="card kpi" style={{ ["--accent" as any]: color || "var(--primary)" }}>
      <div className="kpi-top">
        <div className="kpi-value" style={{ color: color || "var(--text)" }}>{value}</div>
        {icon && <div className="kpi-icon">{icon}</div>}
      </div>
      <div className="kpi-label">{label}</div>
    </div>
  );
}
