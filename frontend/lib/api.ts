export const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8080/api/v1";

type Options = {
  method?: string;
  body?: unknown;
};

export async function api<T = any>(path: string, opts: Options = {}): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: opts.method || "GET",
    credentials: "include", // kirim cookie JWT
    headers: { "Content-Type": "application/json" },
    body: opts.body ? JSON.stringify(opts.body) : undefined,
  });

  if (res.status === 401) {
    if (typeof window !== "undefined" && !location.pathname.startsWith("/login")) {
      location.href = "/login";
    }
    throw new Error("Sesi berakhir");
  }

  if (!res.ok) {
    let msg = "Terjadi kesalahan";
    try {
      const j = await res.json();
      msg = j?.error?.message || msg;
    } catch {}
    throw new Error(msg);
  }
  return res.json();
}

// helper khusus download file (ekspor)
export function exportUrl(path: string): string {
  return `${API_BASE}${path}`;
}

// upload file (multipart) — mis. import santri dari Excel
export async function upload<T = any>(path: string, file: File): Promise<T> {
  const fd = new FormData();
  fd.append("file", file);
  const res = await fetch(`${API_BASE}${path}`, {
    method: "POST",
    credentials: "include",
    body: fd,
  });
  if (res.status === 401) {
    if (typeof window !== "undefined") location.href = "/login";
    throw new Error("Sesi berakhir");
  }
  if (!res.ok) {
    let msg = "Gagal mengunggah";
    try { const j = await res.json(); msg = j?.error?.message || msg; } catch {}
    throw new Error(msg);
  }
  return res.json();
}
