"use client";

import MasterCrud from "@/components/MasterCrud";

export default function LiburMaster() {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
      <p className="muted" style={{ margin: 0, fontSize: 13 }}>
        Hari sekolah: <strong>Sabtu–Rabu</strong> (Kamis & Jumat libur tetap). Tanggal di sini <strong>dikecualikan</strong> dari perhitungan persentase kehadiran.
      </p>
      <MasterCrud
        title="Hari Libur"
        basePath="/hari-libur"
        columns={[
          { key: "tanggal", label: "Tanggal" },
          { key: "keterangan", label: "Keterangan" },
        ]}
        fields={[
          { key: "tanggal", label: "Tanggal", type: "date", required: true },
          { key: "keterangan", label: "Keterangan", placeholder: "mis. Libur Maulid Nabi" },
        ]}
      />
    </div>
  );
}
