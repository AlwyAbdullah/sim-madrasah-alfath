"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";

type Tautan = {
  tertaut: boolean;
  nama_telegram?: string;
  bot_username?: string;
  kode?: string;
  tautan?: string;
  berlaku_sampai?: string;
  token_ada: boolean;
};

export default function TelegramPage() {
  const [status, setStatus] = useState<Tautan | null>(null);
  const [kode, setKode] = useState<Tautan | null>(null);
  const [msg, setMsg] = useState("");
  const [sibuk, setSibuk] = useState(false);

  const muat = useCallback(async () => {
    try { setStatus(await api("/telegram/tautan")); } catch (e: any) { setMsg(e.message); }
  }, []);

  useEffect(() => { muat(); }, [muat]);

  // Selama menunggu, cek berkala apakah penautannya sudah masuk — supaya
  // halamannya berubah sendiri begitu pengguna menekan START di Telegram.
  useEffect(() => {
    if (!kode || status?.tertaut) return;
    const t = setInterval(async () => {
      try {
        const d: Tautan = await api("/telegram/tautan");
        if (d.tertaut) { setStatus(d); setKode(null); setMsg("✅ Telegram berhasil dihubungkan."); }
      } catch {}
    }, 4000);
    return () => clearInterval(t);
  }, [kode, status?.tertaut]);

  async function ambilKode() {
    setSibuk(true); setMsg("");
    try { setKode(await api("/telegram/tautan", { method: "POST" })); }
    catch (e: any) { setMsg("❌ " + e.message); }
    finally { setSibuk(false); }
  }

  async function lepas() {
    if (!confirm("Lepas hubungan dengan Telegram?\n\nPengingat tidak akan dikirim ke Anda lagi.")) return;
    setSibuk(true);
    try {
      await api("/telegram/tautan", { method: "DELETE" });
      setKode(null); setMsg("Tautan Telegram dilepas.");
      await muat();
    } catch (e: any) { setMsg("❌ " + e.message); }
    finally { setSibuk(false); }
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 14, maxWidth: 640 }}>
      <h1 style={{ margin: 0 }}>Hubungkan Telegram</h1>
      <p className="muted" style={{ margin: 0, fontSize: 13 }}>
        Kalau akun Anda terhubung ke Telegram, pengingat absensi kelas yang Anda ampu dikirim
        <strong> langsung ke HP Anda</strong> — bukan ke grup yang dibaca 20 orang. Pesannya masuk
        sebagai notifikasi Telegram biasa.
      </p>

      {status && !status.token_ada && (
        <div className="card" style={{ padding: 14, borderLeft: "4px solid var(--danger)" }}>
          Bot Telegram belum dipasang di server. Hubungi pengelola sistem.
        </div>
      )}

      {status?.tertaut ? (
        <div className="card" style={{ padding: 16 }}>
          <div className="row" style={{ justifyContent: "space-between", alignItems: "center", flexWrap: "wrap", gap: 10 }}>
            <div>
              <span className="badge hadir">Terhubung</span>
              <div style={{ marginTop: 6, fontSize: 14 }}>
                {status.nama_telegram || "Akun Telegram Anda"}
              </div>
              <div className="muted" style={{ fontSize: 12, marginTop: 2 }}>
                Pengingat absensi akan dikirim ke chat ini.
              </div>
            </div>
            <button className="btn secondary" style={{ color: "var(--danger)" }} disabled={sibuk} onClick={lepas}>
              Lepas hubungan
            </button>
          </div>
        </div>
      ) : kode ? (
        <div className="card" style={{ padding: 16 }}>
          <strong>Langkah terakhir</strong>
          <ol style={{ fontSize: 14, lineHeight: 1.8, paddingLeft: 20, marginTop: 8 }}>
            <li>
              Buka bot madrasah di Telegram:{" "}
              {kode.tautan
                ? <a href={kode.tautan} target="_blank" rel="noreferrer"><strong>@{kode.bot_username}</strong></a>
                : <strong>bot madrasah</strong>}
            </li>
            <li>Tekan <strong>START</strong>, atau kirim kode di bawah ini sebagai pesan.</li>
          </ol>

          <div style={{
            fontFamily: "monospace", fontSize: 32, letterSpacing: 6, textAlign: "center",
            padding: "14px 0", background: "#f8fafc", borderRadius: 10, margin: "10px 0",
          }}>
            {kode.kode}
          </div>

          <div className="row" style={{ gap: 8, flexWrap: "wrap" }}>
            {kode.tautan && (
              <a className="btn" href={kode.tautan} target="_blank" rel="noreferrer">Buka Telegram</a>
            )}
            <button className="btn secondary" onClick={() => { navigator.clipboard?.writeText(kode.kode || ""); setMsg("Kode disalin."); }}>
              Salin kode
            </button>
            <button className="btn secondary" disabled={sibuk} onClick={ambilKode}>Kode baru</button>
          </div>

          <div className="muted" style={{ fontSize: 12, marginTop: 10 }}>
            Kode berlaku sampai pukul {kode.berlaku_sampai}. Halaman ini akan berubah sendiri
            begitu Telegram Anda terhubung — tidak perlu dimuat ulang.
          </div>
        </div>
      ) : (
        <div className="card" style={{ padding: 16 }}>
          <div className="row" style={{ justifyContent: "space-between", alignItems: "center", flexWrap: "wrap", gap: 10 }}>
            <div>
              <span className="badge sakit">Belum terhubung</span>
              <div className="muted" style={{ fontSize: 12, marginTop: 4 }}>
                Butuh aplikasi Telegram di HP Anda. Prosesnya sekali saja.
              </div>
            </div>
            <button className="btn" disabled={sibuk || (status ? !status.token_ada : false)} onClick={ambilKode}>
              {sibuk ? "Menyiapkan…" : "Hubungkan sekarang"}
            </button>
          </div>
        </div>
      )}

      {msg && <div className="card" style={{ padding: 12 }}>{msg}</div>}

      <div className="muted" style={{ fontSize: 12 }}>
        Bot hanya membaca pesan yang Anda kirim langsung kepadanya. Di dalam grup, bot tidak bisa
        membaca percakapan biasa — hanya pesan yang menyebut namanya.
      </div>
    </div>
  );
}
