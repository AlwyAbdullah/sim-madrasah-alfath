"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";

type Opt = { id: number; nama: string; is_active?: boolean };
type Santri = { id: number; nis: string; nama: string };

const MUDIR = "Ustad Sholeh bin Hamid Assegaf";

function tanggalIndo(d = new Date()): string {
  const bulan = ["Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"];
  return `${d.getDate()} ${bulan[d.getMonth()]} ${d.getFullYear()}`;
}

// Kepadatan menyesuaikan jumlah mapel; konten dijaga agar muat < 1 halaman,
// sisanya diisi oleh kotak Catatan yang memuai (flex-grow).
function density(n: number) {
  if (n <= 4) return { fz: 13, pad: "5px 9px", kop: 460, title: 15, ar: 12, rowH: 28, gap: 6, sigH: 34 };
  if (n <= 7) return { fz: 12, pad: "4px 7px", kop: 420, title: 13.5, ar: 11, rowH: 22, gap: 5, sigH: 30 };
  if (n <= 10) return { fz: 11, pad: "3px 6px", kop: 380, title: 12.5, ar: 10, rowH: 19, gap: 4, sigH: 26 };
  return { fz: 10, pad: "2px 5px", kop: 340, title: 11.5, ar: 9, rowH: 16, gap: 3, sigH: 22 };
}

