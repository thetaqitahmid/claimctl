import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { Plus, Trash, Edit2, Zap, FileText, X, Copy, Check } from 'lucide-react';

interface Webhook {
  id: string;
  name: string;
  url: string;
  method: string;
  headers: string; // JSON string
  template: string;
  description: string;
  signingSecret?: string;
}

interface WebhookLog {
  id: string;
  event: string;
  statusCode: number;
  requestBody: string;
  responseBody: string;
  durationMs: number;
  createdAt: string;
}

const Webhooks = () => {
    const [webhooks, setWebhooks] = useState<Webhook[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [showModal, setShowModal] = useState(false);
    const [isEditing, setIsEditing] = useState(false);
    const [currentWebhook, setCurrentWebhook] = useState<Partial<Webhook>>({
        method: 'POST',
        headers: '{}',
        template: '',
    });
    const [copied, setCopied] = useState(false);
    const [showLogsModal, setShowLogsModal] = useState(false);
    const [logs, setLogs] = useState<WebhookLog[]>([]);
    const [isLoadingLogs, setIsLoadingLogs] = useState(false);
    const [selectedWebhookName, setSelectedWebhookName] = useState("");

    const navigate = useNavigate();

    const fetchWebhooks = useCallback(async () => {
        setIsLoading(true);
        try {
            const response = await fetch(`/api/webhooks?t=${new Date().getTime()}`, {
                credentials: 'include',
                headers: { 'Cache-Control': 'no-cache' }
            });

            if (response.redirected && response.url.includes('/login')) {
                 navigate('/login');
                 return;
            }

            if (!response.ok) {
                if (response.status === 401) {
                    navigate('/login');
                    return;
                }
                throw new Error('Failed to fetch webhooks');
            }

            const contentType = response.headers.get("content-type");
            if (contentType && contentType.indexOf("application/json") !== -1) {
                const data = await response.json();
                // Ensure headers are strings for display
                const mappedData = data.map((w: Webhook & { headers: string | object }) => ({
                    ...w,
                    headers: typeof w.headers === 'object' ? JSON.stringify(w.headers, null, 2) : w.headers
                }));
                setWebhooks(mappedData || []);
            } else {
                const text = await response.text();
                 if (text.trim().startsWith('<')) {
                      navigate('/login');
                      return;
                 }
                 setWebhooks([]);
            }
        } catch (err: unknown) {
            if (err instanceof Error) {
                setError(err.message);
            } else {
                setError(String(err));
            }
        } finally {
            setIsLoading(false);
        }
    }, [navigate]);

    useEffect(() => {
        fetchWebhooks();
    }, [fetchWebhooks]);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        try {
            const url = isEditing ? `/api/webhooks/${currentWebhook.id}` : '/api/webhooks';
            const method = isEditing ? 'PUT' : 'POST';

            let parsedHeaders = {};
            try {
                 if (currentWebhook.headers) {
                     // Check if it's already an object (from editing without changing) or string
                     parsedHeaders = typeof currentWebhook.headers === 'string'
                        ? JSON.parse(currentWebhook.headers)
                        : currentWebhook.headers;
                 }
            } catch {
                alert("Invalid JSON in Headers field");
                return;
            }

            const payload = {
                ...currentWebhook,
                headers: parsedHeaders
            };

            const response = await fetch(url, {
                method: method,
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(payload)
            });

            if (!response.ok) {
                 const errorData = await response.json();
                throw new Error(errorData.error || 'Operation failed');
            }

            setShowModal(false);
            setCurrentWebhook({ method: 'POST', headers: '{}', template: '' });
            setIsEditing(false);
            fetchWebhooks();
        } catch (err: unknown) {
             if (err instanceof Error) {
                alert(err.message);
            } else {
                alert(String(err));
            }
        }
    };

    const handleDelete = async (id: string) => {
        if (!confirm('Are you sure?')) return;
        try {
            await fetch(`/api/webhooks/${id}`, {
                method: 'DELETE',
            });
            fetchWebhooks();
        } catch (err: unknown) {
             if (err instanceof Error) {
                alert(err.message);
            } else {
                alert(String(err));
            }
        }
    };

    const openEdit = (webhook: Webhook) => {
        setCurrentWebhook({ ...webhook });
        setIsEditing(true);
        setShowModal(true);
    };

    const openCreate = () => {
        setCurrentWebhook({ method: 'POST', headers: '{\n  "Content-Type": "application/json"\n}', template: '', name: '', url: '', description: '' });
        setIsEditing(false);
        setShowModal(true);
    };

    const openLogs = async (webhook: Webhook) => {
        setSelectedWebhookName(webhook.name);
        setShowLogsModal(true);
        setIsLoadingLogs(true);
        try {
            const res = await fetch(`/api/webhooks/${webhook.id}/logs`, { credentials: 'include' });
            if (res.ok) {
                const data = await res.json();
                setLogs(data || []);
            } else {
                setLogs([]);
            }
        } catch (e) {
            console.error(e);
        } finally {
            setIsLoadingLogs(false);
        }
    };

    return (
        <div className="min-h-screen bg-slate-950 text-slate-200 font-sans selection:bg-cyan-500/30">
            <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
                <div className="flex justify-between items-center mb-8">
                    <div>
                        <h1 className="text-3xl font-bold text-white tracking-tight flex items-center gap-3">
                            <Zap className="w-8 h-8 text-cyan-400" />
                            Webhooks
                        </h1>
                        <p className="text-slate-400 mt-1">Manage event callbacks and integrations</p>
                    </div>
                    <button
                        onClick={openCreate}
                        className="btn-primary flex items-center gap-2 bg-cyan-600 hover:bg-cyan-700 text-white px-4 py-2 rounded-md"
                    >
                        <Plus className="w-4 h-4" /> Create Webhook
                    </button>
                </div>

                {error && <div className="text-red-500 mb-4">{error}</div>}

                {isLoading ? (
                     <div className="text-white">Loading...</div>
                ) : (
                    <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
                        {webhooks.map((hook) => (
                            <div key={hook.id} className="glass-panel p-6 rounded-xl border border-slate-700 bg-slate-800/50 hover:border-cyan-500/50 transition-colors">
                                <div className="flex justify-between items-start mb-4">
                                    <h3 className="text-lg font-semibold text-white">{hook.name}</h3>
                                    <div className="flex gap-2">
                                        <button onClick={() => openLogs(hook)} className="text-slate-400 hover:text-white" title="View Logs">
                                            <FileText className="w-4 h-4" />
                                        </button>
                                        <button onClick={() => openEdit(hook)} className="text-slate-400 hover:text-white" title="Edit">
                                            <Edit2 className="w-4 h-4" />
                                        </button>
                                        <button onClick={() => handleDelete(hook.id)} className="text-red-400 hover:text-red-300">
                                            <Trash className="w-4 h-4" />
                                        </button>
                                    </div>
                                </div>

                                <div className="space-y-3">
                                    <div className="flex items-center gap-2">
                                        <span className={`px-2 py-1 rounded text-xs font-mono font-bold ${
                                            hook.method === 'GET' ? 'bg-cyan-900/50 text-cyan-400' :
                                            hook.method === 'POST' ? 'bg-green-900/50 text-green-400' :
                                            'bg-orange-900/50 text-orange-400'
                                        }`}>
                                            {hook.method}
                                        </span>
                                        <span className="text-slate-300 text-sm truncate font-mono" title={hook.url}>{hook.url}</span>
                                    </div>

                                    {hook.description && (
                                        <p className="text-sm text-slate-400 line-clamp-2" title={hook.description}>
                                            {hook.description}
                                        </p>
                                    )}
                                </div>
                            </div>
                        ))}
                        {webhooks.length === 0 && (
                             <div className="col-span-full text-center text-slate-500 py-12">
                                No webhooks configured.
                             </div>
                        )}
                    </div>
                )}
            </main>

            {showModal && (
                <div className="fixed inset-0 z-50 overflow-y-auto">
                    <div className="flex items-center justify-center min-h-screen px-4">
                        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm" onClick={() => setShowModal(false)}></div>

                        <div className="relative bg-slate-900 rounded-xl max-w-2xl w-full p-6 shadow-2xl border border-slate-700">
                            <h3 className="text-xl font-bold text-white mb-6">
                                {isEditing ? 'Edit Webhook' : 'New Webhook'}
                            </h3>

                            {isEditing && currentWebhook.signingSecret && (
                                <div className="mb-6 p-4 bg-slate-900 rounded-lg border border-slate-700">
                                    <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">
                                        Signing Secret
                                    </label>
                                    <div className="flex gap-2">
                                        <div className="flex-1 bg-slate-950 text-slate-300 font-mono text-sm px-3 py-2 rounded-lg border border-slate-800 truncate select-all">
                                            {currentWebhook.signingSecret}
                                        </div>
                                        <button
                                            type="button"
                                            onClick={() => {
                                                navigator.clipboard.writeText(currentWebhook.signingSecret || '');
                                                setCopied(true);
                                                setTimeout(() => setCopied(false), 2000);
                                            }}
                                            className="p-2 text-slate-400 hover:text-white bg-slate-800 border border-slate-700 rounded-lg transition-colors"
                                            title="Copy Secret"
                                        >
                                            {copied ? <Check className="w-5 h-5 text-green-500" /> : <Copy className="w-5 h-5" />}
                                        </button>
                                    </div>
                                    <p className="mt-2 text-xs text-slate-500">
                                        Use this secret to verify the authenticity of webhook requests.
                                    </p>
                                </div>
                            )}

                            <form onSubmit={handleSubmit} className="space-y-4">
                                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                    <div>
                                        <label className="block text-sm font-medium text-slate-300 mb-1">Name</label>
                                        <input
                                            className="w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-cyan-500 outline-none"
                                            value={currentWebhook.name || ''}
                                            onChange={(e) => setCurrentWebhook({...currentWebhook, name: e.target.value})}
                                            required
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-slate-300 mb-1">Method</label>
                                        <select
                                            className="w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-cyan-500 outline-none"
                                            value={currentWebhook.method}
                                            onChange={(e) => setCurrentWebhook({...currentWebhook, method: e.target.value})}
                                        >
                                            <option value="GET">GET</option>
                                            <option value="POST">POST</option>
                                            <option value="PUT">PUT</option>
                                            <option value="DELETE">DELETE</option>
                                            <option value="PATCH">PATCH</option>
                                        </select>
                                    </div>
                                </div>

                                <div>
                                    <label className="block text-sm font-medium text-slate-300 mb-1">Target URL</label>
                                    <input
                                        type="url"
                                        className="w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-cyan-500 outline-none font-mono text-sm"
                                        placeholder="https://api.example.com/webhook"
                                        value={currentWebhook.url || ''}
                                        onChange={(e) => setCurrentWebhook({...currentWebhook, url: e.target.value})}
                                        required
                                    />
                                </div>

                                <div>
                                    <label className="block text-sm font-medium text-slate-300 mb-1">
                                        Headers (JSON)
                                        <span className="text-slate-500 text-xs ml-2">Use {'{{Secret.KEY}}'} for secrets</span>
                                    </label>
                                    <textarea
                                        className="w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-cyan-500 outline-none font-mono text-sm"
                                        rows={4}
                                        value={currentWebhook.headers || '{}'}
                                        onChange={(e) => setCurrentWebhook({...currentWebhook, headers: e.target.value})}
                                    />
                                </div>

                                <div>
                                    <label className="block text-sm font-medium text-slate-300 mb-1">
                                        Payload Template (Optional)
                                        <span className="text-slate-500 text-xs ml-2">Go template syntax. Leave empty for default JSON.</span>
                                    </label>
                                    <textarea
                                        className="w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-cyan-500 outline-none font-mono text-sm"
                                        rows={4}
                                        placeholder='{"text": "Reservation {{.Reservation.ID}} created!"}'
                                        value={currentWebhook.template || ''}
                                        onChange={(e) => setCurrentWebhook({...currentWebhook, template: e.target.value})}
                                    />
                                </div>

                                <div>
                                    <label className="block text-sm font-medium text-slate-300 mb-1">Description</label>
                                    <input
                                        className="w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-cyan-500 outline-none"
                                        value={currentWebhook.description || ''}
                                        onChange={(e) => setCurrentWebhook({...currentWebhook, description: e.target.value})}
                                    />
                                </div>

                                <div className="flex justify-end gap-3 mt-6">
                                    <button
                                        type="button"
                                        onClick={() => setShowModal(false)}
                                        className="px-4 py-2 rounded-lg text-slate-300 hover:bg-slate-700 transition-colors"
                                    >
                                        Cancel
                                    </button>
                                    <button
                                        type="submit"
                                        className="bg-cyan-600 hover:bg-cyan-700 text-white px-6 py-2 rounded-lg font-medium transition-colors"
                                    >
                                        {isEditing ? 'Save Changes' : 'Create Webhook'}
                                    </button>
                                </div>
                            </form>
                        </div>
                    </div>
                </div>
            )}

            {showLogsModal && (
                <div className="fixed inset-0 z-50 overflow-y-auto">
                    <div className="flex items-center justify-center min-h-screen px-4">
                        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm" onClick={() => setShowLogsModal(false)}></div>
                        <div className="relative bg-slate-900 rounded-xl max-w-4xl w-full p-6 shadow-2xl border border-slate-700 max-h-[85vh] flex flex-col">
                            <div className="flex justify-between items-center mb-4">
                                <h3 className="text-xl font-bold text-white">logs: {selectedWebhookName}</h3>
                                <button onClick={() => setShowLogsModal(false)} className="text-slate-400 hover:text-white">
                                    <X className="w-6 h-6" />
                                </button>
                            </div>

                            <div className="flex-1 overflow-y-auto">
                                {isLoadingLogs ? (
                                    <div className="text-slate-500">Loading logs...</div>
                                ) : logs.length === 0 ? (
                                    <div className="text-slate-500">No execution logs found.</div>
                                ) : (
                                    <table className="w-full text-left text-sm text-slate-400">
                                        <thead className="text-xs uppercase bg-slate-800 text-slate-300">
                                            <tr>
                                                <th className="px-4 py-2">Status</th>
                                                <th className="px-4 py-2">Event</th>
                                                <th className="px-4 py-2">Duration</th>
                                                <th className="px-4 py-2">Time</th>
                                                <th className="px-4 py-2 text-right">Details</th>
                                            </tr>
                                        </thead>
                                        <tbody className="divide-y divide-slate-800">
                                            {logs.map((log) => (
                                                <React.Fragment key={log.id}>
                                                    <tr className="hover:bg-slate-800/50">
                                                        <td className="px-4 py-2">
                                                            <span className={`px-2 py-0.5 rounded text-xs font-bold ${
                                                                log.statusCode >= 200 && log.statusCode < 300 ? 'bg-green-900/50 text-green-400' :
                                                                log.statusCode >= 400 ? 'bg-red-900/50 text-red-400' :
                                                                'bg-slate-700 text-slate-300'
                                                            }`}>
                                                                {log.statusCode}
                                                            </span>
                                                        </td>
                                                        <td className="px-4 py-2">{log.event}</td>
                                                        <td className="px-4 py-2">{log.durationMs}ms</td>
                                                        <td className="px-4 py-2 whitespace-nowrap">{new Date(log.createdAt).toLocaleString()}</td>
                                                        <td className="px-4 py-2 text-right">
                                                            <details className="cursor-pointer">
                                                                <summary className="text-cyan-400 hover:text-cyan-300 text-xs">View Body</summary>
                                                                <div className="mt-2 p-2 bg-slate-950 rounded text-xs font-mono whitespace-pre-wrap text-left absolute z-10 w-96 right-10 border border-slate-700 shadow-xl">
                                                                    <div className="mb-2 border-b border-slate-800 pb-1 font-bold text-slate-300">Request:</div>
                                                                    <div className="mb-4 text-slate-400 max-h-32 overflow-y-auto">{log.requestBody}</div>
                                                                    <div className="mb-2 border-b border-slate-800 pb-1 font-bold text-slate-300">Response:</div>
                                                                    <div className="text-slate-400 max-h-32 overflow-y-auto">{log.responseBody}</div>
                                                                </div>
                                                            </details>
                                                        </td>
                                                    </tr>
                                                </React.Fragment>
                                            ))}
                                        </tbody>
                                    </table>
                                )}
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

export default Webhooks;
