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
        { key: "telegram_user_id", label: "Telegram ID", render: (r) => r.telegram_user_id ?? "-" },
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
        { key: "telegram_user_id", label: "Telegram User ID", type: "number", placeholder: "mis. 123456789 (kosongkan jika tidak pakai bot)" },
        { key: "is_active", label: "Status", type: "boolean", hideOnCreate: true },
      ]}
    />
  );
}
