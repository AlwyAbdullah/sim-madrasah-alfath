"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";

type Wali = { user_id: number; username: string; nama: string; urutan: number };
type Kelas = { id: number; nama: string; tingkat: string; aktif: boolean; wali: Wali[] };
type User = { id: number; username: string; nama: string; role: string; is_active: boolean };

export default function WaliKelasPage() {
  const [kelas, setKelas] = useState<Kelas[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [draft, setDraft] = useState<Record<number, number[]>>({});
  const [msg, setMsg] = useState("");
  const [sibuk, setSibuk] = useState<number | null>(null);

  const load = useCallback(async () => {
    const [k, u] = await Promise.all([api("/kelas"), api("/users")]);
    setKelas(k);
    setUsers((u as User[]).filter((x) => x.is_active));
    const d: Record<number, number[]> = {};
    for (const item of k as Kelas[]) d[item.id] = item.wali.map((w) => w.user_id);
    setDraft(d);
  }, []);

  useEffect(() => { load().catch((e) => setMsg(e.message)); }, [load]);

  function ubah(kelasID: number, userID: number, pilih: boolean) {
    const sekarang = draft[kelasID] || [];
    const baru = pilih
      ? [...sekarang, userID]
      : sekarang.filter((x) => x !== userID);
    setDraft({ ...draft, [kelasID]: baru });
  }

  async function simpan(k: Kelas) {
    setSibuk(k.id);
    try {
      await api(`/kelas/${k.id}/wali`, { method: "PUT", body: { user_ids: draft[k.id] || [] } });
      setMsg(`Wali ${k.nama} disimpan.`);
      await load();
    } catch (e: any) { setMsg("❌ " + e.message); }
    finally { setSibuk(null); }
  }

  function berubah(k: Kelas) {
    const asli = [...k.wali.map((w) => w.user_id)].sort().join(",");
    const baru = [...(draft[k.id] || [])].sort().join(",");
    return asli !== baru;
  }

  const aktif = kelas.filter((k) => k.aktif);
  const nonaktif = kelas.filter((k) => !k.aktif);

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
      <h1 style={{ margin: 0 }}>Wali Kelas</h1>
      <p className="muted" style={{ margin: 0, fontSize: 13 }}>
        Satu kelas boleh punya lebih dari satu wali. Kelas yang <strong>sudah</strong> punya wali akan
        menerima pengingat absensi atas nama walinya; kelas yang <strong>belum</strong> punya wali
        pengingatnya tetap dikirim ke grup, jadi tidak ada kelas yang luput.
      </p>

      {msg && <div className="card" style={{ padding: 12 }}>{msg}</div>}

      {[{ judul: "Kelas aktif", data: aktif }, { judul: "Kelas nonaktif", data: nonaktif }].map(
        (grup) => grup.data.length === 0 ? null : (
          <div key={grup.judul} style={{ display: "flex", flexDirection: "column", gap: 10 }}>
            <div className="nav-section" style={{ padding: 0 }}>{grup.judul}</div>
            {grup.data.map((k) => (
              <div key={k.id} className="card" style={{ padding: 14 }}>
                <div className="row" style={{ justifyContent: "space-between", alignItems: "center", flexWrap: "wrap", gap: 8 }}>
                  <div>
                    <strong>{k.nama}</strong>
                    <div className="muted" style={{ fontSize: 12 }}>
                      {k.wali.length > 0
                        ? `Wali sekarang: ${k.wali.map((w) => w.nama).join(", ")}`
                        : "Belum ada wali — pengingat dikirim ke grup"}
                    </div>
                  </div>
                  <button className="btn" disabled={sibuk === k.id || !berubah(k)} onClick={() => simpan(k)}>
                    {sibuk === k.id ? "Menyimpan…" : "Simpan"}
                  </button>
                </div>

                <div style={{ display: "flex", flexWrap: "wrap", gap: 8, marginTop: 10 }}>
                  {users.map((u) => {
                    const pilih = (draft[k.id] || []).includes(u.id);
                    return (
                      <label key={u.id}
                        style={{
                          display: "flex", alignItems: "center", gap: 6, fontSize: 13,
                          padding: "4px 10px", borderRadius: 999, cursor: "pointer",
                          border: `1px solid ${pilih ? "var(--primary, #2563eb)" : "var(--border, #e2e8f0)"}`,
                          background: pilih ? "rgba(37,99,235,0.08)" : "transparent",
                        }}>
                        <input type="checkbox" checked={pilih}
                          onChange={(e) => ubah(k.id, u.id, e.target.checked)} />
                        {u.nama}
                      </label>
                    );
                  })}
                </div>
              </div>
            ))}
          </div>
        )
      )}
    </div>
  );
}
