"use client";

import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import Link from "next/link";
import { api } from "@/lib/api";

const NAV = [
  { href: "/dashboard", label: "Dashboard", icon: "📊" },
  { href: "/absensi", label: "Absensi", icon: "🗓️" },
  { href: "/nilai", label: "Nilai", icon: "📝" },
  { href: "/leger", label: "Leger & Peringkat", icon: "🏆" },
  { href: "/spp", label: "SPP", icon: "💳" },
  { href: "/rapor", label: "Rapor", icon: "📄" },
];

const MASTER_NAV = [
  { href: "/master/santri", label: "Santri", icon: "🧑‍🎓" },
  { href: "/master/kelas", label: "Kelas", icon: "🏫" },
  { href: "/master/mapel", label: "Mata Pelajaran", icon: "📚" },
  { href: "/master/periode", label: "Periode", icon: "📆" },
  { href: "/master/users", label: "User", icon: "👤" },
];

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [user, setUser] = useState<{ nama: string; role: string } | null>(null);
  const [ready, setReady] = useState(false);
  const [navOpen, setNavOpen] = useState(false);

  useEffect(() => {
    api("/auth/me")
      .then((d) => setUser(d.user))
      .catch(() => router.push("/login"))
      .finally(() => setReady(true));
  }, [router]);

  useEffect(() => { setNavOpen(false); }, [pathname]);

  async function logout() {
    try { await api("/auth/logout", { method: "POST" }); } catch {}
    router.push("/login");
  }

  if (!ready) return <div style={{ padding: 40 }} className="muted">Memuat...</div>;

  const initial = (user?.nama || "?").trim().charAt(0).toUpperCase();

  return (
    <div className="layout">
      <aside className={`sidebar ${navOpen ? "open" : ""}`}>
        <div className="brand">
          <span className="brand-logo">🕌</span>
          <span>SIM-Madrasah</span>
        </div>
        <nav style={{ display: "flex", flexDirection: "column", gap: 3 }}>
          {NAV.map((n) => <NavLink key={n.href} {...n} pathname={pathname} />)}

          {user?.role === "admin" && (
            <>
              <div className="nav-section">Master Data</div>
              {MASTER_NAV.map((n) => <NavLink key={n.href} {...n} pathname={pathname} />)}
            </>
          )}
        </nav>
      </aside>

      <div className={`sidebar-overlay ${navOpen ? "show" : ""}`} onClick={() => setNavOpen(false)} />

      <div className="app-main">
        <header className="app-header">
          <button className="hamburger" aria-label="Menu" onClick={() => setNavOpen((v) => !v)}>☰</button>
          <div style={{ flex: 1 }} />
          <div className="user-chip">
            <span className="avatar">{initial}</span>
            <span style={{ fontSize: 13 }}>
              {user?.nama}<span className="muted" style={{ fontSize: 11 }}> · {user?.role}</span>
            </span>
          </div>
          <button className="btn secondary" onClick={logout}>Logout</button>
        </header>
        <main style={{ padding: 24 }}>{children}</main>
      </div>
    </div>
  );
}

function NavLink({ href, label, icon, pathname }: { href: string; label: string; icon?: string; pathname: string }) {
  const active = pathname === href || pathname.startsWith(href + "/");
  return (
    <Link href={href} className={`nav-link ${active ? "active" : ""}`}>
      <span className="nav-ico">{icon}</span>
      {label}
    </Link>
  );
}
