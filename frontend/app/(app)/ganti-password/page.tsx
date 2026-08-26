"use client";

import { useState } from "react";
import { api } from "@/lib/api";

export default function GantiPasswordPage() {
  const [lama, setLama] = useState("");
  const [baru, setBaru] = useState("");
  const [ulang, setUlang] = useState("");
  const [msg, setMsg] = useState("");
  const [sibuk, setSibuk] = useState(false);

  const cocok = baru !== "" && baru === ulang;
  const cukupPanjang = baru.length >= 6;
  const bisaSimpan = lama !== "" && cocok && cukupPanjang && !sibuk;

  async function simpan(e: React.FormEvent) {
    e.preventDefault();
    setSibuk(true);
    setMsg("");
    try {
      await api("/auth/ganti-password", {
        method: "POST",
        body: { password_lama: lama, password_baru: baru },
      });
      setMsg("✅ Password berhasil diganti. Pakai password baru saat login berikutnya.");
      setLama(""); setBaru(""); setUlang("");
    } catch (e: any) {
      setMsg("❌ " + e.message);
    } finally {
      setSibuk(false);
    }
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 14, maxWidth: 460 }}>
      <h1 style={{ margin: 0 }}>Ganti Password</h1>
      <p className="muted" style={{ margin: 0, fontSize: 13 }}>
        Kalau password Anda masih <code>guru123</code> (bawaan), sebaiknya diganti — selama masih
        bawaan, siapa pun yang tahu polanya bisa masuk memakai akun Anda.
      </p>

      <form className="card" style={{ padding: 16, display: "flex", flexDirection: "column", gap: 12 }} onSubmit={simpan}>
        <label style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          <span style={{ fontSize: 13 }}>Password lama</span>
          <input className="input" type="password" value={lama} autoComplete="current-password"
            onChange={(e) => setLama(e.target.value)} />
        </label>

        <label style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          <span style={{ fontSize: 13 }}>Password baru</span>
          <input className="input" type="password" value={baru} autoComplete="new-password"
            onChange={(e) => setBaru(e.target.value)} />
          {baru !== "" && !cukupPanjang && (
            <span style={{ fontSize: 12, color: "var(--danger)" }}>Minimal 6 karakter.</span>
          )}
        </label>

        <label style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          <span style={{ fontSize: 13 }}>Ulangi password baru</span>
          <input className="input" type="password" value={ulang} autoComplete="new-password"
            onChange={(e) => setUlang(e.target.value)} />
          {ulang !== "" && !cocok && (
            <span style={{ fontSize: 12, color: "var(--danger)" }}>Belum sama dengan password baru.</span>
          )}
        </label>

        <button className="btn" type="submit" disabled={!bisaSimpan}>
          {sibuk ? "Menyimpan…" : "Simpan password baru"}
        </button>
      </form>

      {msg && <div className="card" style={{ padding: 12 }}>{msg}</div>}
    </div>
  );
}
