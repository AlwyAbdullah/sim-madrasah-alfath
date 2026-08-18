"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";

type Item = {
  id: number;
  nama: string;
  jenis: string;
  ref_tanggal: string | null;
  tujuan: string;
  pesan: string;
  status: string;
  percobaan: number;
  catatan: string | null;
  dikirim_at: string | null;
  created_at: string;
};

type TunggakanKelas = { kelas_id: number; kelas: string; total: number; belum: string[] };
type RekapSPP = {
  tahun: number;
  bulan: number;
  nama_bulan: string;
  total_santri: number;
  lunas: number;
  belum: number;
  kelas: TunggakanKelas[];
};
type PengaturanSPP = { aktif: boolean; tanggal: number; jam: number; menit: number };

const WARNA: Record<string, string> = {
  pending: "sakit",     // kuning
  terkirim: "hadir",    // hijau
  gagal: "alpha",       // merah
  batal: "izin",        // biru
};

const JENIS_LABEL: Record<string, string> = {
  pengingat_absensi: "Pengingat absensi",
  pengingat_spp: "Pengingat SPP",
};

function jam2(j: number, m: number) {
  return `${String(j).padStart(2, "0")}:${String(m).padStart(2, "0")}`;
}

export default function NotifikasiPage() {
  const [items, setItems] = useState<Item[]>([]);
  const [ringkasan, setRingkasan] = useState<Record<string, number>>({});
  const [status, setStatus] = useState("");
  const [msg, setMsg] = useState("");
  const [lihat, setLihat] = useState<Item | null>(null);

  const [aktif, setAktif] = useState<boolean | null>(null); // null = belum termuat
  const [mengubah, setMengubah] = useState(false);

  // pengaturan pengingat absensi harian
  const [ping, setPing] = useState<{ aktif: boolean; jam: number; menit: number } | null>(null);
  const [simpanPing, setSimpanPing] = useState(false);

  // pengaturan + rekap pengingat SPP bulanan
  const [spp, setSpp] = useState<PengaturanSPP | null>(null);
  const [rekap, setRekap] = useState<RekapSPP | null>(null);
  const [sppSibuk, setSppSibuk] = useState(false);
  const [rinci, setRinci] = useState(false);

  // pengaturan Telegram
  const [tg, setTg] = useState<{ token_terpasang: boolean; chat_id: string } | null>(null);
  const [tgChat, setTgChat] = useState("");
  const [tgSibuk, setTgSibuk] = useState(false);

  const load = useCallback(async () => {
    const d = await api(`/notifikasi${status ? `?status=${status}` : ""}`);
    setItems(d.items || []);
    setRingkasan(d.ringkasan || {});
  }, [status]);

  const loadSPP = useCallback(async () => {
    const d = await api("/pengingat-spp");
    setSpp(d.pengaturan);
    setRekap(d.rekap);
  }, []);

  useEffect(() => { load().catch((e) => setMsg(e.message)); }, [load]);
  useEffect(() => {
    api("/notifikasi/pengaturan").then((d) => setAktif(!!d.aktif)).catch(() => {});
    api("/pengingat-absensi").then((d) => setPing(d.pengaturan)).catch(() => {});
    api("/telegram/pengaturan").then((d) => { setTg(d); setTgChat(d.chat_id || ""); }).catch(() => {});
    loadSPP().catch(() => {});
  }, [loadSPP]);

  async function simpanTelegram() {
    setTgSibuk(true);
    try {
      const d = await api("/telegram/pengaturan", { method: "POST", body: { chat_id: tgChat } });
      setTg((t) => (t ? { ...t, chat_id: d.chat_id } : t));
      setMsg(d.chat_id ? `Tujuan Telegram disimpan: ${d.chat_id}` : "Tujuan Telegram dikosongkan.");
    } catch (e: any) { setMsg(e.message); }
    finally { setTgSibuk(false); }
  }

  async function ujiTelegram() {
    setTgSibuk(true);
    try {
      await api("/telegram/uji", { method: "POST" });
      setMsg("✅ Pesan uji Telegram terkirim — silakan cek Telegram Anda.");
    } catch (e: any) { setMsg("❌ " + e.message); }
    finally { setTgSibuk(false); }
  }

  async function simpanPengingat(baru: { aktif: boolean; jam: number; menit: number }) {
    setSimpanPing(true);
    try {
      const d = await api("/pengingat-absensi", { method: "POST", body: baru });
      setPing(d);
      setMsg(d.aktif
        ? `Pengingat absensi aktif setiap pukul ${jam2(d.jam, d.menit)}.`
        : "Pengingat absensi dimatikan.");
    } catch (e: any) { setMsg(e.message); }
    finally { setSimpanPing(false); }
  }

  async function simpanPengingatSPP(baru: PengaturanSPP) {
    setSppSibuk(true);
    try {
      const d = await api("/pengingat-spp", { method: "POST", body: baru });
      setSpp(d);
      setMsg(d.aktif
        ? `Pengingat SPP aktif setiap tanggal ${d.tanggal} pukul ${jam2(d.jam, d.menit)}.`
        : "Pengingat SPP dimatikan.");
    } catch (e: any) { setMsg(e.message); }
    finally { setSppSibuk(false); }
  }

  async function kirimSPPSekarang() {
    if (!confirm(
      `Kirim rekap tunggakan SPP ${rekap?.nama_bulan} ${rekap?.tahun} ke Telegram sekarang?\n\n` +
      `${rekap?.belum} dari ${rekap?.total_santri} santri belum membayar.`
    )) return;
    setSppSibuk(true);
    try {
      const d = await api("/pengingat-spp/kirim", { method: "POST" });
      setMsg("✅ " + d.message);
      await load();
    } catch (e: any) { setMsg("❌ " + e.message); }
    finally { setSppSibuk(false); }
  }

  async function toggleAktif() {
    const nilaiBaru = !aktif;
    if (!nilaiBaru && !confirm("Matikan pengiriman Telegram otomatis?\n\nPesan tetap tercatat di antrean seperti biasa, hanya saja tidak akan dikirim sampai diaktifkan kembali.")) return;
    setMengubah(true);
    try {
      const d = await api("/notifikasi/pengaturan", { method: "POST", body: { aktif: nilaiBaru } });
      setAktif(!!d.aktif);
      setMsg(d.aktif ? "Pengiriman Telegram diaktifkan." : "Pengiriman Telegram dimatikan — pesan baru tetap mengantre.");
    } catch (e: any) {
      setMsg(e.message);
    } finally {
      setMengubah(false);
    }
  }

  async function aksi(id: number, jenis: "ulang" | "batal") {
    try {
      await api(`/notifikasi/${id}/${jenis}`, { method: "POST" });
      setMsg(jenis === "ulang" ? "Dikembalikan ke antrean." : "Dibatalkan.");
      await load();
    } catch (e: any) { setMsg(e.message); }
  }

  const kelasBelum = (rekap?.kelas || []).filter((k) => k.belum.length > 0);

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
      <h1 style={{ margin: 0 }}>Notifikasi</h1>
      <p className="muted" style={{ margin: 0, fontSize: 13 }}>
        Pengingat otomatis dikirim ke <strong>Telegram</strong> lewat bot madrasah. Backend menyusun
        antrean di sini, lalu worker mengirimnya — jadi pesan tidak hilang walau pengiriman sedang dijeda.
      </p>

      <div className="card" style={{ padding: 14, display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12, flexWrap: "wrap" }}>
        <div>
          <strong>Pengiriman otomatis: </strong>
          {aktif === null ? (
            <span className="muted">memuat…</span>
          ) : (
            <span className={`badge ${aktif ? "hadir" : "alpha"}`}>{aktif ? "Aktif" : "Nonaktif"}</span>
          )}
          <div className="muted" style={{ fontSize: 12, marginTop: 2 }}>
            Saat nonaktif, pesan tetap tercatat di antrean seperti biasa — hanya tidak dikirim sampai diaktifkan lagi.
          </div>
        </div>
        <button
          className={aktif ? "btn secondary" : "btn"}
          onClick={toggleAktif}
          disabled={aktif === null || mengubah}
          style={aktif ? { color: "var(--danger)" } : undefined}
        >
          {mengubah ? "Menyimpan…" : aktif ? "🔕 Matikan" : "🔔 Aktifkan"}
        </button>
      </div>

      {/* ===== tujuan Telegram ===== */}
      <div className="card" style={{ padding: 14 }}>
        <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", gap: 12, flexWrap: "wrap" }}>
          <div style={{ minWidth: 260, flex: 1 }}>
            <strong>✈️ Tujuan Telegram: </strong>
            {tg === null ? <span className="muted">memuat…</span>
              : !tg.token_terpasang ? <span className="badge alpha">Token belum dipasang</span>
              : tg.chat_id ? <span className="badge hadir">Siap</span>
              : <span className="badge sakit">Tujuan chat belum diisi</span>}
            <div className="muted" style={{ fontSize: 12, marginTop: 4 }}>
              API resmi Telegram — gratis, tanpa nomor HP, tanpa risiko nomor diblokir.
              Semua pengingat dikirim ke chat/grup ini.
            </div>
            {tg && !tg.token_terpasang && (
              <div className="muted" style={{ fontSize: 12, marginTop: 6 }}>
                Langkah: buka <strong>@BotFather</strong> di Telegram → <code>/newbot</code> → salin token →
                minta pengelola memasangnya di server (<code>TELEGRAM_BOT_TOKEN</code>).
              </div>
            )}
          </div>
          {tg && (
            <div className="row" style={{ gap: 8, alignItems: "center" }}>
              <input className="input" style={{ width: 210 }} placeholder="chat_id (mis. -1001234567890)"
                value={tgChat} onChange={(e) => setTgChat(e.target.value)} />
              <button className="btn secondary" disabled={tgSibuk} onClick={simpanTelegram}>Simpan</button>
              <button className="btn" disabled={tgSibuk || !tg.token_terpasang || !tg.chat_id}
                onClick={ujiTelegram}>Kirim uji</button>
            </div>
          )}
        </div>
      </div>

      {/* ===== pengingat absensi harian ===== */}
      <div className="card" style={{ padding: 14 }}>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12, flexWrap: "wrap" }}>
          <div>
            <strong>⏰ Pengingat absensi harian: </strong>
            {ping === null ? <span className="muted">memuat…</span>
              : <span className={`badge ${ping.aktif ? "hadir" : "alpha"}`}>{ping.aktif ? "Aktif" : "Nonaktif"}</span>}
            <div className="muted" style={{ fontSize: 12, marginTop: 2 }}>
              Bila masih ada kelas yang belum diabsen, pengingat dikirim ke Telegram.
              Otomatis dilewati pada Kamis, Jumat, dan hari libur.
            </div>
          </div>
          {ping && (
            <div className="row" style={{ gap: 8 }}>
              <span className="muted" style={{ fontSize: 13 }}>Jam</span>
              <input className="input" type="time" style={{ width: 130 }}
                value={jam2(ping.jam, ping.menit)}
                onChange={(e) => {
                  const [j, m] = e.target.value.split(":").map(Number);
                  if (!isNaN(j) && !isNaN(m)) setPing({ ...ping, jam: j, menit: m });
                }} />
              <button className="btn secondary" disabled={simpanPing}
                onClick={() => simpanPengingat(ping)}>Simpan jam</button>
              <button className={ping.aktif ? "btn secondary" : "btn"} disabled={simpanPing}
                style={ping.aktif ? { color: "var(--danger)" } : undefined}
                onClick={() => simpanPengingat({ ...ping, aktif: !ping.aktif })}>
                {ping.aktif ? "🔕 Matikan" : "🔔 Aktifkan"}
              </button>
            </div>
          )}
        </div>
      </div>

      {/* ===== pengingat SPP bulanan ===== */}
      <div className="card" style={{ padding: 14 }}>
        <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", gap: 12, flexWrap: "wrap" }}>
          <div style={{ minWidth: 280, flex: 1 }}>
            <strong>💰 Pengingat SPP bulanan: </strong>
            {spp === null ? <span className="muted">memuat…</span>
              : <span className={`badge ${spp.aktif ? "hadir" : "alpha"}`}>{spp.aktif ? "Aktif" : "Nonaktif"}</span>}
            <div className="muted" style={{ fontSize: 12, marginTop: 2 }}>
              Sekali sebulan, daftar santri yang <strong>belum membayar SPP bulan berjalan</strong> dikirim ke Telegram.
              Santri yang belum punya catatan pembayaran sama sekali ikut terhitung belum bayar.
            </div>
            {rekap && (
              <div style={{ marginTop: 8 }}>
                <span className="badge alpha">Belum bayar {rekap.belum}</span>{" "}
                <span className="badge hadir">Lunas {rekap.lunas}</span>{" "}
                <span className="muted" style={{ fontSize: 12 }}>
                  dari {rekap.total_santri} santri · {rekap.nama_bulan} {rekap.tahun}
                </span>
                {kelasBelum.length > 0 && (
                  <button className="btn secondary" style={{ padding: "2px 8px", marginLeft: 8, fontSize: 12 }}
                    onClick={() => setRinci((v) => !v)}>
                    {rinci ? "Sembunyikan rincian" : "Lihat rincian"}
                  </button>
                )}
              </div>
            )}
            {rinci && kelasBelum.length > 0 && (
              <div style={{ marginTop: 10, display: "flex", flexDirection: "column", gap: 8 }}>
                {kelasBelum.map((k) => (
                  <div key={k.kelas_id} style={{ fontSize: 13 }}>
                    <strong>{k.kelas}</strong>{" "}
                    <span className="muted">— {k.belum.length} dari {k.total} belum bayar</span>
                    <div className="muted" style={{ fontSize: 12, marginTop: 2 }}>{k.belum.join(" · ")}</div>
                  </div>
                ))}
              </div>
            )}
          </div>
          {spp && (
            <div style={{ display: "flex", flexDirection: "column", gap: 8, alignItems: "flex-end" }}>
              <div className="row" style={{ gap: 8, alignItems: "center" }}>
                <span className="muted" style={{ fontSize: 13 }}>Tiap tanggal</span>
                <input className="input" type="number" min={1} max={28} style={{ width: 70 }}
                  value={spp.tanggal}
                  onChange={(e) => setSpp({ ...spp, tanggal: Number(e.target.value) })} />
                <input className="input" type="time" style={{ width: 130 }}
                  value={jam2(spp.jam, spp.menit)}
                  onChange={(e) => {
                    const [j, m] = e.target.value.split(":").map(Number);
                    if (!isNaN(j) && !isNaN(m)) setSpp({ ...spp, jam: j, menit: m });
                  }} />
              </div>
              <div className="row" style={{ gap: 8 }}>
                <button className="btn secondary" disabled={sppSibuk}
                  onClick={() => simpanPengingatSPP(spp)}>Simpan jadwal</button>
                <button className={spp.aktif ? "btn secondary" : "btn"} disabled={sppSibuk}
                  style={spp.aktif ? { color: "var(--danger)" } : undefined}
                  onClick={() => simpanPengingatSPP({ ...spp, aktif: !spp.aktif })}>
                  {spp.aktif ? "🔕 Matikan" : "🔔 Aktifkan"}
                </button>
                <button className="btn" disabled={sppSibuk || !rekap || rekap.belum === 0 || !tg?.chat_id}
                  onClick={kirimSPPSekarang}>Kirim sekarang</button>
              </div>
              <div className="muted" style={{ fontSize: 11 }}>
                Tanggal 1–28 agar selalu ada di setiap bulan.
              </div>
            </div>
          )}
        </div>
      </div>

      <div className="row" style={{ fontSize: 13 }}>
        <span className="badge hadir">Terkirim {ringkasan.terkirim || 0}</span>
        <span className="badge sakit">Menunggu {ringkasan.pending || 0}</span>
        <span className="badge alpha">Gagal {ringkasan.gagal || 0}</span>
        <span className="badge izin">Batal {ringkasan.batal || 0}</span>
        <select className="input" value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value="">— semua status —</option>
          <option value="pending">Menunggu</option>
          <option value="terkirim">Terkirim</option>
          <option value="gagal">Gagal</option>
          <option value="batal">Batal</option>
        </select>
        <button className="btn secondary" onClick={() => load().catch(() => {})}>↻ Muat ulang</button>
      </div>

      {msg && <div className="card" style={{ padding: 12 }}>{msg}</div>}

      {items.length === 0 ? (
        <p className="muted">Belum ada notifikasi. Pesan akan muncul otomatis sesuai jadwal pengingat di atas.</p>
      ) : (
        <div className="card table-wrap" style={{ padding: 0 }}>
          <table>
            <thead>
              <tr>
                <th style={{ width: 150 }}>Jenis</th>
                <th style={{ width: 100 }}>Tanggal</th>
                <th style={{ width: 130 }}>Tujuan</th>
                <th>Pesan</th>
                <th style={{ width: 90 }}>Status</th>
                <th style={{ width: 150 }}>Aksi</th>
              </tr>
            </thead>
            <tbody>
              {items.map((it) => (
                <tr key={it.id}>
                  <td style={{ fontSize: 13 }}>{JENIS_LABEL[it.jenis] || it.jenis}</td>
                  <td>{it.ref_tanggal || "-"}</td>
                  <td style={{ fontSize: 13 }}>{it.tujuan}</td>
                  <td>
                    <div style={{ fontSize: 12, color: "var(--muted)", maxWidth: 380, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                      {it.pesan.replace(/\n+/g, " ").slice(0, 90)}…
                    </div>
                    {it.catatan && <div style={{ fontSize: 12, color: "var(--danger)" }}>⚠ {it.catatan}</div>}
                  </td>
                  <td>
                    <span className={`badge ${WARNA[it.status] || ""}`}>{it.status}</span>
                    {it.percobaan > 0 && <div className="muted" style={{ fontSize: 11 }}>{it.percobaan}× coba</div>}
                  </td>
                  <td>
                    <div className="row" style={{ gap: 6 }}>
                      <button className="btn secondary" style={{ padding: "4px 8px" }} onClick={() => setLihat(it)}>Lihat</button>
                      {it.status !== "terkirim" && it.status !== "pending" && (
                        <button className="btn secondary" style={{ padding: "4px 8px" }} onClick={() => aksi(it.id, "ulang")}>Kirim ulang</button>
                      )}
                      {it.status === "pending" && (
                        <button className="btn secondary" style={{ padding: "4px 8px", color: "var(--danger)" }} onClick={() => aksi(it.id, "batal")}>Batal</button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {lihat && (
        <div className="card" style={{ padding: 16 }}>
          <div className="row" style={{ justifyContent: "space-between" }}>
            <strong>Isi pesan — {JENIS_LABEL[lihat.jenis] || lihat.jenis}</strong>
            <button className="btn secondary" style={{ padding: "4px 10px" }} onClick={() => setLihat(null)}>Tutup</button>
          </div>
          <pre style={{ whiteSpace: "pre-wrap", fontFamily: "inherit", fontSize: 14, background: "#f8fafc", padding: 12, borderRadius: 8, marginTop: 10 }}>
            {lihat.pesan}
          </pre>
          <div className="muted" style={{ fontSize: 12 }}>
            Ke: {lihat.tujuan} · dibuat {lihat.created_at}{lihat.dikirim_at ? ` · terkirim ${lihat.dikirim_at}` : ""}
          </div>
        </div>
      )}
    </div>
  );
}
