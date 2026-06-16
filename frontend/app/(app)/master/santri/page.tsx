"use client";

import MasterCrud from "@/components/MasterCrud";
import ImportSantri from "@/components/ImportSantri";
import { api } from "@/lib/api";

export default function SantriMaster() {
  return (
    <MasterCrud
      title="Santri"
      basePath="/santri"
      headerExtra={(reload) => <ImportSantri onDone={reload} />}
      columns={[
        { key: "nis", label: "NIS" },
        { key: "nama", label: "Nama" },
        { key: "jenis_kelamin", label: "L/P" },
        { key: "kelas_nama", label: "Kelas" },
      ]}
      fields={[
        { key: "nis", label: "NIS", placeholder: "opsional" },
        { key: "nama", label: "Nama", required: true },
        {
          key: "jenis_kelamin", label: "Jenis Kelamin", type: "select", required: true,
          options: [{ value: "L", label: "Laki-laki" }, { value: "P", label: "Perempuan" }],
        },
        {
          key: "kelas_id", label: "Kelas", type: "select", required: true, numeric: true,
          options: () => api("/kelas").then((ks: any[]) => ks.map((k) => ({ value: k.id, label: k.nama }))),
        },
      ]}
    />
  );
}
