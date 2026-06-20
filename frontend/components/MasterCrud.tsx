"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";

export type Option = { value: string | number; label: string };
export type Field = {
  key: string;
  label: string;
  type?: "text" | "number" | "select" | "password" | "boolean" | "date";
  options?: Option[] | (() => Promise<Option[]>);
  required?: boolean;
  numeric?: boolean;
  hideOnCreate?: boolean;
  hideOnEdit?: boolean;
  disabledOnEdit?: boolean;
  optionalOnEdit?: boolean;
  placeholder?: string;
};
export type Column = { key: string; label: string; render?: (row: any) => React.ReactNode };

type Props = {
  title: string;
  basePath: string; // mis. "/kelas"
  listPath?: string;
  columns: Column[];
  fields: Field[];
  headerExtra?: (reload: () => void) => React.ReactNode;
};

export default function MasterCrud({ title, basePath, listPath, columns, fields, headerExtra }: Props) {
  const [rows, setRows] = useState<any[]>([]);
  const [opts, setOpts] = useState<Record<string, Option[]>>({});
  const [editing, setEditing] = useState<any | null>(null); // null = tutup, {} = tambah, {id} = edit
  const [form, setForm] = useState<Record<string, string>>({});
  const [msg, setMsg] = useState("");
  const [saving, setSaving] = useState(false);
  const [q, setQ] = useState("");

  async function load() {
    try {
      const data = await api(listPath || basePath);
      setRows(data);
    } catch (e: any) { setMsg(e.message); }
  }

  useEffect(() => {
    load();
    // muat opsi select dinamis
    fields.forEach((f) => {
      if (typeof f.options === "function") {
        (f.options as () => Promise<Option[]>)().then((o) =>
          setOpts((prev) => ({ ...prev, [f.key]: o }))
        );
      } else if (Array.isArray(f.options)) {
        setOpts((prev) => ({ ...prev, [f.key]: f.options as Option[] }));
      }
    });
    // eslint-disable-next-line
  }, []);

  const isEdit = editing && editing.id != null;

  function openCreate() {
    const init: Record<string, string> = {};
    fields.forEach((f) => (init[f.key] = ""));
    setForm(init);
    setEditing({});
    setMsg("");
  }
  function openEdit(row: any) {
    const init: Record<string, string> = {};
    fields.forEach((f) => (init[f.key] = row[f.key] != null ? String(row[f.key]) : ""));
    setForm(init);
    setEditing(row);
    setMsg("");
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setMsg("");
    try {
      const body: Record<string, any> = {};
      for (const f of fields) {
        if (isEdit && f.hideOnEdit) continue;
        if (!isEdit && f.hideOnCreate) continue;
        let v: any = form[f.key] ?? "";
        if (isEdit && f.optionalOnEdit && v === "") continue; // jangan kirim password kosong saat edit
        if (f.type === "boolean") v = v === "true";
        else if (f.type === "number" || f.numeric) v = v === "" ? null : Number(v);
        body[f.key] = v;
      }
      if (isEdit) {
        await api(`${basePath}/${editing.id}`, { method: "PUT", body });
      } else {
        await api(basePath, { method: "POST", body });
      }
      setEditing(null);
      await load();
      setMsg("Tersimpan.");
    } catch (e: any) {
      setMsg(e.message);
    } finally {
      setSaving(false);
    }
  }

  async function del(row: any) {
    if (!confirm(`Hapus / nonaktifkan "${row.nama || row.username}"?`)) return;
    try {
      await api(`${basePath}/${row.id}`, { method: "DELETE" });
      await load();
    } catch (e: any) {
      setMsg(e.message);
    }
  }

  const ql = q.toLowerCase().trim();
  const filtered = ql
    ? rows.filter((r) => Object.values(r).some((v) => v != null && String(v).toLowerCase().includes(ql)))
    : rows;

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
      <div className="row" style={{ justifyContent: "space-between" }}>
        <h1 style={{ margin: 0 }}>{title}</h1>
        <div className="row">
          {headerExtra && headerExtra(load)}
          <button className="btn" onClick={openCreate}>+ Tambah</button>
        </div>
      </div>

      <div className="row" style={{ position: "relative", maxWidth: 360 }}>
        <input className="input" style={{ width: "100%", paddingRight: 28 }}
          placeholder={`Cari ${title.toLowerCase()}…`}
          value={q} onChange={(e) => setQ(e.target.value)} />
        {q && (
          <button onClick={() => setQ("")} title="Hapus"
            style={{ position: "absolute", right: 8, top: 8, border: "none", background: "transparent", cursor: "pointer", color: "var(--muted)", fontSize: 16 }}>×</button>
        )}
      </div>

      {msg && <div className="card" style={{ padding: 12 }}>{msg}</div>}

      {editing && (
        <form className="card" onSubmit={submit}>
          <h3 style={{ marginTop: 0 }}>{isEdit ? "Edit" : "Tambah"} {title}</h3>
          <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))", gap: 12 }}>
            {fields.map((f) => {
              if (isEdit && f.hideOnEdit) return null;
              if (!isEdit && f.hideOnCreate) return null;
              const disabled = isEdit && f.disabledOnEdit;
              const required = f.required && !(isEdit && f.optionalOnEdit);
              return (
                <div key={f.key} style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                  <label style={{ fontSize: 13, fontWeight: 600 }}>
                    {f.label}{required ? " *" : ""}
                  </label>
                  {f.type === "select" || f.type === "boolean" ? (
                    <select className="input" value={form[f.key] ?? ""} disabled={disabled}
                      required={required}
                      onChange={(e) => setForm({ ...form, [f.key]: e.target.value })}>
                      <option value="">— pilih —</option>
                      {(f.type === "boolean"
                        ? [{ value: "true", label: "Aktif" }, { value: "false", label: "Nonaktif" }]
                        : (opts[f.key] || [])
                      ).map((o) => (
                        <option key={String(o.value)} value={String(o.value)}>{o.label}</option>
                      ))}
                    </select>
                  ) : (
                    <input className="input" type={f.type === "password" ? "password" : f.type === "number" ? "number" : f.type === "date" ? "date" : "text"}
                      value={form[f.key] ?? ""} disabled={disabled} required={required}
                      placeholder={f.placeholder || (isEdit && f.optionalOnEdit ? "(kosongkan jika tidak diubah)" : "")}
                      onChange={(e) => setForm({ ...form, [f.key]: e.target.value })} />
                  )}
                </div>
              );
            })}
          </div>
          <div className="row" style={{ marginTop: 14 }}>
            <button className="btn" disabled={saving}>{saving ? "Menyimpan..." : "Simpan"}</button>
            <button type="button" className="btn secondary" onClick={() => setEditing(null)}>Batal</button>
          </div>
        </form>
      )}

      <div className="card" style={{ padding: 0, overflow: "auto" }}>
        <table>
          <thead>
            <tr>
              {columns.map((c) => <th key={c.key}>{c.label}</th>)}
              <th style={{ width: 150 }}>Aksi</th>
            </tr>
          </thead>
          <tbody>
            {filtered.length === 0 && (
              <tr><td colSpan={columns.length + 1} className="muted" style={{ padding: 16 }}>
                {rows.length === 0 ? "Belum ada data." : `Tidak ada hasil untuk "${q}".`}
              </td></tr>
            )}
            {filtered.map((row) => (
              <tr key={row.id}>
                {columns.map((c) => (
                  <td key={c.key}>{c.render ? c.render(row) : (row[c.key] ?? "-")}</td>
                ))}
                <td>
                  <div className="row" style={{ gap: 6 }}>
                    <button className="btn secondary" style={{ padding: "5px 10px" }} onClick={() => openEdit(row)}>Edit</button>
                    <button className="btn secondary" style={{ padding: "5px 10px", color: "var(--danger)" }} onClick={() => del(row)}>Hapus</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
