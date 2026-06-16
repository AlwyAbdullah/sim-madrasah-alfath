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

  // tahun ajaran sudah dalam format Hijriah (mis. "1446 / 1447")
  const tahunTitle = data?.periode?.tahun_ajaran
    ? `TAHUN PELAJARAN ${data.periode.tahun_ajaran} H`
    : "";
  const semester = data?.periode?.semester === "genap" ? "GENAP" : "GANJIL";
  const semAngka = data?.periode?.semester === "genap" ? "2" : "1";

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
        <div className="rapor" style={{ background: "#fff", maxWidth: 720, margin: "0 auto", width: "100%", padding: 16, color: "#000", fontSize: 12 }}>
          {/* KOP */}
          <img src="/kop-madrasah.jpeg" alt="Madrasah Al Fath" style={{ width: "100%", maxWidth: 430, display: "block", margin: "0 auto 2px" }} />
          <div style={{ borderBottom: "2px solid #000", marginBottom: 6 }} />

          {/* Judul */}
          <div style={{ textAlign: "center", fontWeight: 800, lineHeight: 1.2, marginBottom: 6, fontSize: 13 }}>
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
          <table style={{ ...tbl, marginTop: 6 }} cellSpacing={0}>
            <thead>
              <tr>
                <th style={th}>NO<div style={ar}>رقم</div></th>
                <th style={th}>PELAJARAN<div style={ar}>المواد الدراسية</div></th>
                <th style={th}>NAMA KITAB<div style={ar}>اسم الكتاب</div></th>
                <th style={{ ...th, width: 90 }}>NILAI<div style={ar}>المكتسبة</div></th>
                <th style={{ ...th, width: 90 }}>RATA-RATA<br />KELAS</th>
              </tr>
            </thead>
            <tbody>
              {data.nilai.length === 0 && (
                <tr><td style={{ ...td, textAlign: "center" }} colSpan={5}>Belum ada nilai pada periode ini.</td></tr>
              )}
              {data.nilai.map((n: any, i: number) => (
                <tr key={i}>
                  <td style={{ ...td, textAlign: "center" }}>{i + 1}</td>
                  <td style={td}>{n.mata_pelajaran}</td>
                  <td style={td}>{n.kitab || "-"}</td>
                  <td style={{ ...td, textAlign: "center", fontWeight: 700 }}>{n.nilai_akhir ?? "-"}</td>
                  <td style={{ ...td, textAlign: "center" }}>{n.rata_kelas ?? "-"}</td>
                </tr>
              ))}
              <tr>
                <td style={{ ...td, fontWeight: 700 }} colSpan={3}>JUMLAH NILAI <span style={arInline}>مجموع النتائج</span></td>
                <td style={{ ...td, textAlign: "center", fontWeight: 700 }} colSpan={2}>{data.jumlah ?? "-"}</td>
              </tr>
              <tr>
                <td style={{ ...td, fontWeight: 700 }} colSpan={3}>RATA-RATA NILAI <span style={arInline}>كمية النتائج</span></td>
                <td style={{ ...td, textAlign: "center", fontWeight: 700 }} colSpan={2}>{data.rata ?? "-"}</td>
              </tr>
              <tr>
                <td style={{ ...td, fontWeight: 700 }} colSpan={3}>PERINGKAT <span style={arInline}>الرتبة</span></td>
                <td style={{ ...td, textAlign: "center", fontWeight: 700 }} colSpan={2}>{data.peringkat || "-"}</td>
              </tr>
            </tbody>
          </table>

          {/* Keterangan */}
          <table style={{ ...tbl, marginTop: 6 }} cellSpacing={0}>
            <thead>
              <tr><th style={th} colSpan={2}>KETERANGAN</th><th style={th} colSpan={2}>KETERANGAN</th></tr>
            </thead>
            <tbody>
              <tr>
                <td style={{ ...td, width: "30%" }}>SAKIT <span style={arInline}>مريض</span></td>
                <td style={{ ...td, width: "20%", textAlign: "center" }}>{data.kehadiran.sakit}</td>
                <td style={{ ...td, width: "30%" }}>KELAKUAN <span style={arInline}>السلوك</span></td>
                <td style={{ ...td, width: "20%" }}></td>
              </tr>
              <tr>
                <td style={td}>IZIN <span style={arInline}>عذر</span></td>
                <td style={{ ...td, textAlign: "center" }}>{data.kehadiran.izin}</td>
                <td style={td}>KETEKUNAN <span style={arInline}>مواظبة</span></td>
                <td style={td}></td>
              </tr>
              <tr>
                <td style={td}>ABSEN <span style={arInline}>لغير عذر</span></td>
                <td style={{ ...td, textAlign: "center" }}>{data.kehadiran.alpha}</td>
                <td style={td}>KEBERSIHAN <span style={arInline}>النظافة</span></td>
                <td style={td}></td>
              </tr>
            </tbody>
          </table>

          {/* Catatan */}
          <table style={{ ...tbl, marginTop: 6 }} cellSpacing={0}>
            <tbody>
              <tr><td style={{ ...td, height: 38, verticalAlign: "top", fontWeight: 700 }}>CATATAN <span style={arInline}>الإرشادات</span> :</td></tr>
            </tbody>
          </table>

          {/* Tanda tangan */}
          <div style={{ display: "flex", justifyContent: "space-between", marginTop: 12, textAlign: "center" }}>
            <div style={{ width: "32%" }}>
              <div>Wali Santri</div>
              <div style={ar}>ولي الطالب</div>
              <div style={{ height: 32 }} />
              <div>(......................)</div>
            </div>
            <div style={{ width: "32%" }}>
              <div>Wali Kelas</div>
              <div style={ar}>ولي الفصل</div>
              <div style={{ height: 32 }} />
              <div>(......................)</div>
            </div>
            <div style={{ width: "32%" }}>
              <div>Malang, {tanggalIndo()}</div>
              <div>Mudir Madrasah</div>
              <div style={{ height: 32 }} />
              <div style={{ fontWeight: 700 }}>( {MUDIR} )</div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

const tbl: React.CSSProperties = { width: "100%", borderCollapse: "collapse", fontSize: 12 };
const td: React.CSSProperties = { border: "1px solid #000", padding: "2px 6px", verticalAlign: "middle", lineHeight: 1.25 };
const th: React.CSSProperties = { border: "1px solid #000", padding: "2px 6px", fontWeight: 800, textAlign: "center", background: "#f1f5f9", lineHeight: 1.2 };
const ar: React.CSSProperties = { fontSize: 11, fontWeight: 400, direction: "rtl" };
const arInline: React.CSSProperties = { fontSize: 11, fontWeight: 400, color: "#333" };
