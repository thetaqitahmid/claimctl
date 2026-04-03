import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  useReactTable,
  getSortedRowModel,
  getFilteredRowModel,
  getExpandedRowModel,
  SortingState,
  ColumnFiltersState,
  ExpandedState,
} from "@tanstack/react-table";
import { useMemo, useState, Fragment } from "react";
import { useTranslation } from "react-i18next";
import { ResourceWithStatus, UserReservation } from "../types";
import {
  Tag,
  Clock,
  Lock,
  Trash,
  Users,
  CheckCircle,
  XCircle,
  ArrowUpDown,
  Eye,
  ChevronRight,
  ChevronDown,
  Wrench,
} from "lucide-react";
import QueueListModal from "./QueueListModal";
import HealthStatusIndicator from "./HealthStatusIndicator";
import TagFilter from "./ui/TagFilter";
import TypeFilter from "./ui/TypeFilter";
import { formatDistanceToNow } from "date-fns";

interface ResourceTableProps {
  data: ResourceWithStatus[];
  userReservations: UserReservation[];
  onReserve: (resourceId: string) => void;
  onRelease: (resourceId: string) => void;
  onView: (resourceId: string) => void;
  onDelete: (resourceId: string) => void;
  onCancelAll?: (resourceId: string) => void;
  onMaintenanceToggle?: (resourceId: string, resourceName: string, currentState: boolean) => void;
  isAdmin: boolean;
}

const columnHelper = createColumnHelper<ResourceWithStatus>();

