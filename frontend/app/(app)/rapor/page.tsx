"use client";

import { useEffect, useRef, useState } from "react";
import { api } from "@/lib/api";

type Opt = { id: number; nama: string; is_active?: boolean };
type Santri = { id: number; nis: string; nama: string };

const MUDIR = "Ustad Sholeh bin Hamid Assegaf";

function tanggalIndo(d = new Date()): string {
  const bulan = ["Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"];
  return `${d.getDate()} ${bulan[d.getMonth()]} ${d.getFullYear()}`;
}

// Kelas tujuan jika NAIK: Sifr A→Kelas 1, Sifr B→Kelas 2, Kelas N→Kelas N+1, Kelas 6→LULUS.
function naikLabel(kelas: string): string {
  const k = (kelas || "").trim().toLowerCase();
  if (k === "sifr a") return "KELAS 1";
  if (k === "sifr b") return "KELAS 2";
  const m = k.match(/(\d+)/);
  if (m) {
    const n = parseInt(m[1], 10);
    return n >= 6 ? "LULUS" : `KELAS ${n + 1}`;
  }
  return "NAIK KELAS";
}

function density(n: number) {
  // kop = lebar maksimum gambar KOP (px). Tinggi otomatis (rasio asli 3.31:1) → tidak gepeng/lonjong.
  if (n <= 4) return { fz: 13, pad: "5px 9px", kop: 600, title: 15, ar: 12, rowH: 28, gap: 6, sigH: 34 };
  if (n <= 7) return { fz: 12, pad: "4px 7px", kop: 540, title: 13.5, ar: 11, rowH: 22, gap: 5, sigH: 30 };
  if (n <= 10) return { fz: 11, pad: "3px 6px", kop: 480, title: 12.5, ar: 10, rowH: 19, gap: 4, sigH: 26 };
  return { fz: 10, pad: "2px 5px", kop: 430, title: 11.5, ar: 9, rowH: 16, gap: 3, sigH: 22 };
}

// angka < 60 → merah
function skor(v: number | null | undefined, base: React.CSSProperties): React.CSSProperties {
  if (v != null && v < 60) return { ...base, color: "#dc2626" };
  return base;
}

