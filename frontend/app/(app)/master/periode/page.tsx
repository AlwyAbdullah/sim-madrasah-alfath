"use client";

import MasterCrud from "@/components/MasterCrud";

// Tahun ajaran Masehi berurutan.
const NOW = new Date().getFullYear();
const TAHUN_OPTS = Array.from({ length: 12 }, (_, i) => {
  const a = NOW - 1 + i;
  return { value: `${a}/${a + 1}`, label: `${a}/${a + 1}` };
});

export default function PeriodeMaster() {
  return (
    <MasterCrud
      title="Periode"
      basePath="/periode"
      columns={[
        { key: "nama", label: "Nama" },
        { key: "tahun_ajaran", label: "Tahun Ajaran (M)" },
        { key: "semester", label: "Semester" },
        { key: "is_active", label: "Aktif", render: (r) => (r.is_active ? "Ya" : "-") },
      ]}
      fields={[
        { key: "nama", label: "Nama Periode", required: true, placeholder: "mis. 2025/2026 Ganjil" },
        {
          key: "tahun_ajaran", label: "Tahun Ajaran (Masehi)", type: "select", required: true,
          options: TAHUN_OPTS,
        },
        {
          key: "semester", label: "Semester", type: "select", required: true,
          options: [{ value: "ganjil", label: "Ganjil" }, { value: "genap", label: "Genap" }],
        },
        { key: "is_active", label: "Status Aktif", type: "boolean" },
      ]}
    />
  );
}
