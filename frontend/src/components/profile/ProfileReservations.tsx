import { useGetUserReservationsQuery, useCancelReservationMutation, useCompleteReservationMutation } from "../../store/api/reservations";
import EmptyState from "../ui/EmptyState";
import LoadingSpinner from "../ui/LoadingSpinner";

const ProfileReservations = () => {
    const { data: reservations, isLoading: reservationsLoading } = useGetUserReservationsQuery();
    const [cancelReservation] = useCancelReservationMutation();
    const [completeReservation] = useCompleteReservationMutation();

    const activeReservations = reservations?.filter(r => r.status === 'active' || r.status === 'pending') || [];

    const handleCancelReservation = async (id: string) => {
        await cancelReservation(id);
    };

    const handleCompleteReservation = async (id: string) => {
        await completeReservation(id);
    };

    if (reservationsLoading) {
        return <LoadingSpinner />;
    }

    if (!activeReservations.length) {
        return <EmptyState message="No active reservations found." />;
    }

    return (
        <div className="animate-fade-in">
            <h2 className="text-xl font-light text-white mb-6">Current Reservations</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {activeReservations.map((reservation) => (
                    <div key={reservation.id} className="glass-panel p-6 rounded-xl hover:shadow-lg hover:shadow-cyan-500/5 transition-all duration-300 border border-slate-700/50 group flex flex-col justify-between">
                        <div>
                            <div className="flex justify-between items-start mb-4">
                                <div className="bg-slate-800/50 p-3 rounded-lg group-hover:bg-slate-800 transition-colors">
                                    <span className="text-2xl">📅</span>
                                </div>
                                <span className={`px-2 py-1 rounded-md text-xs font-medium border ${
                                    reservation.status === 'active'
                                        ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                                        : 'bg-amber-500/10 text-amber-400 border-amber-500/20'
                                    }`}>
                            {reservation.status.toUpperCase()}
                          </span>
                            </div>
                            <h3 className="text-lg font-medium text-white mb-2">Resource #{reservation.resourceId}</h3>

                            {reservation.status === 'pending' && (
                                <div className="text-sm text-slate-400 mt-2 mb-4">
                                    Queue Position: <span className="text-cyan-400 font-mono font-bold">{reservation.queuePosition}</span>
                                </div>
                            )}
                        </div>

                        <div className="flex gap-2 mt-4 pt-4 border-t border-slate-700/50">
                            {reservation.status === 'active' ? (
                                <button
                                    onClick={() => handleCompleteReservation(reservation.id)}
                                    className="flex-1 bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 text-xs font-medium py-2 px-3 rounded-lg transition-colors border border-emerald-500/20"
                                >
                                    Complete
                                </button>
                            ) : (
                                <button
                                    onClick={() => handleCancelReservation(reservation.id)}
                                    className="flex-1 bg-red-500/10 hover:bg-red-500/20 text-red-400 text-xs font-medium py-2 px-3 rounded-lg transition-colors border border-red-500/20"
                                >
                                    Cancel
                                </button>
                            )}
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
};

export default ProfileReservations;