// Satu lembar rapor (dipakai single & bulk)
function RaporSheet({ data, catatan }: { data: any; catatan?: string }) {
  const d = density(data?.nilai?.length || 0);
  const td: React.CSSProperties = { border: "1px solid #000", padding: d.pad, verticalAlign: "middle", lineHeight: 1.25 };
  const th: React.CSSProperties = { ...td, fontWeight: 800, textAlign: "center", background: "#f1f5f9" };
  const tbl: React.CSSProperties = { width: "100%", borderCollapse: "collapse", fontSize: d.fz };
  const ar: React.CSSProperties = { fontSize: d.ar, fontWeight: 400, direction: "rtl" };
  const arIn: React.CSSProperties = { fontSize: d.ar, fontWeight: 400, color: "#333" };
  const tahunTitle = data.periode?.tahun_ajaran ? `TAHUN PELAJARAN ${data.periode.tahun_ajaran} H` : "";
  const semester = data.periode?.semester === "genap" ? "GENAP" : "GANJIL";
  const semAngka = data.periode?.semester === "genap" ? "2" : "1";
  const tdC = { ...td, textAlign: "center" as const, fontWeight: 700 };

  return (
    <div className="rapor rapor-page" style={{ background: "#fff", maxWidth: 720, margin: "0 auto", width: "100%", padding: 18, color: "#000", fontSize: d.fz, display: "flex", flexDirection: "column", boxShadow: "var(--shadow)" }}>
      <img src="/kop-madrasah.png" alt="Madrasah Al Fath" style={{ width: "100%", maxWidth: d.kop, display: "block", margin: "0 auto 3px" }} />
      <div style={{ borderBottom: "2px solid #000", marginBottom: d.gap }} />

      <div style={{ textAlign: "center", fontWeight: 800, lineHeight: 1.2, marginBottom: d.gap, fontSize: d.title }}>
        <div>LAPORAN HASIL BELAJAR SANTRI SEMESTER {semester}</div>
        <div>{tahunTitle}</div>
      </div>

      <table style={tbl} cellSpacing={0}><tbody>
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
      </tbody></table>

      <table style={{ ...tbl, marginTop: d.gap }} cellSpacing={0}>
        <thead><tr>
          <th style={{ ...th, width: 36 }}>NO<div style={ar}>رقم</div></th>
          <th style={th}>PELAJARAN<div style={ar}>المواد الدراسية</div></th>
          <th style={th}>NAMA KITAB<div style={ar}>اسم الكتاب</div></th>
          <th style={{ ...th, width: 80 }}>NILAI<div style={ar}>المكتسبة</div></th>
          <th style={{ ...th, width: 80 }}>RATA-RATA<br />KELAS</th>
        </tr></thead>
        <tbody>
          {data.nilai.length === 0 && (
            <tr><td style={{ ...td, textAlign: "center", height: d.rowH }} colSpan={5}>Belum ada nilai pada periode ini.</td></tr>
          )}
          {data.nilai.map((it: any, i: number) => (
            <tr key={i}>
              <td style={{ ...td, textAlign: "center", height: d.rowH }}>{i + 1}</td>
              <td style={td}>{it.mata_pelajaran}</td>
              <td style={td}>{it.kitab || "-"}</td>
              <td style={skor(it.nilai_akhir, tdC)}>{it.nilai_akhir ?? "-"}</td>
              <td style={skor(it.rata_kelas, { ...td, textAlign: "center" })}>{it.rata_kelas ?? "-"}</td>
            </tr>
          ))}
          <tr>
            <td style={{ ...td, fontWeight: 700 }} colSpan={3}>JUMLAH NILAI <span style={arIn}>مجموع النتائج</span></td>
            <td style={tdC} colSpan={2}>{data.jumlah ?? "-"}</td>
          </tr>
          <tr>
            <td style={{ ...td, fontWeight: 700 }} colSpan={3}>RATA-RATA NILAI <span style={arIn}>كمية النتائج</span></td>
            <td style={skor(data.rata, tdC)} colSpan={2}>{data.rata ?? "-"}</td>
          </tr>
          <tr>
            <td style={{ ...td, fontWeight: 700 }} colSpan={3}>PERINGKAT <span style={arIn}>الرتبة</span></td>
            <td style={tdC} colSpan={2}>{data.peringkat || "-"}</td>
          </tr>
        </tbody>
      </table>

      <table style={{ ...tbl, marginTop: d.gap }} cellSpacing={0}>
        <thead><tr><th style={th} colSpan={2}>KETERANGAN</th><th style={th} colSpan={2}>KETERANGAN</th></tr></thead>
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

      {/* Catatan (memuai) — memuat teks kenaikan jika ada */}
      <div style={{ border: "1px solid #000", padding: d.pad, marginTop: d.gap, flex: "1 1 auto", minHeight: 28 }}>
        <strong>CATATAN <span style={arIn}>الإرشادات</span> :</strong>
        {catatan && <span style={{ fontWeight: 800, marginLeft: 8 }}>{catatan}</span>}
      </div>

      <div style={{ display: "flex", justifyContent: "space-between", marginTop: d.gap + 4, textAlign: "center", fontSize: d.fz }}>
        <div style={{ width: "32%" }}><div>Wali Santri</div><div style={ar}>ولي الطالب</div><div style={{ height: d.sigH }} /><div>(......................)</div></div>
        <div style={{ width: "32%" }}><div>Wali Kelas</div><div style={ar}>ولي الفصل</div><div style={{ height: d.sigH }} /><div>(......................)</div></div>
        <div style={{ width: "32%" }}><div>Malang, {tanggalIndo()}</div><div>Mudir Madrasah</div><div style={{ height: d.sigH }} /><div style={{ fontWeight: 700 }}>( {MUDIR} )</div></div>
      </div>
    </div>
  );
}

