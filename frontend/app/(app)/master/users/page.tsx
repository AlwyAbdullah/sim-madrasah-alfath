"use client";

import MasterCrud from "@/components/MasterCrud";

export default function UsersMaster() {
  return (
    <MasterCrud
      title="User"
      basePath="/users"
      columns={[
        { key: "username", label: "Username" },
        { key: "nama", label: "Nama" },
        { key: "role", label: "Role" },
        { key: "is_active", label: "Status", render: (r) => (r.is_active ? "Aktif" : "Nonaktif") },
      ]}
      fields={[
        { key: "username", label: "Username", required: true, disabledOnEdit: true },
        { key: "nama", label: "Nama", required: true },
        {
          key: "role", label: "Role", type: "select", required: true,
          options: [
            { value: "admin", label: "Admin" },
            { value: "guru", label: "Guru" },
            { value: "kepala", label: "Kepala" },
          ],
        },
        { key: "password", label: "Password", type: "password", required: true, optionalOnEdit: true },
        { key: "is_active", label: "Status", type: "boolean", hideOnCreate: true },
      ]}
    />
  );
}
