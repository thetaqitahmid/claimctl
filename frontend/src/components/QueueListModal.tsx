import { Dialog } from "@headlessui/react";
import { useTranslation } from "react-i18next";
import { X, User, Clock } from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { useGetQueueForResourceQuery } from "../store/api/reservations";

interface QueueListModalProps {
  isOpen: boolean;
  onClose: () => void;
  resourceId: string;
  resourceName: string;
}

const QueueListModal = ({ isOpen, onClose, resourceId, resourceName }: QueueListModalProps) => {
  const { t } = useTranslation(["components", "common"]);
  const { data: queue, isLoading, isFetching, error, refetch } = useGetQueueForResourceQuery(resourceId, {
    skip: !isOpen, pollingInterval: isOpen ? 5000 : 0,
  });

  return (
    <Dialog open={isOpen} onClose={onClose} className="relative z-50">
      <div className="fixed inset-0 bg-black/50 backdrop-blur-sm" aria-hidden="true" />
      <div className="fixed inset-0 flex items-center justify-center p-4">
        <Dialog.Panel className="w-full max-w-md bg-slate-900 rounded-xl border border-slate-700 shadow-2xl overflow-hidden glass-panel">
          <div className="flex items-center justify-between p-4 border-b border-slate-800 gap-3">
            <Dialog.Title className="text-lg font-semibold text-white flex items-center gap-2">
              <Clock className="w-5 h-5 text-cyan-400 shrink-0" aria-hidden="true" />
              {t("components:queueModal.title", { resourceName })}
            </Dialog.Title>
            <button type="button" onClick={onClose} aria-label={t("common:close")}
              className="p-2 text-slate-400 hover:text-white rounded-lg hover:bg-slate-800">
              <X className="w-5 h-5" aria-hidden="true" />
            </button>
          </div>
          <div className="p-4 max-h-[60vh] overflow-y-auto" aria-busy={isFetching}>
            {isLoading && <p role="status" className="text-slate-400 py-8 text-center">{t("common:loading")}</p>}
            {error && <div role="alert" className="text-red-400 text-center py-4">
              <p>{t("components:queueModal.error")}</p>
              <button type="button" onClick={() => refetch()} className="underline p-2">{t("common:refresh")}</button>
            </div>}
            {!isLoading && !error && queue?.length === 0 && (
              <p className="text-slate-500 text-center py-8 italic">{t("components:queueModal.empty")}</p>
            )}
            <ul className="space-y-3">
              {queue?.map((item) => (
                <li key={item.id} className={`p-3 rounded-lg border space-y-2 ${
                  item.status === "active" ? "bg-emerald-900/10 border-emerald-500/30" : "bg-slate-800/50 border-slate-700"
                }`}>
                  <div className="flex items-center justify-between gap-2">
                    <div className="flex items-center gap-3 min-w-0">
                      <User className="w-4 h-4 shrink-0 text-slate-400" aria-hidden="true" />
                      <div className="min-w-0 break-words">
                        <p className="font-medium text-slate-200">{item.userName || t("components:queueModal.unknownUser")}</p>
                        <p className="text-xs text-slate-500">{item.userEmail}</p>
                      </div>
                    </div>
                    <span className="text-xs text-cyan-400 shrink-0">
                      {item.status === "active" ? t("common:status.active") : t("components:queueModal.position", { position: item.queuePosition })}
                    </span>
                  </div>
                  <div className="flex flex-wrap gap-3 text-xs text-slate-400">
                    <span>{t("components:queueModal.joined", { elapsed: formatDistanceToNow(item.createdAt * 1000) })}</span>
                    {item.status === "active" && !!item.startTime && <span>
                      {t("components:queueModal.started", { elapsed: formatDistanceToNow(item.startTime * 1000) })}
                    </span>}
                    <span>{t("components:queueModal.duration", { duration: item.duration || t("components:reservationModal.indefinite") })}</span>
                  </div>
                </li>
              ))}
            </ul>
          </div>
        </Dialog.Panel>
      </div>
    </Dialog>
  );
};

export default QueueListModal;
