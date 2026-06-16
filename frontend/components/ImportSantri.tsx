"use client";

import { useRef, useState } from "react";
import { upload } from "@/lib/api";

export default function ImportSantri({ onDone }: { onDone: () => void }) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<any>(null);

  async function handleFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setBusy(true);
    setResult(null);
    try {
      const r = await upload("/santri/import", file);
      setResult(r);
      onDone();
    } catch (err: any) {
      setResult({ error: err.message });
    } finally {
      setBusy(false);
      if (inputRef.current) inputRef.current.value = "";
    }
  }

  return (
    <>
      <input ref={inputRef} type="file" accept=".xlsx,.xls" style={{ display: "none" }} onChange={handleFile} />
      <button className="btn secondary" onClick={() => inputRef.current?.click()} disabled={busy}>
        {busy ? "Mengimpor..." : "⬆ Import Excel"}
      </button>

      {result && (
        <div className="card" style={{
          position: "fixed", right: 24, bottom: 24, maxWidth: 360, zIndex: 50,
          boxShadow: "0 8px 24px rgba(0,0,0,.15)",
        }}>
          <div className="row" style={{ justifyContent: "space-between" }}>
            <strong>Hasil Import</strong>
            <button className="btn secondary" style={{ padding: "2px 8px" }} onClick={() => setResult(null)}>×</button>
          </div>
          {result.error ? (
            <p style={{ color: "var(--danger)", margin: "8px 0 0" }}>{result.error}</p>
          ) : (
            <>
              <p style={{ margin: "8px 0 4px" }}>
                Tersimpan: <strong>{result.tersimpan}</strong> · Gagal: <strong>{result.gagal}</strong>
              </p>
              {result.errors?.length > 0 && (
                <ul style={{ margin: "4px 0 0", paddingLeft: 18, fontSize: 13, maxHeight: 160, overflow: "auto" }}>
                  {result.errors.map((er: any, i: number) => (
                    <li key={i}>Baris {er.baris}: {er.pesan}</li>
                  ))}
                </ul>
              )}
            </>
          )}
        </div>
      )}
    </>
  );
}
