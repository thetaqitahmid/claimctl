import { format } from "date-fns";
import { useGetUserHistoryQuery } from "../../store/api/reservations";
import EmptyState from "../ui/EmptyState";
import LoadingSpinner from "../ui/LoadingSpinner";

const ProfileHistory = () => {
    const { data: history, isLoading: historyLoading } = useGetUserHistoryQuery();

    if (historyLoading) {
        return <LoadingSpinner />;
    }

    if (!history || history.length === 0) {
        return <EmptyState message="No activity history found." />;
    }

    return (
        <div className="animate-fade-in">
            <h2 className="text-xl font-light text-white mb-6">Activity History</h2>
            <div className="glass-panel rounded-xl overflow-hidden border border-slate-700/50">
                <table className="w-full text-left text-sm text-slate-400">
                    <thead className="bg-slate-800/50 text-slate-200 uppercase text-xs font-medium tracking-wider">
                    <tr>
                        <th className="px-6 py-4">Resource</th>
                        <th className="px-6 py-4">Action</th>
                        <th className="px-6 py-4">Date</th>
                        <th className="px-6 py-4">Details</th>
                    </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-700/50">
                    {history.map((item) => (
                        <tr key={item.id} className="hover:bg-slate-800/30 transition-colors">
                            <td className="px-6 py-4 font-medium text-white">{item.resourceName}</td>
                            <td className="px-6 py-4">
                            <span className={`inline-flex px-2 py-1 rounded text-xs font-medium border ${
                                item.action === 'created' ? 'bg-cyan-500/10 text-cyan-400 border-cyan-500/20' :
                                    item.action === 'cancelled' ? 'bg-red-500/10 text-red-400 border-red-500/20' :
                                        item.action === 'completed' ? 'bg-green-500/10 text-green-400 border-green-500/20' :
                                            item.action === 'activated' ? 'bg-amber-500/10 text-amber-400 border-amber-500/20' :
                                                'bg-slate-700 text-slate-300'
                            }`}>
                                {item.action.toUpperCase()}
                            </span>
                            </td>
                            <td className="px-6 py-4">
                                {format(new Date(item.timestamp * 1000), "PPP p")}
                            </td>
                            <td className="px-6 py-4">
                                {item.details ? JSON.stringify(item.details) : "-"}
                            </td>
                        </tr>
                    ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
};

export default ProfileHistory;
