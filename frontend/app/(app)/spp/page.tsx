"use client";

import { Fragment, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { api, exportUrl } from "@/lib/api";

type Kelas = { id: number; nama: string };
type Item = {
  santri_id: number;
  nis: string;
  nama: string;
  kelas_id: number;
  kelas: string;
  bulan: Record<string, boolean>;
  ket: Record<string, string>;
  lunas: number;
};
type Riwayat = {
  batch_id: string;
  aksi: string;
  jumlah_sel: number;
  tahun_ajaran: number;
  dikembalikan: boolean;
  oleh: string;
  waktu: string;
};
/** Satu sel yang diubah tapi BELUM disimpan. */
type Draft = { lunas: boolean; ket?: string };

const BULAN = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];
const URUTAN = [7, 8, 9, 10, 11, 12, 1, 2, 3, 4, 5, 6]; // Juli..Juni

function taStartNow() {
  const d = new Date();
  return d.getMonth() + 1 >= 7 ? d.getFullYear() : d.getFullYear() - 1;
}
const kunci = (santriId: number, bulan: number) => `${santriId}-${bulan}`;

export default function SppPage() {
  const [kelas, setKelas] = useState<Kelas[]>([]);
  const [kelasId, setKelasId] = useState(""); // "" = semua kelas
  const [tahun, setTahun] = useState(taStartNow());
  const [items, setItems] = useState<Item[]>([]);
  const [cari, setCari] = useState("");
  const [msg, setMsg] = useState("");
  const [memuat, setMemuat] = useState(false);
  const [menyimpan, setMenyimpan] = useState(false);

  // ===== draft (perubahan belum tersimpan) =====
  const [draft, setDraft] = useState<Map<string, Draft>>(new Map());
  const undoStack = useRef<Map<string, Draft>[]>([]);
  const redoStack = useRef<Map<string, Draft>[]>([]);
  // penghitung agar tombol Undo/Redo ikut ter-render ulang (ref sendiri tidak memicu render)
  const [histori, setHistori] = useState({ undo: 0, redo: 0 });
  const segarkanHistori = () =>
    setHistori({ undo: undoStack.current.length, redo: redoStack.current.length });

  // ===== riwayat =====
  const [riwayat, setRiwayat] = useState<Riwayat[]>([]);
  const [bukaRiwayat, setBukaRiwayat] = useState(false);

  // ===== sapu (drag) =====
  const menyapu = useRef(false);
  const nilaiSapu = useRef(true);

  // ===== keterangan =====
  const [editKet, setEditKet] = useState<{ santriId: number; bulan: number; nama: string } | null>(null);
  const [ketDraft, setKetDraft] = useState("");

  useEffect(() => { api("/kelas").then(setKelas).catch(() => {}); }, []);

  const muat = useCallback(async () => {
    setMemuat(true);
    try {
      const d = await api(`/spp?tahun=${tahun}${kelasId ? `&kelas_id=${kelasId}` : ""}`);
      setItems((d.items || []).map((it: Item) => ({ ...it, bulan: it.bulan || {}, ket: it.ket || {} })));
    } catch (e: any) { setMsg(e.message); }
    finally { setMemuat(false); }
  }, [tahun, kelasId]);

  const muatRiwayat = useCallback(async () => {
    try { setRiwayat((await api("/spp/riwayat?limit=30")).items || []); } catch { /* abaikan */ }
  }, []);

  useEffect(() => { muat(); }, [muat]);
  useEffect(() => { muatRiwayat(); }, [muatRiwayat]);

  // Ganti tahun/kelas: draft lama tidak boleh terbawa ke kumpulan data yang berbeda.
  const konteks = `${tahun}|${kelasId}`;
  const konteksSebelumnya = useRef(konteks);
  useEffect(() => {
    if (konteksSebelumnya.current === konteks) return;
    konteksSebelumnya.current = konteks;
    setDraft(new Map());
    undoStack.current = [];
    redoStack.current = [];
    setHistori({ undo: 0, redo: 0 });
  }, [konteks]);

  // Peringatan bila menutup/menyegarkan tab saat masih ada perubahan.
  useEffect(() => {
    if (draft.size === 0) return;
    const h = (e: BeforeUnloadEvent) => { e.preventDefault(); e.returnValue = ""; };
    window.addEventListener("beforeunload", h);
    return () => window.removeEventListener("beforeunload", h);
  }, [draft.size]);

  // ===== nilai efektif (DB + draft) =====
  const nilaiLunas = useCallback((it: Item, b: number): boolean => {
    const d = draft.get(kunci(it.santri_id, b));
    return d ? d.lunas : !!it.bulan[b];
  }, [draft]);

  const nilaiKet = useCallback((it: Item, b: number): string => {
    const d = draft.get(kunci(it.santri_id, b));
    if (d && d.ket !== undefined) return d.ket;
    return it.ket[b] || "";
  }, [draft]);

  const berubah = useCallback((it: Item, b: number): boolean => draft.has(kunci(it.santri_id, b)), [draft]);

  // ===== perubahan draft (dengan undo) =====
  function terapkan(ubah: { santriId: number; bulan: number; lunas: boolean; ket?: string }[], catatUndo = true) {
    if (ubah.length === 0) return;
    // dicatat di luar updater agar tidak terdorong dua kali (React StrictMode)
    if (catatUndo) {
      undoStack.current.push(new Map(draft));
      redoStack.current = [];
      if (undoStack.current.length > 100) undoStack.current.shift();
      segarkanHistori();
    }
    setDraft((prev) => {
      const next = new Map(prev);
      for (const u of ubah) {
        const it = items.find((x) => x.santri_id === u.santriId);
        const k = kunci(u.santriId, u.bulan);
        const ketBaru = u.ket !== undefined ? u.ket : (next.get(k)?.ket ?? undefined);
        // kembali ke nilai asal DB → buang dari draft agar tidak ikut terkirim
        const samaDenganDB = it && !!it.bulan[u.bulan] === u.lunas &&
          (ketBaru === undefined || ketBaru === (it.ket[u.bulan] || ""));
        if (samaDenganDB) next.delete(k);
        else next.set(k, { lunas: u.lunas, ket: ketBaru });
      }
      return next;
    });
    setMsg("");
  }

  function undo() {
    const prev = undoStack.current.pop();
    if (!prev) return;
    redoStack.current.push(new Map(draft));
    setDraft(prev);
    segarkanHistori();
  }
  function redo() {
    const next = redoStack.current.pop();
    if (!next) return;
    undoStack.current.push(new Map(draft));
    setDraft(next);
    segarkanHistori();
  }
  function buangDraft() {
    if (draft.size === 0) return;
    if (!confirm(`Batalkan ${draft.size} perubahan yang belum disimpan?`)) return;
    undoStack.current.push(new Map(draft));
    redoStack.current = [];
    setDraft(new Map());
    segarkanHistori();
    setMsg("Perubahan dibatalkan.");
  }

  // pintasan papan ketik
  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if (editKet) return;
      const ctrl = e.ctrlKey || e.metaKey;
      if (ctrl && e.key.toLowerCase() === "z" && !e.shiftKey) { e.preventDefault(); undo(); }
      else if (ctrl && (e.key.toLowerCase() === "y" || (e.key.toLowerCase() === "z" && e.shiftKey))) { e.preventDefault(); redo(); }
      else if (ctrl && e.key.toLowerCase() === "s") { e.preventDefault(); simpan(); }
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  });

  // lepas mouse di mana pun = akhiri sapuan
  useEffect(() => {
    const up = () => { menyapu.current = false; };
    window.addEventListener("mouseup", up);
    window.addEventListener("touchend", up);
    return () => { window.removeEventListener("mouseup", up); window.removeEventListener("touchend", up); };
  }, []);

  function mulaiSapu(it: Item, b: number) {
    menyapu.current = true;
    nilaiSapu.current = !nilaiLunas(it, b);
    terapkan([{ santriId: it.santri_id, bulan: b, lunas: nilaiSapu.current }]);
  }
  function lanjutSapu(it: Item, b: number) {
    if (!menyapu.current || nilaiLunas(it, b) === nilaiSapu.current) return;
    terapkan([{ santriId: it.santri_id, bulan: b, lunas: nilaiSapu.current }], false);
  }

  // ===== aksi massal =====
  function toggleKolom(b: number) {
    const semua = tampil.every((x) => nilaiLunas(x, b));
    terapkan(tampil.map((x) => ({ santriId: x.santri_id, bulan: b, lunas: !semua })));
  }
  function toggleBaris(it: Item) {
    const semua = URUTAN.every((b) => nilaiLunas(it, b));
    terapkan(URUTAN.map((b) => ({ santriId: it.santri_id, bulan: b, lunas: !semua })));
  }

  // ===== simpan =====
  async function simpan() {
    if (draft.size === 0 || menyimpan) return;
    setMenyimpan(true);
    setMsg("");
    try {
      const daftar = Array.from(draft.entries()).map(([k, v]) => {
        const [sid, bln] = k.split("-").map(Number);
        return { santri_id: sid, bulan: bln, lunas: v.lunas, keterangan: v.ket ?? null };
      });
      const d = await api("/spp/batch", { method: "POST", body: { tahun, items: daftar } });
      setDraft(new Map());
      undoStack.current = []; redoStack.current = [];
      setMsg(d.saved > 0 ? `Tersimpan: ${d.saved} perubahan.` : "Tidak ada perubahan nyata untuk disimpan.");
      await muat();
      await muatRiwayat();
    } catch (e: any) { setMsg(e.message); }
    finally { setMenyimpan(false); }
  }

  async function kembalikan(b: Riwayat) {
    if (draft.size > 0) { setMsg("Simpan atau batalkan perubahan Anda dulu sebelum mengembalikan riwayat."); return; }
    if (!confirm(`Kembalikan ${b.jumlah_sel} sel ke kondisi sebelum perubahan ${b.waktu} oleh ${b.oleh}?`)) return;
    try {
      const d = await api(`/spp/riwayat/${b.batch_id}/kembalikan`, { method: "POST" });
      setMsg(`${d.dikembalikan} sel dikembalikan ke nilai sebelumnya.`);
      await muat();
      await muatRiwayat();
    } catch (e: any) { setMsg(e.message); }
  }

  // ===== keterangan =====
  function bukaKet(it: Item, b: number) {
    setEditKet({ santriId: it.santri_id, bulan: b, nama: it.nama });
    setKetDraft(nilaiKet(it, b));
  }
  function simpanKet() {
    if (!editKet) return;
    const it = items.find((x) => x.santri_id === editKet.santriId);
    if (it) terapkan([{ santriId: editKet.santriId, bulan: editKet.bulan, lunas: nilaiLunas(it, editKet.bulan), ket: ketDraft }]);
    setEditKet(null);
  }

  // ===== penyaringan & pengelompokan =====
  const tampil = useMemo(() => {
    const q = cari.toLowerCase().trim();
    if (!q) return items;
    return items.filter((x) => x.nama.toLowerCase().includes(q) || (x.nis || "").toLowerCase().includes(q));
  }, [items, cari]);

  const grup = useMemo(() => {
    const g: { kelas: string; anggota: Item[] }[] = [];
    for (const it of tampil) {
      const akhir = g[g.length - 1];
      if (akhir && akhir.kelas === it.kelas) akhir.anggota.push(it);
      else g.push({ kelas: it.kelas, anggota: [it] });
    }
    return g;
  }, [tampil]);

  const years = [taStartNow() - 1, taStartNow(), taStartNow() + 1, taStartNow() + 2];
  const totalSlot = tampil.length * 12;
  const totalLunas = tampil.reduce((a, x) => a + URUTAN.filter((b) => nilaiLunas(x, b)).length, 0);
  const persen = totalSlot ? Math.round((totalLunas / totalSlot) * 100) : 0;

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
      <h1 style={{ margin: 0 }}>Pembayaran SPP</h1>
      <p className="muted" style={{ margin: 0, fontSize: 13 }}>
        Tahun ajaran Juli–Juni. Perubahan <strong>tidak langsung tersimpan</strong> — tekan <strong>Simpan</strong>.
        Klik &amp; seret untuk menandai banyak sel · klik <strong>nama bulan</strong> = satu kolom ·
        <strong> klik kanan</strong> sel = keterangan · <strong>Ctrl+Z</strong> untuk membatalkan.
      </p>

      {/* ===== bilah aksi ===== */}
      <div className="card" style={{ padding: 12, display: "flex", gap: 10, alignItems: "center", flexWrap: "wrap" }}>
        <select className="input" value={kelasId} onChange={(e) => setKelasId(e.target.value)}>
          <option value="">Semua kelas</option>
          {kelas.map((k) => <option key={k.id} value={k.id}>{k.nama}</option>)}
        </select>
        <select className="input" value={tahun} onChange={(e) => setTahun(Number(e.target.value))}>
          {years.map((y) => <option key={y} value={y}>{y}/{y + 1}</option>)}
        </select>
        <input className="input" placeholder="Cari nama / NIS…" value={cari}
          onChange={(e) => setCari(e.target.value)} style={{ minWidth: 190 }} />

        <div style={{ flex: 1 }} />

        <button className="btn secondary" onClick={undo} disabled={histori.undo === 0} title="Ctrl+Z">↶ Undo</button>
        <button className="btn secondary" onClick={redo} disabled={histori.redo === 0} title="Ctrl+Y">↷ Redo</button>
        <button className="btn secondary" onClick={buangDraft} disabled={draft.size === 0} style={{ color: "var(--danger)" }}>
          Batalkan
        </button>
        <button className="btn" onClick={simpan} disabled={draft.size === 0 || menyimpan} title="Ctrl+S">
          {menyimpan ? "Menyimpan…" : draft.size > 0 ? `💾 Simpan ${draft.size} perubahan` : "💾 Simpan"}
        </button>
      </div>

      <div className="row" style={{ fontSize: 13 }}>
        {draft.size > 0 && (
          <span className="badge sakit">⚠ {draft.size} perubahan belum disimpan</span>
        )}
        <span className="muted">
          {tampil.length} santri · lunas <strong>{totalLunas}</strong> / {totalSlot} ({persen}%)
        </span>
        <button className="btn secondary" style={{ padding: "4px 10px" }}
          onClick={() => { setBukaRiwayat(!bukaRiwayat); muatRiwayat(); }}>
          🕘 Riwayat {bukaRiwayat ? "▲" : "▼"}
        </button>
        <button className="btn secondary" style={{ padding: "4px 10px" }}
          onClick={() => window.open(exportUrl(`/spp/export?kelas_id=${kelasId || items[0]?.kelas_id || ""}&tahun=${tahun}`), "_blank")}
          disabled={!items.length}>⬇ Ekspor Excel</button>
      </div>

      {msg && <div className="card" style={{ padding: 12 }}>{msg}</div>}

      {/* ===== panel riwayat ===== */}
      {bukaRiwayat && (
        <div className="card" style={{ padding: 0 }}>
          <div style={{ padding: "10px 14px", borderBottom: "1px solid var(--border)", fontWeight: 600 }}>
            Riwayat Perubahan — bisa dikembalikan bila ada kesalahan
          </div>
          {riwayat.length === 0 ? (
            <p className="muted" style={{ padding: 14, margin: 0 }}>Belum ada perubahan tercatat.</p>
          ) : (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Waktu</th><th>Oleh</th><th>Aksi</th>
                    <th style={{ width: 90 }}>Jumlah</th><th style={{ width: 150 }}></th>
                  </tr>
                </thead>
                <tbody>
                  {riwayat.map((b) => (
                    <tr key={b.batch_id}>
                      <td style={{ fontSize: 13 }}>{b.waktu}</td>
                      <td>{b.oleh}</td>
                      <td>
                        {b.aksi === "kembalikan"
                          ? <span className="badge izin">pengembalian</span>
                          : <span className="badge hadir">simpan</span>}
                        {b.dikembalikan && <span className="badge alpha" style={{ marginLeft: 6 }}>sudah dikembalikan</span>}
                      </td>
                      <td>{b.jumlah_sel} sel</td>
                      <td>
                        {!b.dikembalikan && b.aksi === "simpan" && (
                          <button className="btn secondary" style={{ padding: "4px 10px" }}
                            onClick={() => kembalikan(b)}>↩ Kembalikan</button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* ===== dialog keterangan ===== */}
      {editKet && (
        <div className="card" style={{ padding: 14 }}>
          <div className="row" style={{ justifyContent: "space-between" }}>
            <strong>Keterangan — {editKet.nama}, {BULAN[editKet.bulan - 1]}</strong>
            <button className="btn secondary" style={{ padding: "4px 10px" }} onClick={() => setEditKet(null)}>Tutup</button>
          </div>
          <input className="input" style={{ width: "100%", marginTop: 10 }} autoFocus
            maxLength={255} placeholder="mis. bayar tunai ke bendahara / dicicil"
            value={ketDraft} onChange={(e) => setKetDraft(e.target.value)}
            onKeyDown={(e) => { if (e.key === "Enter") simpanKet(); if (e.key === "Escape") setEditKet(null); }} />
          <div className="row" style={{ marginTop: 10 }}>
            <button className="btn" onClick={simpanKet}>Terapkan</button>
            <button className="btn secondary" onClick={() => { setKetDraft(""); }}>Kosongkan</button>
            <span className="muted" style={{ fontSize: 12 }}>Masih perlu ditekan <strong>Simpan</strong> agar masuk database.</span>
          </div>
        </div>
      )}

      {/* ===== tabel ===== */}
      {memuat ? (
        <p className="muted">Memuat…</p>
      ) : tampil.length === 0 ? (
        <p className="muted">{cari ? `Tidak ada santri cocok dengan "${cari}".` : "Belum ada data santri."}</p>
      ) : (
        <div className="card table-wrap" style={{ padding: 0, maxHeight: "70vh", overflow: "auto" }}>
          <table style={{ userSelect: "none" }}>
            <thead>
              <tr>
                <th style={{ position: "sticky", left: 0, top: 0, zIndex: 3, background: "#f8fafc", minWidth: 170 }}>Nama</th>
                {URUTAN.map((b) => (
                  <th key={b} onClick={() => toggleKolom(b)}
                    title={`Klik: tandai/kosongkan ${BULAN[b - 1]} untuk semua yang tampil`}
                    style={{ position: "sticky", top: 0, zIndex: 2, background: "#f8fafc", textAlign: "center", padding: "8px 6px", cursor: "pointer" }}>
                    {BULAN[b - 1]}
                  </th>
                ))}
                <th style={{ position: "sticky", top: 0, zIndex: 2, background: "#f8fafc", textAlign: "center" }}>Lunas</th>
              </tr>
            </thead>
            <tbody>
              {grup.map((g) => (
                <Fragment key={`grup-${g.kelas}`}>
                  <tr>
                    <td colSpan={14} style={{ background: "#eef2f7", fontWeight: 700, fontSize: 13, position: "sticky", left: 0 }}>
                      {g.kelas} — {g.anggota.length} santri
                    </td>
                  </tr>
                  {g.anggota.map((it) => {
                    const lunasBaris = URUTAN.filter((b) => nilaiLunas(it, b)).length;
                    return (
                      <tr key={it.santri_id}>
                        <td style={{ position: "sticky", left: 0, background: "#fff", zIndex: 1 }}>
                          {it.nama}<div className="muted" style={{ fontSize: 12 }}>{it.nis}</div>
                        </td>
                        {URUTAN.map((b) => {
                          const on = nilaiLunas(it, b);
                          const ubah = berubah(it, b);
                          const ket = nilaiKet(it, b);
                          return (
                            <td key={b}
                              onMouseDown={(e) => { e.preventDefault(); mulaiSapu(it, b); }}
                              onMouseEnter={() => lanjutSapu(it, b)}
                              onContextMenu={(e) => { e.preventDefault(); bukaKet(it, b); }}
                              title={ket ? `${it.nama} — ${BULAN[b - 1]}\n📝 ${ket}` : `${it.nama} — ${BULAN[b - 1]}\n(klik kanan untuk keterangan)`}
                              style={{
                                textAlign: "center", padding: 0, cursor: "pointer", position: "relative",
                                background: ubah ? (on ? "#bbf7d0" : "#fee2e2") : (on ? "#dcfce7" : undefined),
                                boxShadow: ubah ? "inset 0 0 0 2px #f59e0b" : undefined,
                              }}>
                              <div style={{ width: "100%", height: 34, display: "grid", placeItems: "center", fontSize: 15, fontWeight: 700, color: on ? "#166534" : "#cbd5e1" }}>
                                {on ? "✓" : "·"}
                              </div>
                              {ket && (
                                <span title={ket} style={{ position: "absolute", top: 1, right: 1, width: 0, height: 0,
                                  borderTop: "7px solid #2563eb", borderLeft: "7px solid transparent" }} />
                              )}
                            </td>
                          );
                        })}
                        <td style={{ textAlign: "center", whiteSpace: "nowrap" }}>
                          <span className={`badge ${lunasBaris === 12 ? "hadir" : lunasBaris === 0 ? "alpha" : "sakit"}`}>{lunasBaris}/12</span>
                          <button className="btn secondary" onClick={() => toggleBaris(it)}
                            title="Tandai/kosongkan seluruh bulan untuk santri ini"
                            style={{ padding: "3px 8px", marginLeft: 6 }}>✓</button>
                        </td>
                      </tr>
                    );
                  })}
                </Fragment>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
