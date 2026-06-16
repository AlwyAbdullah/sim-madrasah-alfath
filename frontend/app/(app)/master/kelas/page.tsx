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
      ]}
      fields={[
        { key: "nama", label: "Nama Kelas", required: true, placeholder: "mis. 4A" },
        { key: "tingkat", label: "Tingkat", placeholder: "mis. 4" },
      ]}
    />
  );
}
