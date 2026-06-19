"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

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
        <input className="input" type="password" style={{ width: "100%", margin: "6px 0 20px" }}
          value={password} onChange={(e) => setPassword(e.target.value)} required />

        <button className="btn" style={{ width: "100%" }} disabled={loading}>
          {loading ? "Memproses..." : "Masuk"}
        </button>
      </form>
    </div>
  );
}
