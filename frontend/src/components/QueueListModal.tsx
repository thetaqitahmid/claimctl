import { X, User, Clock, Calendar } from "lucide-react";
import { useGetQueueForResourceQuery } from "../store/api/reservations";
import { formatDistanceToNow } from "date-fns";

interface QueueListModalProps {
  isOpen: boolean;
  onClose: () => void;
  resourceId: string;
  resourceName: string;
}

const QueueListModal = ({ isOpen, onClose, resourceId, resourceName }: QueueListModalProps) => {
  const { data: queue, isLoading, error } = useGetQueueForResourceQuery(resourceId, {
    skip: !isOpen,
    pollingInterval: isOpen ? 5000 : 0, // Poll every 5s while open
  });

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
      <div className="w-full max-w-md bg-slate-900 rounded-xl border border-slate-700 shadow-2xl overflow-hidden glass-panel animate-in fade-in zoom-in-95 duration-200">
        <div className="flex items-center justify-between p-4 border-b border-slate-800">
          <h2 className="text-lg font-semibold text-white flex items-center gap-2">
            <Clock className="w-5 h-5 text-cyan-400" />
            Queue for {resourceName}
          </h2>
          <button
            onClick={onClose}
            className="p-1 text-slate-400 hover:text-white rounded-lg hover:bg-slate-800 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-4 max-h-[60vh] overflow-y-auto">
          {isLoading ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-cyan-500"></div>
            </div>
          ) : error ? (
            <div className="text-red-400 text-center py-4">
              Failed to load queue. Please try again.
            </div>
          ) : queue?.length === 0 ? (
            <div className="text-slate-500 text-center py-8 italic">
              No active reservations or queue.
            </div>
          ) : (
            <div className="space-y-3">
              {queue?.map((item) => (
                <div
                  key={item.id}
                  className={`p-3 rounded-lg border flex flex-col gap-2 relative overflow-hidden ${
                    item.status === "active"
                      ? "bg-emerald-900/10 border-emerald-500/30"
                      : "bg-slate-800/50 border-slate-700"
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className={`p-2 rounded-full ${
                          item.status === "active" ? "bg-emerald-500/10 text-emerald-400" : "bg-slate-700 text-slate-400"
                      }`}>
                        <User className="w-4 h-4" />
                      </div>
                      <div>
                        <div className="font-medium text-slate-200">
                          {item.userName || "Unknown User"}
                        </div>
                        <div className="text-xs text-slate-500">
                          {item.userEmail}
                        </div>
                      </div>
                    </div>

                    {item.status === "active" ? (
                       <span className="px-2 py-1 rounded text-xs font-semibold bg-emerald-500/20 text-emerald-400 border border-emerald-500/20">
                          Active
                       </span>
                    ) : (
                        <span className="px-2 py-1 rounded text-xs font-semibold bg-cyan-500/20 text-cyan-400 border border-cyan-500/20">
                           #{item.queuePosition}
                        </span>
                    )}
                  </div>

                  <div className="flex items-center gap-4 text-xs text-slate-500 pl-11">
                    <div className="flex items-center gap-1">
                        <Calendar className="w-3 h-3" />
                         joined {formatDistanceToNow(item.createdAt * 1000)} ago
                    </div>
                    {item.status === 'active' && item.startTime && (
                         <div className="flex items-center gap-1 text-emerald-400/70">
                            <Clock className="w-3 h-3" />
                            started {formatDistanceToNow(item.startTime * 1000)} ago
                        </div>
                    )}
                    {item.duration ? (
                         <div className="flex items-center gap-1 text-slate-400/70">
                            <Clock className="w-3 h-3" />
                            for {item.duration}
                        </div>
                    ) : (
                        <div className="flex items-center gap-1 text-slate-400/50 italic">
                            <Clock className="w-3 h-3" />
                            for indefinite
                        </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default QueueListModal;