const ResourceTable = ({
  data,
  userReservations,
  onReserve,
  onRelease,
  onView,
  onDelete,
  onCancelAll,
  onMaintenanceToggle,
  isAdmin,
}: ResourceTableProps) => {
  const { t } = useTranslation(["components", "common"]);
  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [expanded, setExpanded] = useState<ExpandedState>({});
  const [selectedQueueResource, setSelectedQueueResource] = useState<{id: string, name: string} | null>(null);

  const uniqueTypes = useMemo(() => {
    if (!data || !Array.isArray(data)) return [];
    const types = new Set(data.map((item) => item.resource.type));
    return Array.from(types);
  }, [data]);

  const uniqueTags = useMemo(() => {
    if (!data || !Array.isArray(data)) return [];
    const tags = new Set<string>();
    data.forEach((item) => {
      if (Array.isArray(item.resource.labels)) {
        item.resource.labels.forEach((label) => tags.add(label));
      }
    });
    return Array.from(tags);
  }, [data]);

  const columns = useMemo(
    () => [
      columnHelper.display({
        id: "expander",
        header: () => null,
        cell: ({ row }) => {
          return row.getCanExpand() ? (
            <button
              onClick={row.getToggleExpandedHandler()}
              className="text-slate-400 hover:text-white transition-colors p-1"
            >
              {row.getIsExpanded() ? (
                <ChevronDown className="w-4 h-4" />
              ) : (
                <ChevronRight className="w-4 h-4" />
              )}
            </button>
          ) : null;
        },
      }),
      columnHelper.accessor("resource.name", {
        header: ({ column }) => {
          return (
            <div className="flex flex-col gap-2 p-1">
                <button
                className="flex items-center gap-1 hover:text-white self-start"
                onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
              >
                {t("components:resourceTable.name")}
                <ArrowUpDown className="w-4 h-4" />
              </button>
              <input
                 placeholder={t("components:resourceTable.filterPlaceholder")}
                 value={(column.getFilterValue() as string) ?? ""}
                 onChange={(event) => column.setFilterValue(event.target.value)}
                 onClick={(e) => e.stopPropagation()}
                 className="w-full rounded bg-slate-800/50 px-2 py-1 text-xs text-slate-200 border border-slate-700 focus:outline-none focus:border-cyan-500 font-normal"
              />
            </div>
          );
        },
        cell: (info) => (
          <span className="font-semibold text-white">
            {info.getValue()}
          </span>
        ),
      }),
      columnHelper.accessor("resource.type", {
        header: ({ column }) => {
          return (
            <div className="flex flex-col gap-2 p-1">
              <button
                className="flex items-center gap-1 hover:text-white self-start"
                onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
              >
                {t("components:resourceTable.type")}
                <ArrowUpDown className="w-4 h-4" />
              </button>
              <TypeFilter
                allTypes={uniqueTypes}
                selectedTypes={(column.getFilterValue() as string[]) ?? []}
                onChange={(types) => column.setFilterValue(types.length > 0 ? types : undefined)}
              />
            </div>
          );
        },
        cell: (info) => (
          <div className="flex items-center text-slate-300">
            <Tag className="w-4 h-4 mr-2" />
            {info.getValue()}
          </div>
        ),
        filterFn: (row, columnId, filterValue: string[]) => {
          if (!filterValue || filterValue.length === 0) return true;
          const type = row.getValue(columnId) as string;
          return filterValue.includes(type);
        },
      }),
      columnHelper.accessor("resource.labels", {
        header: ({ column }) => {
          return (
            <div className="flex flex-col gap-2 p-1">
              <button
                className="flex items-center gap-1 hover:text-white self-start"
                onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
              >
                {t("components:resourceTable.labels")}
                <ArrowUpDown className="w-4 h-4" />
              </button>
              <TagFilter
                allTags={uniqueTags}
                selectedTags={(column.getFilterValue() as string[]) ?? []}
                onChange={(tags) => column.setFilterValue(tags.length > 0 ? tags : undefined)}
              />
            </div>
          );
        },
        cell: (info) => (
          <div className="flex flex-wrap gap-1.5">
            {Array.isArray(info.getValue()) && info.getValue()?.map((label, i) => (
              <span
                key={i}
                className="px-2 py-0.5 rounded text-[10px] font-medium bg-slate-800/50 text-slate-300 border border-slate-700/50"
              >
                {label}
              </span>
            ))}
          </div>
        ),
        filterFn: (row, columnId, filterValue: string[]) => {
          // Multi-tag filtering with AND logic - resource must have ALL selected tags
          if (!filterValue || filterValue.length === 0) return true;

          const labels = row.getValue(columnId) as string[];
          if (!Array.isArray(labels) || labels.length === 0) return false;

          // Check if resource has all selected tags
          return filterValue.every((selectedTag) =>
            labels.some((label) => label.toLowerCase() === selectedTag.toLowerCase())
          );
        },
      }),
      columnHelper.accessor(
        (row) => {
            const userReservation = userReservations.find(
                (res) => res.resourceId === row.resource.id &&
                (res.status === 'active' || res.status === 'pending')
            );
            if (userReservation?.status === 'active') return 'In Use (You)';
            if (userReservation?.status === 'pending') return 'Queued';
            if (row.activeReservations > 0) return 'In Use';
            return 'Available';
        },
        {
          id: "status",
          header: ({ column }) => {
            return (
                <button
                    className="flex items-center gap-1 hover:text-white self-start"
                    onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
                >
                    {t("components:resourceTable.status")}
                    <ArrowUpDown className="w-4 h-4" />
                </button>
            )
          },
          cell: (info) => {
            const status = info.getValue();
            const isUnderMaintenance = info.row.original.resource.isUnderMaintenance;

            // Show maintenance status first if enabled
            if (isUnderMaintenance) {
              return (
                <span className="badge bg-orange-500/10 text-orange-500 border border-orange-500/20">
                  {t("common:status.maintenance")}
                </span>
              );
            }

            let badgeClass = "badge-available";
            if (status.includes("In Use")) badgeClass = "badge-busy";
            if (status.includes("Queued")) badgeClass = "badge-queued";
            if (status.includes("(You)")) badgeClass = "badge-available shadow-[0_0_10px_rgba(16,185,129,0.2)]";

            return <span className={`badge ${badgeClass}`}>{status}</span>;
          },
        }
      ),
      columnHelper.display({
        id: "health_status",
        header: "Health",
        cell: (info) => {
          const row = info.row.original;
          const healthStatus = row.healthStatus;
          const healthConfig = row.healthConfig;

          // Only show health indicator if health check is configured and enabled
          if (!healthConfig || !healthConfig.enabled) {
            return (
              <span className="text-xs text-slate-500 italic">
                {t("components:resourceTable.notConfigured")}
              </span>
            );
          }

          return (
            <HealthStatusIndicator
              status={healthStatus?.status}
              responseTimeMs={healthStatus?.responseTimeMs}
              checkedAt={healthStatus?.checkedAt}
              errorMessage={healthStatus?.errorMessage}
              size="small"
            />
          );
        },
      }),
      columnHelper.display({
        id: "queue_info",
        header: t("components:resourceTable.queueActive"),
        cell: (info) => {
            const row = info.row.original;
            const userReservation = userReservations.find(
                (res) => res.resourceId === row.resource.id &&
                (res.status === 'active' || res.status === 'pending')
            );

             return (
                 <div className="text-sm flex flex-col gap-1">
                 {row.activeReservations > 0 && (
                   <div className="flex flex-col gap-0.5">
                     <button
                       onClick={() => setSelectedQueueResource({ id: row.resource.id, name: row.resource.name })}
                       className="flex items-center text-slate-400 gap-1 hover:text-white transition-colors"
                     >
                     <Users className="w-3 h-3" />
                      {row.activeReservations} {t("common:status.active")}
                     </button>
                     {row.activeReservationStartTime && (
                         <span className="text-[10px] text-emerald-400/80 pl-4">
                             Running for {formatDistanceToNow(row.activeReservationStartTime * 1000)}
                             {` (for ${row.activeReservationDuration || 'indefinite'})`}
                         </span>
                     )}
                   </div>
                 )}
                 {row.queueLength > 0 && (
                     <button
                       onClick={() => setSelectedQueueResource({ id: row.resource.id, name: row.resource.name })}
                       className="flex items-center text-amber-500 gap-1 hover:text-amber-400 transition-colors"
                     >
                     <Clock className="h-3 h-3" />
                      {row.queueLength} {t("components:resourceList.inQueue")}
                     </button>
                 )}
                 {userReservation?.status === 'pending' && userReservation.queuePosition > 0 && (
                     <div className="text-cyan-400 text-xs pl-4">
                        Pos: #{userReservation.queuePosition}
                     </div>
                 )}
                 </div>
             )
        }
      }),
      columnHelper.display({
        id: "actions",
        header: t("components:resourceTable.actions"),
        cell: ({ row }) => {
          const resource = row.original;
          const userReservation = userReservations.find(
            (res) =>
              res.resourceId === resource.resource.id &&
              (res.status === "active" || res.status === "pending")
          );
          const isUnderMaintenance = resource.resource.isUnderMaintenance;

          const handleAction = () => {
             if (userReservation) {
                 onRelease(resource.resource.id);
             } else {
                 onReserve(resource.resource.id);
             }
          };

          return (
            <div className="flex items-center gap-1 flex-wrap">
                 {userReservation?.status === 'active' ? (
                    <button
                     onClick={handleAction}
                      title={t("components:resourceList.release")}
                     className="p-1.5 rounded bg-brand-available/10 text-brand-available hover:bg-brand-available/20 border border-brand-available/20 transition-all"
                   >
                     <CheckCircle className="w-3.5 h-3.5" />
                   </button>
                 ) : userReservation?.status === 'pending' ? (
                     <button
                      onClick={handleAction}
                      title={t("common:cancel")}
                     className="p-1.5 rounded bg-brand-busy/10 text-brand-busy hover:bg-brand-busy/20 border border-brand-busy/20 transition-all"
                    >
                      <XCircle className="w-3.5 h-3.5" />
                    </button>
                ) : (
                     <button
                         onClick={handleAction}
                         disabled={isUnderMaintenance}
                         title={isUnderMaintenance ? t("common:status.maintenance") : (resource.activeReservations > 0 ? t("components:resourceList.queue") : t("components:resourceList.reserve"))}
                        className={`p-1.5 rounded transition-all border ${
                            isUnderMaintenance
                            ? "bg-slate-700/50 text-slate-500 cursor-not-allowed border-slate-600/50"
                            : resource.activeReservations > 0
                            ? "bg-brand-queued/10 text-brand-queued hover:bg-brand-queued/20 border-brand-queued/20"
                            : "bg-brand-available/10 text-brand-available hover:bg-brand-available/20 border-brand-available/20"
                        }`}
                    >
                        {resource.activeReservations > 0 ? <Clock className="w-3.5 h-3.5"/> : <Lock className="w-3.5 h-3.5"/>}
                    </button>
                )}

              <button
                onClick={() => onView(resource.resource.id)}
                className="p-1.5 rounded bg-slate-800/50 text-slate-400 hover:text-white border border-slate-700/50 hover:border-slate-600 transition-all"
                title={t("components:resourceList.view")}
              >
                <Eye className="w-3.5 h-3.5" />
              </button>

              {isAdmin && (
                <>
                  {(resource.activeReservations > 0 || resource.queueLength > 0) && onCancelAll && (
                    <button
                      onClick={() => onCancelAll(resource.resource.id)}
                      className="p-1.5 rounded bg-red-500/10 text-red-500 hover:bg-red-500/20 border border-red-500/20 transition-all"
                      title={t("components:resourceList.cancelAll")}
                    >
                      <XCircle className="w-3.5 h-3.5" />
                    </button>
                  )}
                  {onMaintenanceToggle && (
                    <button
                      onClick={() => onMaintenanceToggle(resource.resource.id, resource.resource.name, !!resource.resource.isUnderMaintenance)}
                      className={`p-1.5 rounded transition-all border ${
                        resource.resource.isUnderMaintenance
                        ? "bg-orange-500/10 text-orange-500 hover:bg-orange-500/20 border-orange-500/20"
                        : "bg-slate-700/50 text-slate-400 hover:text-orange-500 hover:bg-orange-500/10 border-slate-600/50 hover:border-orange-500/20"
                      }`}
                      title={resource.resource.isUnderMaintenance ? t("components:resourceList.maintenance.disable") : t("components:resourceList.maintenance.enable")}
                    >
                      <Wrench className="w-3.5 h-3.5" />
                    </button>
                  )}
                  <button
                    onClick={() => onDelete(resource.resource.id)}
                    className="p-1.5 rounded bg-brand-busy/10 text-brand-busy hover:bg-brand-busy/20 border border-brand-busy/20 transition-all"
                    title={t("components:resourceList.delete")}
                  >
                    <Trash className="w-3.5 h-3.5" />
                  </button>
                </>
              )}
            </div>
          );
        },
      }),
    ],
    [userReservations, uniqueTypes, onReserve, onRelease, onView, onCancelAll, onDelete, onMaintenanceToggle, isAdmin, uniqueTags, t]
  );

  const table = useReactTable({
    data,
    columns,
    state: {
      sorting,
      columnFilters,
      expanded,
    },
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onExpandedChange: setExpanded,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getExpandedRowModel: getExpandedRowModel(),
    getRowCanExpand: (row) => !!row.original.resource.properties && Object.keys(row.original.resource.properties).length > 0,
  });

  return (
    <div className="w-full space-y-6 min-w-0">
      <div className="w-full rounded-xl glass-panel">
        <table className="w-full text-left text-sm block xl:table">
            <thead className="block xl:table-header-group">
              {table.getHeaderGroups().map((headerGroup) => (
                <tr key={headerGroup.id} className="border-b border-slate-800/50 bg-slate-900/50 flex flex-col xl:table-row p-4 xl:p-0 gap-3 xl:gap-0">
                  {headerGroup.headers.map((header) => (
                    <th key={header.id} className="px-2 xl:px-4 py-1 xl:py-3 font-semibold text-slate-300 uppercase tracking-wider text-[11px] block xl:table-cell">
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
            <tbody className="divide-y divide-slate-800/50 text-slate-400 block xl:table-row-group p-4 xl:p-0">
              {table.getRowModel().rows.length === 0 ? (
                <tr className="block xl:table-row">
                  <td colSpan={columns.length} className="px-6 py-12 text-center text-slate-500 italic block xl:table-cell">
                    {t("components:resourceTable.noData")}
                  </td>
                </tr>
              ) : (
                table.getRowModel().rows.map((row) => (
                  <Fragment key={row.id}>
                  <tr
                    className="group hover:bg-white/[0.02] transition-colors duration-150 border border-slate-800/50 xl:border-b xl:border-slate-800/10 block xl:table-row mb-4 xl:mb-0 rounded-lg xl:rounded-none bg-slate-900/20 xl:bg-transparent overflow-hidden"
                  >
                    {row.getVisibleCells().map((cell) => {
                      const id = cell.column.id;
                      const label = id === 'resource_name' ? 'Name' :
                                    id === 'resource_type' ? 'Type' :
                                    id === 'resource_labels' ? 'Labels' :
                                    id === 'status' ? 'Status' :
                                    id === 'health_status' ? 'Health' :
                                    id === 'queue_info' ? 'Queue' :
                                    id === 'actions' ? 'Actions' : '';
                      return (
                      <td key={cell.id} className="px-4 py-3 xl:whitespace-nowrap align-middle block xl:table-cell border-b border-slate-800/30 xl:border-0 last:border-0">
                        {id !== 'expander' && <div className="xl:hidden text-[10px] text-slate-500 font-semibold mb-1.5 uppercase tracking-wider">{label}</div>}
                        <div className="w-full overflow-x-auto xl:overflow-visible">
                          {flexRender(cell.column.columnDef.cell, cell.getContext())}
                        </div>
                      </td>
                    )})}
                  </tr>
                  {row.getIsExpanded() && (
                     <tr className="bg-slate-900/30 block xl:table-row border border-t-0 border-slate-800/50 rounded-b-lg -mt-4 xl:mt-0 mb-4 xl:mb-0">
                       <td colSpan={row.getVisibleCells().length} className="block xl:table-cell">
                          <div className="px-4 py-4 xl:pl-12">
                             <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">{t("components:addResource.propertiesLabel")}</h4>
                            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3">
                                {Object.entries(row.original.resource.properties || {}).map(([key, value]) => (
                                    <div key={key} className="flex flex-col bg-slate-950/50 p-3 rounded-lg border border-slate-800/50">
                                        <span className="text-xs text-slate-400 font-medium">{key}</span>
                                        <span className="text-sm text-slate-200 font-mono mt-1 break-all">{value}</span>
                                    </div>
                                ))}
                            </div>
                         </div>
                      </td>
                    </tr>
                  )}
                  </Fragment>
                ))
              )}
            </tbody>
          </table>
        </div>


      {selectedQueueResource && (
        <QueueListModal
            isOpen={!!selectedQueueResource}
            onClose={() => setSelectedQueueResource(null)}
            resourceId={selectedQueueResource.id}
            resourceName={selectedQueueResource.name}
        />
      )}
    </div>
  );
};

export default ResourceTable;