export default function RaporPage() {
  const [kelas, setKelas] = useState<Opt[]>([]);
  const [periode, setPeriode] = useState<Opt[]>([]);
  const [kelasId, setKelasId] = useState("");
  const [periodeId, setPeriodeId] = useState("");
  const [santriList, setSantriList] = useState<Santri[]>([]);
  const [santriId, setSantriId] = useState("");
  const [data, setData] = useState<any>(null);
  const [catatan, setCatatan] = useState("");
  const [bulk, setBulk] = useState<any[] | null>(null);
  const [zipState, setZipState] = useState<{ done: number; total: number } | null>(null);
  const hiddenRef = useRef<HTMLDivElement>(null);
  const runZip = useRef(false);

  useEffect(() => {
    api("/kelas?aktif=1").then(setKelas).catch(() => {});
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
    if (!santriId || !periodeId) { setData(null); setCatatan(""); return; }
    api(`/rapor?santri_id=${santriId}&periode_id=${periodeId}`).then((d) => {
      setData(d);
      // Saran awal catatan untuk semester genap (boleh diedit / dikosongkan).
      if (d?.periode?.semester === "genap") {
        const lbl = naikLabel(d.santri.kelas);
        setCatatan(lbl === "LULUS" ? "LULUS" : `NAIK ${lbl}`);
      } else {
        setCatatan("");
      }
    }).catch(() => {});
  }, [santriId, periodeId]);

  // Saat data bulk siap: render tersembunyi → tiap santri jadi 1 PDF → kemas ke ZIP → unduh.
  useEffect(() => {
    if (!bulk || !runZip.current) return;
    runZip.current = false;
    let cancelled = false;

    (async () => {
      const container = hiddenRef.current;
      if (!container) { setBulk(null); setZipState(null); return; }

      const [{ default: html2canvas }, { jsPDF }, { default: JSZip }] = await Promise.all([
        import("html2canvas-pro"),
        import("jspdf"),
        import("jszip"),
      ]);

      // tunggu gambar KOP termuat agar tidak kosong saat di-capture
      await Promise.all(
        Array.from(container.querySelectorAll("img")).map((img) =>
          img.complete ? Promise.resolve() : new Promise((res) => { img.onload = img.onerror = () => res(null); })
        )
      );
      await new Promise((r) => setTimeout(r, 120));

      const pages = Array.from(container.querySelectorAll<HTMLElement>(".rapor-page"));
      const zip = new JSZip();
      const dipakai: Record<string, number> = {};

      for (let i = 0; i < pages.length; i++) {
        if (cancelled) return;
        const canvas = await html2canvas(pages[i], { scale: 2, useCORS: true, backgroundColor: "#ffffff" });
        const img = canvas.toDataURL("image/jpeg", 0.92);
        const pdf = new jsPDF({ unit: "mm", format: "a4", orientation: "portrait" });
        const pw = 210, ph = 297;
        let w = pw, h = (canvas.height * pw) / canvas.width;
        if (h > ph) { h = ph; w = (canvas.width * ph) / canvas.height; }
        pdf.addImage(img, "JPEG", (pw - w) / 2, (ph - h) / 2, w, h);

        const s = bulk[i]?.santri || {};
        let base = `Rapor - ${[s.nama, s.kelas].filter(Boolean).join(" - ")}`
          .replace(/[\\/:*?"<>|]+/g, " ").replace(/\s+/g, " ").trim();
        if (dipakai[base] != null) base = `${base} (${++dipakai[base]})`; else dipakai[base] = 0;
        zip.file(`${base}.pdf`, pdf.output("blob"));
        setZipState({ done: i + 1, total: pages.length });
      }

      const s0 = bulk[0]?.santri || {};
      const ta = bulk[0]?.periode?.tahun_ajaran ? ` ${bulk[0].periode.tahun_ajaran}` : "";
      const zipName = `Rapor ${s0.kelas || "Kelas"}${ta}`
        .replace(/[\\/:*?"<>|]+/g, " ").replace(/\s+/g, " ").trim();

      const content = await zip.generateAsync({ type: "blob" });
      const url = URL.createObjectURL(content);
      const a = document.createElement("a");
      a.href = url; a.download = `${zipName}.zip`;
      document.body.appendChild(a); a.click(); a.remove();
      setTimeout(() => URL.revokeObjectURL(url), 4000);

      setBulk(null);
      setZipState(null);
    })().catch((e) => {
      console.error(e);
      alert("Gagal membuat ZIP: " + (e?.message || e));
      setBulk(null);
      setZipState(null);
    });

    return () => { cancelled = true; };
  }, [bulk]);

  async function unduhZip() {
    if (!kelasId || !periodeId || zipState) return;
    setZipState({ done: 0, total: 0 });
    try {
      const list: Santri[] = await api(`/santri?kelas_id=${kelasId}`);
      const hasil: any[] = [];
      for (const s of list) {
        try { hasil.push(await api(`/rapor?santri_id=${s.id}&periode_id=${periodeId}`)); } catch {}
      }
      if (hasil.length === 0) { setZipState(null); alert("Tidak ada data rapor untuk kelas ini."); return; }
      setZipState({ done: 0, total: hasil.length });
      runZip.current = true;
      setBulk(hasil); // render tersembunyi → memicu useEffect pembuat ZIP
    } catch (e: any) {
      setZipState(null);
      alert("Gagal mengambil data: " + (e?.message || e));
    }
  }

  // ===== MODE SINGLE =====
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
      <div className="no-print" style={{ display: "flex", flexDirection: "column", gap: 10 }}>
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
          <button className="btn secondary" onClick={unduhZip} disabled={!kelasId || !periodeId || !!zipState}>
            {zipState
              ? (zipState.total ? `Memproses ${zipState.done}/${zipState.total}…` : "Menyiapkan…")
              : "⬇ Unduh ZIP per Santri"}
          </button>
        </div>

        {zipState && zipState.total > 0 && (
          <p className="muted" style={{ margin: 0, fontSize: 13 }}>
            Membuat PDF {zipState.done}/{zipState.total} santri, lalu dikemas ke satu file ZIP…
          </p>
        )}

        {/* Catatan rapor (mis. keterangan kenaikan) — diketik bebas, tampil di bagian Catatan */}
        {data && (
          <div className="row" style={{ fontSize: 13, alignItems: "center" }}>
            <span className="muted" style={{ whiteSpace: "nowrap" }}>Catatan (muncul di rapor):</span>
            <input className="input" style={{ flex: 1, minWidth: 280 }}
              value={catatan} onChange={(e) => setCatatan(e.target.value)}
              placeholder='mis. "NAIK KELAS 4", "TINGGAL KELAS 3", atau catatan lain' />
            {catatan && (
              <button type="button" className="btn secondary" style={{ padding: "6px 10px" }} onClick={() => setCatatan("")}>Kosongkan</button>
            )}
          </div>
        )}
      </div>

      {!data && <p className="muted no-print">Pilih kelas, santri, dan periode untuk menampilkan rapor. Atau pilih kelas + periode lalu "Unduh ZIP per Santri" (1 file PDF per santri).</p>}

      {data && <RaporSheet data={data} catatan={catatan} />}

      {/* Kontainer tersembunyi: render rapor untuk di-capture saat membuat ZIP (tidak tampil & tidak ikut tercetak) */}
      <div ref={hiddenRef} aria-hidden className="no-print"
        style={{ position: "fixed", left: -10000, top: 0, width: 760, pointerEvents: "none", zIndex: -1 }}>
        {bulk?.map((d, i) => <RaporSheet key={i} data={d} />)}
      </div>
    </div>
  );
}
