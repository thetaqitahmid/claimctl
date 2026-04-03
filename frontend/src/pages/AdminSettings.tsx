import React, { useState } from 'react';
import { useGetSettingsQuery, useUpdateSettingMutation, AppSetting } from '../store/api/settings';
import { useAppSelector } from '../store/store';
import BackupRestore from '../components/BackupRestore';
import { useNotificationContext } from '../hooks/useNotification';

const AdminSettings: React.FC = () => {
    const authData = useAppSelector((state) => state.authSlice);
    const { data: settings = [], isLoading, refetch } = useGetSettingsQuery();
    const [updateSetting] = useUpdateSettingMutation();
    const [editingKey, setEditingKey] = useState<string | null>(null);
    const [editValue, setEditValue] = useState('');
    const [activeTab, setActiveTab] = useState<'auth' | 'notification' | 'backup'>('auth');
    const { showNotification } = useNotificationContext();

    if (authData.role !== 'admin') {
        return <div className="p-8 text-center text-red-500">Access Denied</div>;
    }

    if (isLoading) {
        return <div className="p-8 text-center">Loading settings...</div>;
    }

    const categories = ['auth', 'notification', 'backup'];

    const filteredSettings = settings.filter(s => s.category === activeTab);

    const handleEdit = (setting: AppSetting) => {
        setEditingKey(setting.key);
        setEditValue(setting.is_secret ? '' : setting.value); // Don't show masked value
    };

    const handleSave = async (setting: AppSetting) => {
        try {
            await updateSetting({ ...setting, value: editValue }).unwrap();
            setEditingKey(null);
            setEditValue('');
            refetch();
            showNotification('success', 'Setting updated successfully');
        } catch (err) {
            const error = err as { data?: { error?: string }; message?: string };
            console.error("Failed to update setting", error);
            showNotification('error', `Failed to update setting: ${error?.data?.error || error.message || 'Unknown error'}`);
        }
    };

    const handleCancel = () => {
        setEditingKey(null);
        setEditValue('');
    };

    return (
        <div className="min-h-screen flex flex-col font-sans selection:bg-cyan-500/30">
            <main className="flex-grow container mx-auto px-4 py-8 max-w-7xl">
                <div className="glass-panel p-8 rounded-2xl mb-8 animate-fade-in-up">
                    <h1 className="text-3xl font-light text-white mb-2">System Settings</h1>
                    <p className="text-slate-400 text-sm">Manage global application configurations</p>
                </div>

                {/* Tabs */}
                <div className="flex gap-4 mb-6 border-b border-slate-700/50 pb-1">
                    {categories.map((category) => (
                        <button
                            key={category}
                            onClick={() => setActiveTab(category as 'auth' | 'notification' | 'backup')}
                            className={`pb-3 px-4 text-sm font-medium transition-all duration-200 relative capitalize ${
                                activeTab === category
                                    ? "text-cyan-400"
                                    : "text-slate-400 hover:text-slate-200"
                            }`}
                        >
                            {category}
                            {activeTab === category && (
                                <span className="absolute bottom-[-1px] left-0 w-full h-0.5 bg-cyan-400 rounded-full" />
                            )}
                        </button>
                    ))}
                </div>

                {activeTab === 'backup' ? (
                    <BackupRestore />
                ) : (
                <div className="glass-panel rounded-xl overflow-hidden border border-slate-700/50 animate-fade-in">
                    <ul className="divide-y divide-slate-700/50">
                        {filteredSettings.map((setting) => (
                            <li key={setting.key} className="px-6 py-6 hover:bg-slate-800/30 transition-colors">
                                <div className="flex items-center justify-between">
                                    <div className="flex-1 min-w-0 pr-8">
                                        <h4 className="text-lg font-medium text-white truncate">{setting.key}</h4>
                                        <p className="text-sm text-slate-400 mt-1">{setting.description}</p>
                                    </div>
                                    <div className="flex-shrink-0 max-w-[60%]">
                                        {editingKey === setting.key ? (
                                            <div className="flex items-center gap-3 justify-end">
                                                <input
                                                    type={setting.is_secret ? "password" : "text"}
                                                    value={editValue}
                                                    onChange={(e) => setEditValue(e.target.value)}
                                                    className="bg-slate-800/50 border border-slate-700 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-cyan-500 transition-colors sm:text-sm w-full max-w-xs"
                                                    placeholder={setting.is_secret ? "Enter new secret" : "Value"}
                                                />
                                                <button
                                                    onClick={() => handleSave(setting)}
                                                    className="bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 border border-emerald-500/20 px-4 py-2 rounded-lg text-sm font-medium transition-colors shrink-0"
                                                >
                                                    Save
                                                </button>
                                                <button
                                                    onClick={handleCancel}
                                                    className="text-slate-400 hover:text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors shrink-0"
                                                >
                                                    Cancel
                                                </button>
                                            </div>
                                        ) : (
                                            <div className="flex items-center gap-6 justify-end">
                                                <span className={`text-sm break-all ${setting.value ? 'text-slate-200' : 'text-slate-500 italic'}`}>
                                                    {setting.is_secret ? (setting.value ? '••••••••' : 'Not Set') : (setting.value || 'Not Set')}
                                                </span>
                                                <button
                                                    onClick={() => handleEdit(setting)}
                                                    className="text-cyan-400 hover:text-cyan-300 font-medium text-sm transition-colors shrink-0"
                                                >
                                                    Edit
                                                </button>
                                            </div>
                                        )}
                                    </div>
                                </div>
                            </li>
                        ))}
                        {filteredSettings.length === 0 && (
                            <li className="px-6 py-12 text-center text-slate-500 italic">
                                No settings found in this category.
                            </li>
                        )}
                    </ul>
                </div>
                )}
            </main>
        </div>
    );
};

export default AdminSettings;
