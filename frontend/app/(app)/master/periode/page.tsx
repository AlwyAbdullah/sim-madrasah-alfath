"use client";

import MasterCrud from "@/components/MasterCrud";

// Tahun ajaran Hijriah berurutan, mulai 1446/1447 H.
const TAHUN_OPTS = Array.from({ length: 12 }, (_, i) => {
  const a = 1446 + i;
  return { value: `${a} / ${a + 1}`, label: `${a} / ${a + 1} H` };
});

export default function PeriodeMaster() {
  return (
    <MasterCrud
      title="Periode"
      basePath="/periode"
      columns={[
        { key: "nama", label: "Nama" },
        { key: "tahun_ajaran", label: "Tahun Ajaran (H)" },
        { key: "semester", label: "Semester" },
        { key: "is_active", label: "Aktif", render: (r) => (r.is_active ? "Ya" : "-") },
      ]}
      fields={[
        { key: "nama", label: "Nama Periode", required: true, placeholder: "mis. 1446/1447 Ganjil" },
        {
          key: "tahun_ajaran", label: "Tahun Ajaran (Hijriah)", type: "select", required: true,
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
