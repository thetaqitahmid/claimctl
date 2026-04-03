import { useState } from "react";
import { Resource } from "../types";
import { X, History, Activity, Heart, Edit3 } from "lucide-react";
import ResourceHistory from "./ResourceHistory";
import HealthCheckConfig from "./HealthCheckConfig";
import EditResourceForm from "./EditResourceForm";
import { useAppSelector } from "../store/store";
import { format } from "date-fns";

interface ResourceDetailsModalProps {
  resource: Resource | null;
  onClose: () => void;
  onSave?: (updatedResource: Resource) => void;
}

const ResourceDetailsModal = ({
  resource,
  onClose,
  onSave,
}: ResourceDetailsModalProps) => {
  const [activeTab, setActiveTab] = useState<"overview" | "history" | "health" | "edit">("overview");
  const authData = useAppSelector((state) => state.authSlice);

  if (!resource) return null;

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-4 overflow-x-hidden overflow-y-auto outline-none focus:outline-none">
      <div
        className="fixed inset-0 bg-slate-950/80 backdrop-blur-md transition-opacity"
        onClick={onClose}
      />

      <div className="relative glass-panel w-full max-w-2xl rounded-2xl overflow-hidden shadow-2xl transition-all duration-200 flex flex-col max-h-[90vh]">
        {/* Header */}
        <div className="flex justify-between items-center p-6 border-b border-slate-800/50 shrink-0">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-cyan-500/10 text-cyan-400">
              <Activity className="h-5 w-5" />
            </div>
            <div>
                 <h3 className="text-xl font-semibold text-white">{resource.name}</h3>
                 <span className="text-xs text-slate-400 uppercase tracking-wider font-medium">{resource.type}</span>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-2 hover:bg-slate-800 rounded-lg text-slate-400 hover:text-white transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Tabs */}
        {authData.role === 'admin' && (
             <div className="flex border-b border-slate-800/50 px-6 shrink-0">
                <button
                    onClick={() => setActiveTab("edit")}
                    className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors flex items-center gap-2 ${
                        activeTab === "edit"
                        ? "border-cyan-500 text-cyan-400"
                        : "border-transparent text-slate-400 hover:text-white"
                    }`}
                >
                    <Edit3 className="w-4 h-4" />
                    Edit
                </button>
                <button
                    onClick={() => setActiveTab("overview")}
                    className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                        activeTab === "overview"
                        ? "border-cyan-500 text-cyan-400"
                        : "border-transparent text-slate-400 hover:text-white"
                    }`}
                >
                    Overview
                </button>
                <button
                    onClick={() => setActiveTab("history")}
                    className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors flex items-center gap-2 ${
                        activeTab === "history"
                        ? "border-cyan-500 text-cyan-400"
                        : "border-transparent text-slate-400 hover:text-white"
                    }`}
                >
                    <History className="w-4 h-4" />
                    History
                </button>
                <button
                    onClick={() => setActiveTab("health")}
                    className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors flex items-center gap-2 ${
                        activeTab === "health"
                        ? "border-cyan-500 text-cyan-400"
                        : "border-transparent text-slate-400 hover:text-white"
                    }`}
                >
                    <Heart className="w-4 h-4" />
                    Health Check
                </button>
             </div>
        )}

        {/* Content */}
        <div className="p-6 overflow-y-auto flex-1">
            {activeTab === "overview" && (
                <div className="space-y-6">
                    <div className="grid grid-cols-2 gap-4">
                        <div className="p-4 rounded-xl bg-slate-900/50 border border-slate-800/50">
                            <h4 className="text-sm font-medium text-slate-400 mb-1">Resource ID</h4>
                            <p className="text-white font-mono text-sm">#{resource.id}</p>
                        </div>
                        <div className="p-4 rounded-xl bg-slate-900/50 border border-slate-800/50">
                             <h4 className="text-sm font-medium text-slate-400 mb-1">Created At</h4>
                             <p className="text-white text-sm">
                                {resource.createdAt ? format(new Date(resource.createdAt * 1000), "PPP") : "N/A"}
                             </p>
                        </div>
                    </div>

                    <div>
                        <h4 className="text-sm font-medium text-slate-400 mb-2">Labels</h4>
                        <div className="flex flex-wrap gap-2">
                             {resource.labels && resource.labels.length > 0 ? (
                                 resource.labels.map((label, i) => (
                                     <span key={i} className="px-2.5 py-1 rounded-md bg-slate-800 text-slate-300 text-xs border border-slate-700">
                                         {label}
                                     </span>
                                 ))
                             ) : (
                                <span className="text-slate-500 italic text-sm">No labels</span>
                             )}
                        </div>
                    </div>
                </div>
            )}

            {activeTab === "history" && authData.role === 'admin' && (
                <ResourceHistory resourceId={resource.id} />
            )}

            {activeTab === "health" && authData.role === 'admin' && (
                <HealthCheckConfig resourceId={resource.id} />
            )}

            {activeTab === "edit" && authData.role === 'admin' && (
                <EditResourceForm
                    resource={resource}
                    onSave={(updated: Resource) => {
                        if (onSave) onSave(updated);
                        setActiveTab("overview");
                    }}
                />
            )}
        </div>
      </div>
    </div>
  );
};

export default ResourceDetailsModal;
