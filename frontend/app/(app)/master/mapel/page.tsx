"use client";

import MasterCrud from "@/components/MasterCrud";

export default function MapelMaster() {
  return (
    <MasterCrud
      title="Mata Pelajaran"
      basePath="/mata-pelajaran"
      columns={[
        { key: "kode", label: "Kode" },
        { key: "nama", label: "Nama" },
      ]}
      fields={[
        { key: "kode", label: "Kode", placeholder: "opsional, mis. FQH" },
        { key: "nama", label: "Nama", required: true },
      ]}
    />
  );
}
