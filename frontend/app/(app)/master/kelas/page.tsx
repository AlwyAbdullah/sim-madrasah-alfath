"use client";

import MasterCrud from "@/components/MasterCrud";

export default function KelasMaster() {
  return (
    <MasterCrud
      title="Kelas"
      basePath="/kelas"
      columns={[
        { key: "nama", label: "Nama Kelas" },
        { key: "tingkat", label: "Tingkat" },
        { key: "aktif", label: "Status", render: (r) => (r.aktif ? "Aktif" : "Non-aktif") },
      ]}
      fields={[
        { key: "nama", label: "Nama Kelas", required: true, placeholder: "mis. 4A / Alumni" },
        { key: "tingkat", label: "Tingkat", placeholder: "mis. 4" },
        { key: "aktif", label: "Status", type: "boolean", hideOnCreate: true },
      ]}
    />
  );
}
