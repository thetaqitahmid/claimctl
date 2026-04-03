import { useGetResourceHistoryQuery } from "../store/api/reservations";
import { format } from "date-fns";
import {
    createColumnHelper,
    flexRender,
    getCoreRowModel,
    useReactTable,
    getSortedRowModel,
    SortingState,
} from "@tanstack/react-table";
import { useMemo, useState } from "react";
import { ArrowUpDown } from "lucide-react";

interface ResourceHistoryProps {
    resourceId: string;
}

interface HistoryItem {
    id: string;
    resourceId: string;
    resourceName: string;
    reservationId: string | null;
    action: string;
    timestamp: number;
    details: string;
    userName: string;
}

const columnHelper = createColumnHelper<HistoryItem>();

const ResourceHistory = ({ resourceId }: ResourceHistoryProps) => {
    const { data: history = [], isLoading, error } = useGetResourceHistoryQuery(resourceId);
    const [sorting, setSorting] = useState<SortingState>([
        { id: "timestamp", desc: true },
    ]);

    const columns = useMemo(
        () => [
            columnHelper.accessor("userName", {
                header: "User",
                cell: (info) => <span className="font-medium text-slate-300">{info.getValue()}</span>,
            }),
            columnHelper.accessor("action", {
                header: "Action",
                cell: (info) => {
                    const action = info.getValue();
                    let colorClass = "text-slate-400";
                    if (action === "created") colorClass = "text-cyan-400";
                    if (action === "activated") colorClass = "text-green-400";
                    if (action === "cancelled") colorClass = "text-red-400";
                    if (action === "completed") colorClass = "text-slate-400";
                    return (
                        <span className={`px-2 py-1 rounded-full text-xs font-medium bg-slate-800/50 border border-slate-700 ${colorClass}`}>
                            {action.toUpperCase()}
                        </span>
                    );
                },
            }),
            columnHelper.accessor("timestamp", {
                header: ({ column }) => {
                    return (
                        <button
                            className="flex items-center gap-1 hover:text-white"
                            onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
                        >
                            Date & Time
                            <ArrowUpDown className="w-3 h-3" />
                        </button>
                    );
                },
                cell: (info) => {
                    const date = new Date(info.getValue() * 1000); // Timestamp is unix seconds from Go backend
                    return <span className="text-slate-400 text-xs">{format(date, "MMM d, yyyy HH:mm")}</span>;
                },
            }),
            columnHelper.accessor("details", {
                header: "Details",
                cell: (info) => {
                    return <span className="text-slate-500 text-xs truncate max-w-[200px] block" title={JSON.stringify(info.getValue())}>{JSON.stringify(info.getValue())}</span>;
                },
            }),
        ],
        []
    );

    const table = useReactTable({
        data: history,
        columns,
        state: {
            sorting,
        },
        onSortingChange: setSorting,
        getCoreRowModel: getCoreRowModel(),
        getSortedRowModel: getSortedRowModel(),
    });

    if (isLoading) {
        return <div className="p-8 text-center text-slate-500">Loading history...</div>;
    }

    if (error) {
       // Log error for debugging if needed, but display user friendly message
       console.error("Failed to load history:", error);
       return (
            <div className="p-4 bg-red-900/10 border border-red-900/20 rounded text-red-500 text-sm">
                Failed to load history. You might not have permission.
            </div>
       )
    }

    if (history.length === 0) {
        return <div className="p-8 text-center text-slate-500">No history found for this resource.</div>;
    }

    return (
        <div className="w-full rounded-lg">
            <table className="w-full text-left text-sm block md:table">
                <thead className="block md:table-header-group">
                    {table.getHeaderGroups().map((headerGroup) => (
                        <tr key={headerGroup.id} className="border-b border-slate-800/50 bg-slate-900/50 flex flex-col md:table-row p-4 md:p-0 gap-3 md:gap-0">
                            {headerGroup.headers.map((header) => (
                                <th key={header.id} className="px-2 md:px-4 py-1 md:py-3 font-semibold text-slate-400 text-xs uppercase tracking-wider block md:table-cell">
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
                    {table.getRowModel().rows.map((row) => (
                        <tr key={row.id} className="hover:bg-white/[0.02] border border-slate-800/50 md:border-b md:border-slate-800/50 block md:table-row mb-4 md:mb-0 rounded-lg md:rounded-none bg-slate-900/20 md:bg-transparent overflow-hidden">
                            {row.getVisibleCells().map((cell) => {
                                const id = cell.column.id;
                                const label = id === 'userName' ? 'User' :
                                              id === 'action' ? 'Action' :
                                              id === 'timestamp' ? 'Date & Time' :
                                              id === 'details' ? 'Details' : '';
                                return (
                                <td key={cell.id} className="px-4 py-3 block md:table-cell border-b border-slate-800/30 md:border-0 last:border-0 align-middle">
                                    <div className="md:hidden text-[10px] text-slate-500 font-semibold mb-1.5 uppercase tracking-wider">{label}</div>
                                    <div className="w-full overflow-x-auto md:overflow-visible text-slate-300">
                                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                                    </div>
                                </td>
                            )})}
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
};

export default ResourceHistory;
