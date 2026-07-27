"use client";

import MasterCrud from "@/components/MasterCrud";

export default function GuruMaster() {
  return (
    <MasterCrud
      title="Guru"
      basePath="/guru"
      columns={[
        { key: "nama", label: "Nama" },
        { key: "no_rekening", label: "No. Rekening" },
        { key: "nama_bank", label: "Bank" },
        { key: "mengajar_per_pekan", label: "Mengajar / Pekan" },
        { key: "no_telepon", label: "No. Telepon" },
      ]}
      fields={[
        { key: "nama", label: "Nama", required: true },
        { key: "no_rekening", label: "No. Rekening", placeholder: "mis. 1234567890" },
        { key: "nama_bank", label: "Nama Bank", placeholder: "mis. BSI, BCA, Mandiri" },
        { key: "mengajar_per_pekan", label: "Jumlah Mengajar / Pekan", type: "number", numeric: true, placeholder: "mis. 12" },
        { key: "no_telepon", label: "No. Telepon", placeholder: "mis. 0812xxxxxxx" },
      ]}
    />
  );
}
