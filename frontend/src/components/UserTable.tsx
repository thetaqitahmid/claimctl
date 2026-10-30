import { useMemo, useState } from "react";
import { Menu, MenuButton, MenuItems, MenuItem } from "@headlessui/react";
import { User as UserProps } from "../store/api/auth";

interface UserTableProps {
  data: UserProps[];
  onEdit: (user: UserProps) => void;
  onToggleStatus: (user: UserProps) => void;
  onDelete: (user: UserProps) => void;
}

const UserTable = ({
  data,
  onEdit,
  onToggleStatus,
  onDelete,
}: UserTableProps) => {
  const [search, setSearch] = useState("");
  const [role, setRole] = useState("");
  const [status, setStatus] = useState("");
  const [page, setPage] = useState(0);
  const [ascending, setAscending] = useState(true);
  const filtered = useMemo(
    () =>
      data
        .filter(
          (user) =>
            `${user.name} ${user.email}`
              .toLowerCase()
              .includes(search.toLowerCase()) &&
            (!role || user.role === role) &&
            (!status || user.status === status),
        )
        .sort((a, b) =>
          ascending
            ? a.name.localeCompare(b.name)
            : b.name.localeCompare(a.name),
        ),
    [data, search, role, status, ascending],
  );
  const pageCount = Math.max(1, Math.ceil(filtered.length / 10));
  const currentPage = Math.min(page, pageCount - 1);
  const users = filtered.slice(currentPage * 10, (currentPage + 1) * 10);

  return (
    <div className="space-y-4">
      <div className="admin-panel p-4 flex flex-wrap items-end gap-4">
        <label className="flex-1 min-w-48 text-sm text-slate-300">
          Search users
          <input
            type="search"
            placeholder="Name or email"
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setPage(0);
            }}
            className="admin-input mt-2"
          />
        </label>
        <label className="text-sm text-slate-300">
          Role
          <select
            className="admin-input mt-2"
            value={role}
            onChange={(e) => {
              setRole(e.target.value);
              setPage(0);
            }}
          >
            <option value="">All roles</option>
            <option value="admin">Admin</option>
            <option value="user">User</option>
          </select>
        </label>
        <label className="text-sm text-slate-300">
          Status
          <select
            className="admin-input mt-2"
            value={status}
            onChange={(e) => {
              setStatus(e.target.value);
              setPage(0);
            }}
          >
            <option value="">All statuses</option>
            <option value="active">Active</option>
            <option value="inactive">Inactive</option>
          </select>
        </label>
        {(search || role || status) && (
          <button
            className="btn-ghost"
            onClick={() => {
              setSearch("");
              setRole("");
              setStatus("");
              setPage(0);
            }}
          >
            Clear filters
          </button>
        )}
      </div>
      <div className="admin-panel overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead className="bg-slate-800 text-slate-200">
            <tr>
              <th
                scope="col"
                className="p-4"
                aria-sort={ascending ? "ascending" : "descending"}
              >
                <button onClick={() => setAscending(!ascending)}>
                  Name {ascending ? "↑" : "↓"}
                </button>
              </th>
              <th scope="col" className="p-4">
                Role
              </th>
              <th scope="col" className="p-4">
                Status
              </th>
              <th scope="col" className="p-4">
                Actions
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-800">
            {users.length === 0 ? (
              <tr>
                <td colSpan={4} className="p-10 text-center text-slate-300">
                  No users match your filters. Try a different search or clear
                  the filters.
                </td>
              </tr>
            ) : (
              users.map((user) => (
                <tr key={user.id}>
                  <td className="p-4">
                    <span className="font-medium text-white">{user.name}</span>
                    <span className="block text-slate-400 mt-1">
                      {user.email}
                    </span>
                  </td>
                  <td className="p-4 capitalize">{user.role}</td>
                  <td className="p-4">
                    <span
                      className={`badge ${user.status === "active" ? "badge-available" : "badge-busy"}`}
                    >
                      {user.status}
                    </span>
                  </td>
                  <td className="p-4">
                    <div className="flex flex-wrap gap-2">
                      <button
                        className="btn-ghost text-sm"
                        aria-label={`Edit ${user.name}`}
                        onClick={() => onEdit(user)}
                      >
                        Edit
                      </button>
                      <Menu>
                        <MenuButton
                          className="btn-ghost"
                          aria-label={`More actions for ${user.name}`}
                        >
                          More
                        </MenuButton>
                        <MenuItems
                          anchor="bottom end"
                          className="z-50 min-w-40 p-1 border border-slate-600 rounded-lg bg-slate-900 shadow-xl [--anchor-gap:8px]"
                        >
                          <MenuItem>
                            <button
                              className="block w-full text-left px-3 py-2 data-[focus]:bg-slate-800 rounded"
                              onClick={() => onToggleStatus(user)}
                            >
                              {user.status === "active"
                                ? "Deactivate"
                                : "Activate"}
                            </button>
                          </MenuItem>
                          <MenuItem>
                            <button
                              className="block w-full text-left px-3 py-2 text-red-300 data-[focus]:bg-slate-800 rounded"
                              onClick={() => onDelete(user)}
                            >
                              Delete
                            </button>
                          </MenuItem>
                        </MenuItems>
                      </Menu>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
      <nav
        aria-label="User pagination"
        className="flex flex-wrap items-center gap-3 text-sm text-slate-300"
      >
        <p role="status" className="mr-auto">
          {filtered.length} users · Page {currentPage + 1} of {pageCount}
        </p>
        <button
          className="btn-ghost"
          disabled={currentPage === 0}
          onClick={() => setPage(currentPage - 1)}
        >
          Previous
        </button>
        <button
          className="btn-ghost"
          disabled={currentPage >= pageCount - 1}
          onClick={() => setPage(currentPage + 1)}
        >
          Next
        </button>
      </nav>
    </div>
  );
};

export default UserTable;