export default function RaporPage() {
  const [kelas, setKelas] = useState<Opt[]>([]);
  const [periode, setPeriode] = useState<Opt[]>([]);
  const [kelasId, setKelasId] = useState("");
  const [periodeId, setPeriodeId] = useState("");
  const [santriList, setSantriList] = useState<Santri[]>([]);
  const [santriId, setSantriId] = useState("");
  const [data, setData] = useState<any>(null);

  useEffect(() => {
    api("/kelas").then(setKelas).catch(() => {});
    api("/periode").then((p: Opt[]) => {
      setPeriode(p);
      const aktif = p.find((x) => x.is_active);
      if (aktif) setPeriodeId(String(aktif.id));
    }).catch(() => {});
  }, []);

  useEffect(() => {
    setSantriId(""); setData(null);
    if (!kelasId) { setSantriList([]); return; }
    api(`/santri?kelas_id=${kelasId}`).then(setSantriList).catch(() => {});
  }, [kelasId]);

  useEffect(() => {
    if (!santriId || !periodeId) { setData(null); return; }
    api(`/rapor?santri_id=${santriId}&periode_id=${periodeId}`).then(setData).catch(() => {});
  }, [santriId, periodeId]);

  const tahunTitle = data?.periode?.tahun_ajaran ? `TAHUN PELAJARAN ${data.periode.tahun_ajaran} H` : "";
  const semester = data?.periode?.semester === "genap" ? "GENAP" : "GANJIL";
  const semAngka = data?.periode?.semester === "genap" ? "2" : "1";

  const n = data?.nilai?.length || 0;
  const d = density(n);
  const td: React.CSSProperties = { border: "1px solid #000", padding: d.pad, verticalAlign: "middle", lineHeight: 1.25 };
  const th: React.CSSProperties = { ...td, fontWeight: 800, textAlign: "center", background: "#f1f5f9" };
  const tbl: React.CSSProperties = { width: "100%", borderCollapse: "collapse", fontSize: d.fz };
  const ar: React.CSSProperties = { fontSize: d.ar, fontWeight: 400, direction: "rtl" };
  const arIn: React.CSSProperties = { fontSize: d.ar, fontWeight: 400, color: "#333" };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
      <div className="no-print" style={{ display: "flex", flexDirection: "column", gap: 12 }}>
        <h1 style={{ margin: 0 }}>Rapor Santri</h1>
        <div className="row">
          <select className="input" value={kelasId} onChange={(e) => setKelasId(e.target.value)}>
            <option value="">— kelas —</option>
            {kelas.map((k) => <option key={k.id} value={k.id}>{k.nama}</option>)}
          </select>
          <select className="input" value={santriId} onChange={(e) => setSantriId(e.target.value)} disabled={!kelasId}>
            <option value="">— santri —</option>
            {santriList.map((s) => <option key={s.id} value={s.id}>{s.nama}</option>)}
          </select>
          <select className="input" value={periodeId} onChange={(e) => setPeriodeId(e.target.value)}>
            <option value="">— periode —</option>
            {periode.map((p) => <option key={p.id} value={p.id}>{p.nama}</option>)}
          </select>
          <button className="btn" onClick={() => window.print()} disabled={!data}>🖨 Cetak / Simpan PDF</button>
        </div>
      </div>

      {!data && <p className="muted no-print">Pilih kelas, santri, dan periode untuk menampilkan rapor.</p>}

      {data && (
        <div className="rapor rapor-page" style={{ background: "#fff", maxWidth: 720, margin: "0 auto", width: "100%", padding: 18, color: "#000", fontSize: d.fz, display: "flex", flexDirection: "column", boxShadow: "var(--shadow)" }}>
          {/* KOP */}
          <img src="/kop-madrasah.jpeg" alt="Madrasah Al Fath" style={{ width: "100%", maxWidth: d.kop, display: "block", margin: "0 auto 3px" }} />
          <div style={{ borderBottom: "2px solid #000", marginBottom: d.gap }} />

          {/* Judul */}
          <div style={{ textAlign: "center", fontWeight: 800, lineHeight: 1.2, marginBottom: d.gap, fontSize: d.title }}>
            <div>LAPORAN HASIL BELAJAR SANTRI SEMESTER {semester}</div>
            <div>{tahunTitle}</div>
          </div>

          {/* Identitas */}
          <table style={tbl} cellSpacing={0}>
            <tbody>
              <tr>
                <td style={{ ...td, width: "18%", fontWeight: 700 }}>NAMA</td>
                <td style={{ ...td, width: "42%" }}>: {data.santri.nama}</td>
                <td style={{ ...td, width: "18%", fontWeight: 700 }}>KELAS</td>
                <td style={{ ...td, width: "22%" }}>: {data.santri.kelas}</td>
              </tr>
              <tr>
                <td style={{ ...td, fontWeight: 700 }}>NO. INDUK</td>
                <td style={td}>: {data.santri.nis || "-"}</td>
                <td style={{ ...td, fontWeight: 700 }}>SEMESTER</td>
                <td style={td}>: {semAngka}</td>
              </tr>
            </tbody>
          </table>

          {/* Nilai */}
          <table style={{ ...tbl, marginTop: d.gap }} cellSpacing={0}>
            <thead>
              <tr>
                <th style={{ ...th, width: 36 }}>NO<div style={ar}>رقم</div></th>
                <th style={th}>PELAJARAN<div style={ar}>المواد الدراسية</div></th>
                <th style={th}>NAMA KITAB<div style={ar}>اسم الكتاب</div></th>
                <th style={{ ...th, width: 80 }}>NILAI<div style={ar}>المكتسبة</div></th>
                <th style={{ ...th, width: 80 }}>RATA-RATA<br />KELAS</th>
              </tr>
            </thead>
            <tbody>
              {data.nilai.length === 0 && (
                <tr><td style={{ ...td, textAlign: "center", height: d.rowH }} colSpan={5}>Belum ada nilai pada periode ini.</td></tr>
              )}
              {data.nilai.map((it: any, i: number) => (
                <tr key={i}>
                  <td style={{ ...td, textAlign: "center", height: d.rowH }}>{i + 1}</td>
                  <td style={td}>{it.mata_pelajaran}</td>
                  <td style={td}>{it.kitab || "-"}</td>
                  <td style={{ ...td, textAlign: "center", fontWeight: 700 }}>{it.nilai_akhir ?? "-"}</td>
                  <td style={{ ...td, textAlign: "center" }}>{it.rata_kelas ?? "-"}</td>
                </tr>
              ))}
              <tr>
                <td style={{ ...td, fontWeight: 700 }} colSpan={3}>JUMLAH NILAI <span style={arIn}>مجموع النتائج</span></td>
                <td style={{ ...td, textAlign: "center", fontWeight: 700 }} colSpan={2}>{data.jumlah ?? "-"}</td>
              </tr>
              <tr>
                <td style={{ ...td, fontWeight: 700 }} colSpan={3}>RATA-RATA NILAI <span style={arIn}>كمية النتائج</span></td>
                <td style={{ ...td, textAlign: "center", fontWeight: 700 }} colSpan={2}>{data.rata ?? "-"}</td>
              </tr>
              <tr>
                <td style={{ ...td, fontWeight: 700 }} colSpan={3}>PERINGKAT <span style={arIn}>الرتبة</span></td>
                <td style={{ ...td, textAlign: "center", fontWeight: 700 }} colSpan={2}>{data.peringkat || "-"}</td>
              </tr>
            </tbody>
          </table>

          {/* Keterangan */}
          <table style={{ ...tbl, marginTop: d.gap }} cellSpacing={0}>
            <thead>
              <tr><th style={th} colSpan={2}>KETERANGAN</th><th style={th} colSpan={2}>KETERANGAN</th></tr>
            </thead>
            <tbody>
              <tr>
                <td style={{ ...td, width: "30%" }}>SAKIT <span style={arIn}>مريض</span></td>
                <td style={{ ...td, width: "20%", textAlign: "center" }}>{data.kehadiran.sakit}</td>
                <td style={{ ...td, width: "30%" }}>KELAKUAN <span style={arIn}>السلوك</span></td>
                <td style={{ ...td, width: "20%" }}></td>
              </tr>
              <tr>
                <td style={td}>IZIN <span style={arIn}>عذر</span></td>
                <td style={{ ...td, textAlign: "center" }}>{data.kehadiran.izin}</td>
                <td style={td}>KETEKUNAN <span style={arIn}>مواظبة</span></td>
                <td style={td}></td>
              </tr>
              <tr>
                <td style={td}>ABSEN <span style={arIn}>لغير عذر</span></td>
                <td style={{ ...td, textAlign: "center" }}>{data.kehadiran.alpha}</td>
                <td style={td}>KEBERSIHAN <span style={arIn}>النظافة</span></td>
                <td style={td}></td>
              </tr>
            </tbody>
          </table>

          {/* Catatan — memuai mengisi sisa ruang */}
          <div style={{ border: "1px solid #000", padding: d.pad, marginTop: d.gap, fontWeight: 700, flex: "1 1 auto", minHeight: 28 }}>
            CATATAN <span style={arIn}>الإرشادات</span> :
          </div>

          {/* Tanda tangan — menempel di bawah */}
          <div style={{ display: "flex", justifyContent: "space-between", marginTop: d.gap + 4, textAlign: "center", fontSize: d.fz }}>
            <div style={{ width: "32%" }}>
              <div>Wali Santri</div>
              <div style={ar}>ولي الطالب</div>
              <div style={{ height: d.sigH }} />
              <div>(......................)</div>
            </div>
            <div style={{ width: "32%" }}>
              <div>Wali Kelas</div>
              <div style={ar}>ولي الفصل</div>
              <div style={{ height: d.sigH }} />
              <div>(......................)</div>
            </div>
            <div style={{ width: "32%" }}>
              <div>Malang, {tanggalIndo()}</div>
              <div>Mudir Madrasah</div>
              <div style={{ height: d.sigH }} />
              <div style={{ fontWeight: 700 }}>( {MUDIR} )</div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
