"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { api, exportUrl } from "@/lib/api";

type Kelas = { id: number; nama: string };
type Item = {
  santri_id: number;
  nis: string;
  nama: string;
  bulan: Record<string, boolean>; // key = bulan kalender (1..12)
  lunas: number;
};
type Ubah = { santri_id: number; bulan: number; lunas: boolean };

const BULAN = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];
const URUTAN = [7, 8, 9, 10, 11, 12, 1, 2, 3, 4, 5, 6]; // Juli..Juni

function taStartNow() {
  const d = new Date();
  return d.getMonth() + 1 >= 7 ? d.getFullYear() : d.getFullYear() - 1;
}

export default function SppPage() {
  const [kelas, setKelas] = useState<Kelas[]>([]);
  const [kelasId, setKelasId] = useState("");
  const [tahun, setTahun] = useState(taStartNow());
  const [items, setItems] = useState<Item[]>([]);
  const [msg, setMsg] = useState("");
  const [saving, setSaving] = useState(false);

  // ==== state untuk sapu (drag) ====
  const dragging = useRef(false);
  const dragValue = useRef(true);          // nilai yang "dikuaskan" selama menyapu
  const pending = useRef<Map<string, Ubah>>(new Map()); // perubahan yang belum dikirim

  useEffect(() => { api("/kelas?aktif=1").then(setKelas).catch(() => {}); }, []);

  const load = useCallback(async () => {
    if (!kelasId) { setItems([]); return; }
    const d = await api(`/spp?kelas_id=${kelasId}&tahun=${tahun}`);
    setItems((d.items || []).map((it: Item) => ({ ...it, bulan: it.bulan || {} })));
  }, [kelasId, tahun]);

  useEffect(() => { load().catch(() => {}); setMsg(""); }, [load]);

  // Terapkan perubahan ke tampilan (optimistis) + antrikan untuk disimpan.
  function terapkan(ubah: Ubah[]) {
    if (ubah.length === 0) return;
    setItems((prev) => prev.map((x) => {
      const milik = ubah.filter((u) => u.santri_id === x.santri_id);
      if (milik.length === 0) return x;
      const bln = { ...x.bulan };
      for (const u of milik) bln[u.bulan] = u.lunas;
      return { ...x, bulan: bln, lunas: Object.values(bln).filter(Boolean).length };
    }));
    for (const u of ubah) pending.current.set(`${u.santri_id}-${u.bulan}`, u);
  }

  // Kirim seluruh antrean dalam SATU request.
  const simpanAntrean = useCallback(async () => {
    const batch = Array.from(pending.current.values());
    if (batch.length === 0) return;
    pending.current.clear();
    setSaving(true);
    try {
      const d = await api("/spp/batch", { method: "POST", body: { tahun, items: batch } });
      setMsg(`Tersimpan ${d.saved} perubahan.`);
    } catch (e: any) {
      setMsg(e.message);
      load().catch(() => {}); // gagal → tarik ulang data asli
    } finally {
      setSaving(false);
    }
  }, [tahun, load]);

  // Lepas mouse di mana pun = akhiri sapuan lalu simpan.
  useEffect(() => {
    function up() {
      if (!dragging.current) return;
      dragging.current = false;
      simpanAntrean();
    }
    window.addEventListener("mouseup", up);
    window.addEventListener("touchend", up);
    return () => { window.removeEventListener("mouseup", up); window.removeEventListener("touchend", up); };
  }, [simpanAntrean]);

  function mulaiSapu(it: Item, bulan: number) {
    dragging.current = true;
    dragValue.current = !it.bulan[bulan];       // arah sapuan = kebalikan sel pertama
    terapkan([{ santri_id: it.santri_id, bulan, lunas: dragValue.current }]);
  }
  function lanjutSapu(it: Item, bulan: number) {
    if (!dragging.current) return;
    if (!!it.bulan[bulan] === dragValue.current) return; // sudah sesuai, lewati
    terapkan([{ santri_id: it.santri_id, bulan, lunas: dragValue.current }]);
  }

  // ==== aksi massal ====
  function toggleKolom(bulan: number) {
    const semuaLunas = items.every((x) => x.bulan[bulan]);
    terapkan(items.map((x) => ({ santri_id: x.santri_id, bulan, lunas: !semuaLunas })));
    simpanAntrean();
  }
  function toggleBaris(it: Item) {
    const semuaLunas = URUTAN.every((b) => it.bulan[b]);
    terapkan(URUTAN.map((b) => ({ santri_id: it.santri_id, bulan: b, lunas: !semuaLunas })));
    simpanAntrean();
  }
  function setSemua(lunas: boolean) {
    if (!lunas && !confirm("Kosongkan SEMUA centang SPP di kelas ini untuk satu tahun ajaran?")) return;
    const ubah: Ubah[] = [];
    for (const x of items) for (const b of URUTAN) ubah.push({ santri_id: x.santri_id, bulan: b, lunas });
    terapkan(ubah);
    simpanAntrean();
  }

  const years = [taStartNow() - 1, taStartNow(), taStartNow() + 1, taStartNow() + 2];
  const totalSlot = items.length * 12;
  const totalLunas = items.reduce((a, x) => a + x.lunas, 0);
  const persen = totalSlot ? Math.round((totalLunas / totalSlot) * 100) : 0;

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
      <h1 style={{ margin: 0 }}>Pembayaran SPP</h1>
      <p className="muted" style={{ margin: 0, fontSize: 13 }}>
        Tahun ajaran Juli–Juni. <strong>Klik &amp; seret</strong> untuk menandai banyak sekaligus · klik <strong>nama bulan</strong> = satu kolom · tombol <strong>✓</strong> di kanan = satu santri setahun. Tersimpan otomatis.
      </p>

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
        <div className="row" style={{ fontSize: 13 }}>
          <button className="btn secondary" onClick={() => setSemua(true)}>✓ Tandai Semua Lunas</button>
          <button className="btn secondary" onClick={() => setSemua(false)} style={{ color: "var(--danger)" }}>✕ Kosongkan Semua</button>
          <span className="muted">
            Lunas <strong>{totalLunas}</strong> / {totalSlot} slot ({persen}%)
          </span>
          {saving && <span className="muted">💾 menyimpan…</span>}
        </div>
      )}

      {msg && <div className="card" style={{ padding: 12 }}>{msg}</div>}

      {items.length > 0 && (
        <div className="card table-wrap" style={{ padding: 0 }}>
          <table style={{ userSelect: "none" }}>
            <thead>
              <tr>
                <th style={{ position: "sticky", left: 0, background: "#f8fafc", minWidth: 150, zIndex: 2 }}>Nama</th>
                {URUTAN.map((b) => {
                  const semua = items.every((x) => x.bulan[b]);
                  return (
                    <th key={b} onClick={() => toggleKolom(b)}
                      title={`Klik: tandai/kosongkan ${BULAN[b - 1]} untuk semua santri`}
                      style={{ textAlign: "center", padding: "8px 6px", cursor: "pointer", color: semua ? "var(--primary)" : undefined }}>
                      {BULAN[b - 1]}
                    </th>
                  );
                })}
                <th style={{ textAlign: "center" }}>Lunas</th>
              </tr>
            </thead>
            <tbody>
              {items.map((it) => (
                <tr key={it.santri_id}>
                  <td style={{ position: "sticky", left: 0, background: "#fff", zIndex: 1 }}>
                    {it.nama}<div className="muted" style={{ fontSize: 12 }}>{it.nis}</div>
                  </td>
                  {URUTAN.map((b) => {
                    const on = !!it.bulan[b];
                    return (
                      <td key={b}
                        onMouseDown={(e) => { e.preventDefault(); mulaiSapu(it, b); }}
                        onMouseEnter={() => lanjutSapu(it, b)}
                        title={`${it.nama} — ${BULAN[b - 1]}`}
                        style={{
                          textAlign: "center", padding: 0, cursor: "pointer",
                          background: on ? "#dcfce7" : undefined,
                        }}>
                        <div style={{
                          width: "100%", height: 34, display: "grid", placeItems: "center",
                          fontSize: 15, fontWeight: 700, color: on ? "#166534" : "#cbd5e1",
                        }}>
                          {on ? "✓" : "·"}
                        </div>
                      </td>
                    );
                  })}
                  <td style={{ textAlign: "center", whiteSpace: "nowrap" }}>
                    <span className={`badge ${it.lunas === 12 ? "hadir" : it.lunas === 0 ? "alpha" : "sakit"}`}>{it.lunas}/12</span>
                    <button className="btn secondary" onClick={() => toggleBaris(it)}
                      title="Tandai/kosongkan seluruh bulan untuk santri ini"
                      style={{ padding: "3px 8px", marginLeft: 6 }}>✓</button>
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
