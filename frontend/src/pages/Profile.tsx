import { useState } from "react";
import { useAppSelector } from "../store/store";
import ProfileReservations from "../components/profile/ProfileReservations";
import ProfileHistory from "../components/profile/ProfileHistory";
import ProfileNotifications from "../components/profile/ProfileNotifications";
import ProfileTokens from "../components/profile/ProfileTokens";
import Tabs, { Tab } from "../components/ui/Tabs";
import ChangePasswordModal from "../components/profile/ChangePasswordModal";

const Profile = () => {
  const authData = useAppSelector((state) => state.authSlice);
  const [activeTab, setActiveTab] = useState("reservations");
  const [isChangePasswordModalOpen, setIsChangePasswordModalOpen] = useState(false);

  const tabs: Tab[] = [
      { id: 'reservations', label: 'My Reservations' },
      { id: 'history', label: 'Activity Log' },
      { id: 'notifications', label: 'Notifications' },
      { id: 'api-tokens', label: 'API Tokens' },
  ];

  return (
    <div className="flex-grow container mx-auto px-4 py-8 max-w-7xl">
        <div className="glass-panel p-8 rounded-2xl mb-8 animate-fade-in-up">
          <div className="flex flex-col md:flex-row items-start md:items-center gap-6">
            <div className="w-24 h-24 rounded-full bg-gradient-to-br from-cyan-400 to-cyan-600 flex items-center justify-center text-3xl font-bold text-white shadow-lg shadow-cyan-500/20">
              {authData.user ? authData.user.charAt(0).toUpperCase() : "U"}
            </div>
            <div>
              <h1 className="text-3xl font-light text-white mb-2">{authData.user}</h1>
              <p className="text-slate-400 text-sm">User Profile</p>
              {authData.role === 'admin' && (
                 <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-cyan-500/10 text-cyan-400 border border-cyan-500/20 mt-2">
                   Admin
                 </span>
              )}
            </div>

            <button
                onClick={() => setIsChangePasswordModalOpen(true)}
                className="ml-auto px-4 py-2 bg-slate-800/50 hover:bg-slate-700/50 text-white text-sm font-medium rounded-lg border border-slate-700/50 transition-all"
            >
                Change Password
            </button>
          </div>
        </div>

        <Tabs tabs={tabs} activeTab={activeTab} onTabChange={setActiveTab} className="mb-6" />

        <div className="min-h-[400px]">
          {activeTab === "reservations" && <ProfileReservations />}
          {activeTab === "history" && <ProfileHistory />}
          {activeTab === "notifications" && <ProfileNotifications />}
          {activeTab === "api-tokens" && <ProfileTokens />}
        </div>

        <ChangePasswordModal
            isOpen={isChangePasswordModalOpen}
            onClose={() => setIsChangePasswordModalOpen(false)}
        />
    </div>
  );
};

export default Profile;
