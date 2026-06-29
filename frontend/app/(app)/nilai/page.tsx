"use client";

import { useEffect, useState } from "react";
import { api, exportUrl } from "@/lib/api";

type Opt = { id: number; nama: string };
type Item = {
  santri_id: number;
  nama: string;
  nis: string;
  tugas: number | null;
  uts: number | null;
  uas: number | null;
};
type TugasEntry = { ke: number; nilai: number };
type TugasRow = { santri_id: number; nama: string; list: TugasEntry[]; rata: number | null };

// Bobot: Tugas 30% + UTS 30% + UAS 40%.
// Jika Tugas kosong → UTS 40% + UAS 60%. Jika Tugas & UTS kosong → UAS 100%.
function hitungAkhir(t: number | null, u: number | null, a: number | null): number | null {
  const v = (x: number | null) => (x == null ? 0 : x);
  const r = (n: number) => Math.round(n * 100) / 100;
  if (t == null && u == null && a == null) return null;
  if (t == null && u == null) return r(v(a));
  if (t == null) return r(v(u) * 0.4 + v(a) * 0.6);
  return r(t * 0.3 + v(u) * 0.3 + v(a) * 0.4);
}

export default function NilaiPage() {
  const [kelas, setKelas] = useState<Opt[]>([]);
  const [mapel, setMapel] = useState<Opt[]>([]);
  const [periode, setPeriode] = useState<Opt[]>([]);
  const [kelasId, setKelasId] = useState("");
  const [mapelId, setMapelId] = useState("");
  const [periodeId, setPeriodeId] = useState("");
  const [items, setItems] = useState<Item[]>([]);
  const [msg, setMsg] = useState("");
  const [saving, setSaving] = useState(false);

  // Tugas (riwayat ke-1..n)
  const [tugasRows, setTugasRows] = useState<TugasRow[]>([]);
  const [nextKe, setNextKe] = useState(1);
  const [editKe, setEditKe] = useState<number | null>(null);
  const [tugasInputs, setTugasInputs] = useState<Record<number, string>>({});
  const [savingTugas, setSavingTugas] = useState(false);

  useEffect(() => {
    api("/kelas?aktif=1").then(setKelas).catch(() => {});
    api("/periode").then(setPeriode).catch(() => {});
  }, []);

  useEffect(() => {
    setMapelId("");
    if (!kelasId) { setMapel([]); return; }
    api(`/kelas/${kelasId}/mapel`)
      .then((list: any[]) => setMapel(list.map((m) => ({ id: m.mata_pelajaran_id, nama: m.nama }))))
      .catch(() => setMapel([]));
  }, [kelasId]);

  const ready = kelasId && mapelId && periodeId;

  async function loadTugas() {
    if (!ready) { setTugasRows([]); setNextKe(1); return; }
    const d = await api(`/nilai/tugas?kelas_id=${kelasId}&mata_pelajaran_id=${mapelId}&periode_id=${periodeId}`);
    setTugasRows(d.items || []);
    setNextKe(d.next_ke || 1);
  }

  async function load() {
    if (!ready) { setItems([]); setTugasRows([]); return; }
    const d = await api(`/nilai?kelas_id=${kelasId}&mata_pelajaran_id=${mapelId}&periode_id=${periodeId}`);
    setItems(d.items);
    setEditKe(null);
    setMsg("");
    await loadTugas();
  }
  useEffect(() => { load(); /* eslint-disable-next-line */ }, [kelasId, mapelId, periodeId]);

  function setVal(id: number, field: "uts" | "uas", val: string) {
    const num = val === "" ? null : Math.max(0, Math.min(100, Number(val)));
    setItems((prev) => prev.map((it) => (it.santri_id === id ? { ...it, [field]: num } : it)));
  }

  // Simpan utama: HANYA UTS/UAS — tugas berasal dari rata-rata riwayat_tugas, jangan ditimpa.
  async function simpan() {
    setSaving(true);
    setMsg("");
    try {
      const d = await api("/nilai/batch", {
        method: "POST",
        body: {
          kelas_id: Number(kelasId),
          mata_pelajaran_id: Number(mapelId),
          periode_id: Number(periodeId),
          items: items.map((i) => ({ santri_id: i.santri_id, uts: i.uts, uas: i.uas })),
        },
      });
      setMsg(`UTS/UAS tersimpan: ${d.saved} santri.`);
      await load();
    } catch (e: any) {
      setMsg(e.message);
    } finally {
      setSaving(false);
    }
  }

  function tugasOf(santriId: number): TugasRow | undefined {
    return tugasRows.find((t) => t.santri_id === santriId);
  }

  function openTugas(ke: number) {
    setEditKe(ke);
    const init: Record<number, string> = {};
    for (const r of tugasRows) {
      const found = r.list.find((e) => e.ke === ke);
      init[r.santri_id] = found ? String(found.nilai) : "";
    }
    setTugasInputs(init);
    setMsg("");
  }

  async function simpanTugas() {
    if (editKe == null) return;
    setSavingTugas(true);
    setMsg("");
    try {
      const itemsT = Object.entries(tugasInputs)
        .filter(([, v]) => v !== "")
        .map(([sid, v]) => ({ santri_id: Number(sid), nilai: Math.max(0, Math.min(100, Number(v))) }));
      if (itemsT.length === 0) { setMsg("Isi minimal satu nilai tugas."); setSavingTugas(false); return; }
      const d = await api("/nilai/tugas/batch", {
        method: "POST",
        body: {
          kelas_id: Number(kelasId),
          mata_pelajaran_id: Number(mapelId),
          periode_id: Number(periodeId),
          ke: editKe,
          items: itemsT,
        },
      });
      setMsg(`Tugas ke-${d.ke} tersimpan: ${d.saved} santri.`);
      await load();
    } catch (e: any) {
      setMsg(e.message);
    } finally {
      setSavingTugas(false);
    }
  }

  function exportExcel() {
    const url = exportUrl(`/nilai/export?kelas_id=${kelasId}&mata_pelajaran_id=${mapelId}&periode_id=${periodeId}`);
    window.open(url, "_blank");
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
      <h1 style={{ margin: 0 }}>Input Nilai</h1>
      <p className="muted" style={{ margin: 0 }}>
        Bobot: Tugas 30% + UTS 30% + UAS 40%. Nilai <strong>Tugas = rata-rata Tugas ke-1..n</strong> — kelola di panel di bawah; kolom Tugas tidak diedit langsung.
      </p>

      <div className="row">
        <select className="input" value={kelasId} onChange={(e) => setKelasId(e.target.value)}>
          <option value="">— kelas —</option>
          {kelas.map((k) => <option key={k.id} value={k.id}>{k.nama}</option>)}
        </select>
        <select className="input" value={mapelId} onChange={(e) => setMapelId(e.target.value)}>
          <option value="">— mata pelajaran —</option>
          {mapel.map((m) => <option key={m.id} value={m.id}>{m.nama}</option>)}
        </select>
        <select className="input" value={periodeId} onChange={(e) => setPeriodeId(e.target.value)}>
          <option value="">— periode —</option>
          {periode.map((p) => <option key={p.id} value={p.id}>{p.nama}</option>)}
        </select>
        <button className="btn" onClick={simpan} disabled={saving || !items.length}>
          {saving ? "Menyimpan..." : "Simpan UTS/UAS"}
        </button>
        <button className="btn secondary" onClick={exportExcel} disabled={!ready || !items.length}>
          ⬇ Ekspor Excel
        </button>
      </div>

      {msg && <div className="card" style={{ padding: 12 }}>{msg}</div>}

      {ready && items.length > 0 && (
        <div className="card" style={{ padding: 12, display: "flex", flexDirection: "column", gap: 10 }}>
          <div className="row" style={{ justifyContent: "space-between", alignItems: "center" }}>
            <strong>Kelola Tugas</strong>
            <div className="row" style={{ gap: 6 }}>
              {Array.from({ length: nextKe - 1 }, (_, i) => i + 1).map((ke) => (
                <button key={ke} className="btn secondary"
                  style={{ padding: "5px 10px", outline: editKe === ke ? "2px solid var(--accent)" : "none" }}
                  onClick={() => openTugas(ke)}>
                  Tugas ke-{ke}
                </button>
              ))}
              <button className="btn" style={{ padding: "5px 10px" }} onClick={() => openTugas(nextKe)}>
                + Tugas ke-{nextKe}
              </button>
            </div>
          </div>
          {editKe != null && (
            <div className="table-wrap" style={{ padding: 0 }}>
              <table>
                <thead>
                  <tr>
                    <th>Nama</th>
                    <th style={{ width: 130 }}>Tugas ke-{editKe}</th>
                  </tr>
                </thead>
                <tbody>
                  {items.map((it) => (
                    <tr key={it.santri_id}>
                      <td>{it.nama}</td>
                      <td>
                        <input className="input" type="number" min={0} max={100} style={{ width: 90 }}
                          value={tugasInputs[it.santri_id] ?? ""}
                          onChange={(e) => setTugasInputs({ ...tugasInputs, [it.santri_id]: e.target.value })} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              <div className="row" style={{ marginTop: 10 }}>
                <button className="btn" onClick={simpanTugas} disabled={savingTugas}>
                  {savingTugas ? "Menyimpan..." : `Simpan Tugas ke-${editKe}`}
                </button>
                <button className="btn secondary" onClick={() => setEditKe(null)}>Tutup</button>
              </div>
            </div>
          )}
        </div>
      )}

      {items.length > 0 && (
        <div className="card table-wrap" style={{ padding: 0 }}>
          <table>
            <thead>
              <tr>
                <th style={{ width: 40 }}>No</th>
                <th>Nama</th>
                <th>Tugas (rata²)</th>
                <th style={{ width: 110 }}>UTS</th>
                <th style={{ width: 110 }}>UAS</th>
                <th style={{ width: 110 }}>Nilai Akhir</th>
              </tr>
            </thead>
            <tbody>
              {items.map((it, idx) => {
                const th = tugasOf(it.santri_id);
                return (
                  <tr key={it.santri_id}>
                    <td>{idx + 1}</td>
                    <td>
                      {it.nama}
                      <div className="muted" style={{ fontSize: 12 }}>{it.nis}</div>
                    </td>
                    <td>
                      <strong>{it.tugas ?? "-"}</strong>
                      {th && th.list.length > 0 && (
                        <div className="muted" style={{ fontSize: 12 }}>
                          {th.list.map((e) => `T${e.ke} ${e.nilai}`).join(" · ")}
                        </div>
                      )}
                    </td>
                    {(["uts", "uas"] as const).map((f) => (
                      <td key={f}>
                        <input className="input" type="number" min={0} max={100} style={{ width: 80 }}
                          value={it[f] ?? ""} onChange={(e) => setVal(it.santri_id, f, e.target.value)} />
                      </td>
                    ))}
                    <td><strong>{hitungAkhir(it.tugas, it.uts, it.uas) ?? "-"}</strong></td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {!ready && <p className="muted">Pilih kelas, mata pelajaran, dan periode untuk mulai input nilai.</p>}
    </div>
  );
}
