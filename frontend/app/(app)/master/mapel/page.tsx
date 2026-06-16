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
        { key: "kitab", label: "Nama Kitab" },
      ]}
      fields={[
        { key: "kode", label: "Kode", placeholder: "opsional, mis. FQH" },
        { key: "nama", label: "Nama", required: true },
        { key: "kitab", label: "Nama Kitab", placeholder: "opsional, mis. MUQODDIMAH" },
      ]}
    />
  );
}
