import { useState, useEffect } from "react";
import { useGetPreferencesQuery, useUpdatePreferenceMutation, useUpdateChannelConfigMutation, useGetUserChannelConfigQuery, useTestUserEmailConfigMutation } from "../../store/api/preferences";
import { useNotificationContext } from "../../hooks/useNotification";
import LoadingSpinner from "../ui/LoadingSpinner";
import HelpPopover from "../ui/HelpPopover";
import { EVENTS, CHANNELS } from "../../constants";

const ProfileNotifications = () => {
    const { data: preferences = [], isLoading: preferencesLoading } = useGetPreferencesQuery();
    const { data: channelConfig } = useGetUserChannelConfigQuery();
    const [updatePreference] = useUpdatePreferenceMutation();
    const [updateChannelConfig] = useUpdateChannelConfigMutation();
    const [testEmailConfig, { isLoading: isTestingEmail }] = useTestUserEmailConfigMutation();
    const { showNotification } = useNotificationContext();

    const [slackDest, setSlackDest] = useState('');
    const [teamsUrl, setTeamsUrl] = useState('');
    const [notificationEmail, setNotificationEmail] = useState('');

    useEffect(() => {
        if (channelConfig) {
            setSlackDest(channelConfig.slack_destination || '');
            setTeamsUrl(channelConfig.teams_webhook_url || '');
            setNotificationEmail(channelConfig.notification_email || '');
        }
    }, [channelConfig]);

    const isEnabled = (event: string, channel: string) => {
        const pref = preferences?.find(p => p.eventType === event && p.channel === channel);
        if (!pref) {
            return false;
        }
        return pref.enabled;
    };

    const handleToggle = async (event: string, channel: string) => {
        const current = isEnabled(event, channel);
        try {
            await updatePreference({
                eventType: event,
                channel: channel,
                enabled: !current
            }).unwrap();
        } catch (err) {
            console.error("Failed to update preference", err);
        }
    };

    const handleSaveConfig = async () => {
        try {
            await updateChannelConfig({
                slack_destination: slackDest,
                teams_webhook_url: teamsUrl,
                notification_email: notificationEmail
            }).unwrap();
            showNotification('success', 'Channel configuration saved.');
        } catch (err) {
            console.error("Failed to save config", err);
            showNotification('error', 'Failed to save configuration.');
        }
    };

    const handleTestEmail = async () => {
        try {
            await testEmailConfig().unwrap();
            showNotification('success', 'Test email sent successfully! Please check your inbox.');
        } catch (err) {
            const error = err as { data?: { error?: string }; message?: string };
            console.error("Failed to send test email", error);
            showNotification('error', `Failed to send test email: ${error?.data?.error || error.message || 'Unknown error'}`);
        }
    };

    return (
        <div className="animate-fade-in space-y-8">
            <div className="glass-panel p-6 rounded-xl border border-slate-700/50">
                <h2 className="text-xl font-light text-white mb-4">Channel Configuration</h2>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6 max-w-4xl">
                    <div>
                        <label className="block text-sm font-medium text-slate-300 mb-2">Notification Email</label>
                        <input
                            type="email"
                            className="w-full bg-slate-800/50 border border-slate-700 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-cyan-500 transition-colors"
                            value={notificationEmail}
                            onChange={(e) => setNotificationEmail(e.target.value)}
                            placeholder="default@example.com"
                        />
                        <p className="text-xs text-slate-400 mt-2">Leave blank to use your login email for notifications.</p>
                        <button
                            onClick={handleTestEmail}
                            disabled={isTestingEmail}
                            className="mt-4 text-sm bg-cyan-500/10 hover:bg-cyan-500/20 text-cyan-400 font-medium py-1.5 px-4 rounded-lg transition-colors border border-cyan-500/20 disabled:opacity-50 disabled:cursor-not-allowed inline-flex items-center gap-2"
                        >
                            {isTestingEmail ? (
                                <>
                                    <div className="w-3.5 h-3.5 border-2 border-cyan-400 border-t-transparent rounded-full animate-spin"></div>
                                    Sending...
                                </>
                            ) : (
                                'Test Email Configuration'
                            )}
                        </button>
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-slate-300 mb-2">Slack Destination</label>
                        <input
                            type="text"
                            className="w-full bg-slate-800/50 border border-slate-700 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-cyan-500 transition-colors"
                            value={slackDest}
                            onChange={(e) => setSlackDest(e.target.value)}
                            placeholder="Webhook URL or Channel ID"
                        />
                        <HelpPopover label="How to get this?">
                            <p><strong className="text-slate-300">Option 1: Webhook URL</strong></p>
                            <ol className="list-decimal pl-4 space-y-1">
                                <li>Go to <a href="https://api.slack.com/apps" target="_blank" rel="noopener noreferrer" className="text-cyan-500 hover:underline">Slack Apps</a> and create a new app.</li>
                                <li>Enable "Incoming Webhooks" and add a new webhook to your channel.</li>
                                <li>Copy the Webhook URL (starts with <code className="bg-slate-900 px-1 rounded">https://hooks.slack.com/...</code>).</li>
                            </ol>
                            <p className="pt-1"><strong className="text-slate-300">Option 2: Channel ID (Bot)</strong></p>
                            <ol className="list-decimal pl-4 space-y-1">
                                <li>Right-click your channel in Slack sidebar {'>'} <strong>Copy Link</strong>.</li>
                                <li>Paste it somewhere, the ID is the last part (e.g., <code className="bg-slate-900 px-1 rounded">C01234ABCDE</code>).</li>
                                <li>Invite your Bot App to this channel using <code className="bg-slate-900 px-1 rounded">/invite @YourBotName</code>.</li>
                            </ol>
                        </HelpPopover>
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-slate-300 mb-2">Microsoft Teams Webhook</label>
                        <input
                            type="text"
                            className="w-full bg-slate-800/50 border border-slate-700 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-cyan-500 transition-colors"
                            value={teamsUrl}
                            onChange={(e) => setTeamsUrl(e.target.value)}
                            placeholder="https://outlook.office.com/webhook/..."
                        />
                        <HelpPopover label="How to get this?">
                            <p><strong className="text-slate-300">Setup Incoming Webhook</strong></p>
                            <ol className="list-decimal pl-4 space-y-1">
                                <li>Go to your Teams Channel {'>'} three dots (...) {'>'} <strong>Connectors</strong>.</li>
                                <li>Search for <strong>"Incoming Webhook"</strong> and click <strong>Configure</strong>.</li>
                                <li>Give it a name, upload an icon if you want, and click <strong>Create</strong>.</li>
                                <li>Copy the generated URL (it's very long, starts with <code className="bg-slate-900 px-1 rounded">https://outlook.office.com/webhook/...</code>).</li>
                            </ol>
                        </HelpPopover>
                    </div>
                </div>
                <div className="mt-6 flex justify-end">
                    <button
                        onClick={handleSaveConfig}
                        className="bg-cyan-500/10 hover:bg-cyan-500/20 text-cyan-400 font-medium py-2 px-6 rounded-lg transition-colors border border-cyan-500/20"
                    >
                        Save Configuration
                    </button>
                </div>
            </div>

            <div className="glass-panel rounded-xl overflow-hidden border border-slate-700/50">
                <div className="px-6 py-4 border-b border-slate-700/50">
                    <h2 className="text-xl font-light text-white">Event Preferences</h2>
                </div>
                {preferencesLoading ? (
                    <LoadingSpinner />
                ) : (
                    <table className="w-full text-left text-sm text-slate-400">
                        <thead className="bg-slate-800/50 text-slate-200 uppercase text-xs font-medium tracking-wider">
                        <tr>
                            <th className="px-6 py-4">Event Type</th>
                            {CHANNELS.map(channel => (
                                <th key={channel} className="px-6 py-4 text-center">{channel}</th>
                            ))}
                        </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-700/50">
                        {EVENTS.map(event => (
                            <tr key={event} className="hover:bg-slate-800/30 transition-colors">
                                <td className="px-6 py-4 font-medium text-white capitalize">
                                    {event.replace('reservation.', 'Reservation ')}
                                </td>
                                {CHANNELS.map(channel => (
                                    <td key={channel} className="px-6 py-4 text-center">
                                        <input
                                            type="checkbox"
                                            className="h-4 w-4 rounded border-slate-600 bg-slate-700/50 text-cyan-500 focus:ring-cyan-500/50"
                                            checked={isEnabled(event, channel)}
                                            onChange={() => handleToggle(event, channel)}
                                        />
                                    </td>
                                ))}
                            </tr>
                        ))}
                        </tbody>
                    </table>
                )}
            </div>
        </div>
    );
};

export default ProfileNotifications;
