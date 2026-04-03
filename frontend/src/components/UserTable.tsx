import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  useReactTable,
  getSortedRowModel,
  getFilteredRowModel,
  SortingState,
  ColumnFiltersState,
} from "@tanstack/react-table";
import { useMemo, useState } from "react";
import { User as UserProps } from "../store/api/auth";
import {
  User,
  Shield,
  Edit,
  Trash2,
  UserCheck,
  UserX,
  ArrowUpDown,
  Mail,
} from "lucide-react";

interface UserTableProps {
  data: UserProps[];
  onEdit: (user: UserProps) => void;
  onToggleStatus: (user: UserProps) => void;
  onDelete: (user: UserProps) => void;
}

const columnHelper = createColumnHelper<UserProps>();

const UserTable = ({
  data,
  onEdit,
  onToggleStatus,
  onDelete,
}: UserTableProps) => {
  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);

  const columns = useMemo(
    () => [
      columnHelper.accessor("name", {
        header: ({ column }) => {
          return (
            <div className="flex flex-col gap-2 p-1">
              <button
                className="flex items-center gap-1 hover:text-white self-start"
                onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
              >
                Name
                <ArrowUpDown className="w-4 h-4" />
              </button>
              <input
                 placeholder="Filter names..."
                 value={(column.getFilterValue() as string) ?? ""}
                 onChange={(event) => column.setFilterValue(event.target.value)}
                 onClick={(e) => e.stopPropagation()}
                 className="w-full rounded bg-slate-800/50 px-2 py-1 text-xs text-slate-200 border border-slate-700 focus:outline-none focus:border-cyan-500 font-normal shadow-inner"
              />
            </div>
          );
        },
        cell: (info) => (
          <div className="flex flex-col">
            <span className="font-semibold text-white">
              {info.getValue()}
            </span>
            <div className="flex items-center text-xs text-slate-400 gap-1 mt-0.5">
               <Mail className="w-3 h-3" />
               {info.row.original.email}
            </div>
          </div>
        ),
      }),
      columnHelper.accessor("role", {
        header: ({ column }) => {
          return (
            <div className="flex flex-col gap-2 p-1">
              <button
                className="flex items-center gap-1 hover:text-white self-start"
                onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
              >
                Role
                <ArrowUpDown className="w-4 h-4" />
              </button>
               <select
                value={(column.getFilterValue() as string) ?? ""}
                onChange={(event) => column.setFilterValue(event.target.value)}
                onClick={(e) => e.stopPropagation()}
                className="rounded bg-slate-800/50 px-2 py-1 text-xs text-slate-200 border border-slate-700 focus:outline-none focus:border-cyan-500 w-full font-normal shadow-inner"
              >
                <option value="">All</option>
                <option value="admin">Admin</option>
                <option value="user">User</option>
              </select>
            </div>
          );
        },
        cell: (info) => (
          <div className="flex items-center text-slate-300 gap-2">
            {info.getValue() === "admin" ? (
              <Shield className="w-4 h-4 text-amber-400" />
            ) : (
              <User className="w-4 h-4 text-cyan-400" />
            )}
            <span className="capitalize">{info.getValue()}</span>
          </div>
        ),
      }),
      columnHelper.accessor("status", {
        header: ({ column }) => {
          return (
            <button
              className="flex items-center gap-1 hover:text-white self-start"
              onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
            >
              Status
              <ArrowUpDown className="w-4 h-4" />
            </button>
          );
        },
        cell: (info) => {
          const status = info.getValue();
          const isActive = status === "active";
          return (
            <span className={`badge ${isActive ? "badge-available" : "badge-busy"}`}>
                {status.charAt(0).toUpperCase() + status.slice(1)}
            </span>
          );
        },
      }),
      columnHelper.display({
        id: "actions",
        header: "Actions",
        cell: ({ row }) => {
          const user = row.original;
          const isActive = user.status === "active";

          return (
            <div className="flex items-center gap-2">
              <button
                onClick={() => onToggleStatus(user)}
                title={isActive ? "Deactivate" : "Activate"}
                className={`p-2 rounded-lg border transition-all ${
                  isActive
                    ? "bg-brand-busy/10 text-brand-busy hover:bg-brand-busy/20 border-brand-busy/20"
                    : "bg-brand-available/10 text-brand-available hover:bg-brand-available/20 border-brand-available/20"
                }`}
              >
                {isActive ? <UserX className="w-4 h-4" /> : <UserCheck className="w-4 h-4" />}
              </button>
              <button
                onClick={() => onEdit(user)}
                className="p-2 rounded-lg bg-slate-800/50 text-slate-400 hover:text-white border border-slate-700/50 hover:border-slate-600 transition-all shadow-sm"
                title="Edit"
              >
                <Edit className="w-4 h-4" />
              </button>
              <button
                onClick={() => onDelete(user)}
                className="p-2 rounded-lg bg-brand-busy/10 text-brand-busy hover:bg-brand-busy/20 border border-brand-busy/20 transition-all shadow-sm"
                title="Delete"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            </div>
          );
        },
      }),
    ],
    [onEdit, onToggleStatus, onDelete]
  );

  const table = useReactTable({
    data,
    columns,
    state: {
      sorting,
      columnFilters,
    },
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
  });

  return (
    <div className="w-full space-y-6">
      <div className="w-full overflow-hidden rounded-xl glass-panel border border-slate-800/50 shadow-2xl">
        <div className="w-full">
          <table className="w-full text-left text-sm block md:table">
            <thead className="block md:table-header-group">
              {table.getHeaderGroups().map((headerGroup) => (
                <tr key={headerGroup.id} className="border-b border-slate-800/50 bg-slate-900/50 flex flex-col md:table-row p-4 md:p-0 gap-3 md:gap-0">
                  {headerGroup.headers.map((header) => (
                    <th key={header.id} className="px-2 md:px-6 py-1 md:py-4 font-semibold text-slate-300 uppercase tracking-wider text-[11px] block md:table-cell">
                      {header.isPlaceholder
                        ? null
                        : flexRender(
                            header.column.columnDef.header,
                            header.getContext()
                          )}
                    </th>
                  ))}
                </tr>
              ))}
            </thead>
            <tbody className="divide-y divide-slate-800/50 text-slate-400 block md:table-row-group p-4 md:p-0">
              {table.getRowModel().rows.length === 0 ? (
                <tr className="block md:table-row">
                  <td colSpan={columns.length} className="px-6 py-12 text-center text-slate-500 italic block md:table-cell">
                    No users found.
                  </td>
                </tr>
              ) : (
                table.getRowModel().rows.map((row) => (
                  <tr
                    key={row.id}
                    className="group hover:bg-white/[0.02] transition-colors duration-150 border border-slate-800/50 md:border-b md:border-slate-800/10 block md:table-row mb-4 md:mb-0 rounded-xl md:rounded-none bg-slate-900/20 md:bg-transparent overflow-hidden"
                  >
                    {row.getVisibleCells().map((cell) => {
                      const id = cell.column.id;
                      const label = id === 'name' ? 'Name' :
                                    id === 'role' ? 'Role' :
                                    id === 'status' ? 'Status' :
                                    id === 'actions' ? 'Actions' : '';
                      return (
                      <td key={cell.id} className="px-4 md:px-6 py-3 md:py-4 md:whitespace-nowrap align-middle block md:table-cell border-b border-slate-800/30 md:border-0 last:border-0">
                        <div className="md:hidden text-[10px] text-slate-500 font-semibold mb-1.5 uppercase tracking-wider">{label}</div>
                        <div className="w-full overflow-x-auto md:overflow-visible">
                          {flexRender(cell.column.columnDef.cell, cell.getContext())}
                        </div>
                      </td>
                    )})}
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

export default UserTable;
