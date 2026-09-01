"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";

// Ikon digambar langsung di sini (bukan emoji 👁) supaya ukurannya sama di
// semua HP — emoji dirender tiap ponsel dengan gaya dan lebarnya sendiri.
function IkonMata() {
  return (
    <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M1 12s4-7 11-7 11 7 11 7-4 7-11 7-11-7-11-7z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}

function IkonMataTertutup() {
  return (
    <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94" />
      <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19" />
      <path d="M14.12 14.12a3 3 0 1 1-4.24-4.24" />
      <line x1="1" y1="1" x2="23" y2="23" />
    </svg>
  );
}

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  // Password bawaan dibagikan lisan ke guru; tanpa tombol ini, salah ketik satu
  // huruf tidak kelihatan dan berakhir jadi "akun saya tidak bisa dibuka".
  const [lihatPass, setLihatPass] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await api("/auth/login", { method: "POST", body: { username, password } });
      router.push("/dashboard");
    } catch (err: any) {
      setError(err.message || "Login gagal");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={{
      minHeight: "100vh", display: "grid", placeItems: "center", padding: 20,
      background: "linear-gradient(135deg, #0b1220 0%, #0e1a2b 45%, #14532d 120%)",
    }}>
      <form onSubmit={submit} className="card" style={{ width: 380, padding: 28, boxShadow: "var(--shadow-lg)" }}>
        <div style={{ textAlign: "center", marginBottom: 22 }}>
          <div style={{
            width: 56, height: 56, borderRadius: 16, margin: "0 auto 12px", display: "grid", placeItems: "center",
            background: "linear-gradient(135deg, var(--primary-2), var(--primary))",
            boxShadow: "0 8px 20px rgba(22,163,74,.4)", fontSize: 28,
          }}>🕌</div>
          <div style={{ fontSize: 22, fontWeight: 800, letterSpacing: "-.02em" }}>SIM-Madrasah</div>
          <div className="muted" style={{ fontSize: 13 }}>Madrasah Al Fath · Masuk untuk melanjutkan</div>
        </div>

        {error && (
          <div style={{ background: "#fee2e2", color: "#991b1b", padding: 10, borderRadius: 10, marginBottom: 14, fontSize: 13 }}>
            {error}
          </div>
        )}

        <label style={{ fontSize: 13, fontWeight: 600 }}>Username</label>
        <input className="input" style={{ width: "100%", margin: "6px 0 14px" }}
          value={username} onChange={(e) => setUsername(e.target.value)} autoFocus required />

        <label style={{ fontSize: 13, fontWeight: 600 }}>Password</label>
        <div style={{ position: "relative", margin: "6px 0 20px" }}>
          <input className="input" type={lihatPass ? "text" : "password"}
            style={{ width: "100%", paddingRight: 44 }}
            value={password} onChange={(e) => setPassword(e.target.value)} required />
          <button type="button" onClick={() => setLihatPass((v) => !v)}
            aria-label={lihatPass ? "Sembunyikan password" : "Lihat password"}
            title={lihatPass ? "Sembunyikan password" : "Lihat password"}
            style={{
              position: "absolute", top: 0, right: 0, height: "100%", width: 42,
              display: "grid", placeItems: "center", cursor: "pointer",
              background: "none", border: 0, padding: 0, color: "var(--muted)",
            }}>
            {lihatPass ? <IkonMataTertutup /> : <IkonMata />}
          </button>
        </div>

        <button className="btn" style={{ width: "100%" }} disabled={loading}>
          {loading ? "Memproses..." : "Masuk"}
        </button>
      </form>
    </div>
  );
}
