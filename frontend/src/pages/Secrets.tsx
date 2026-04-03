import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { Plus, Trash, Edit2 } from 'lucide-react';

interface Secret {
  id: string;
  key: string;
  value: string;
  description: string;
  createdAt: number;
  updatedAt: number;
}

const Secrets: React.FC = () => {
    const [secrets, setSecrets] = useState<Secret[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [showCreateModal, setShowCreateModal] = useState(false);
    const [newSecret, setNewSecret] = useState({ key: '', value: '', description: '' });
    const [editId, setEditId] = useState<string | null>(null);
    const navigate = useNavigate();

    const fetchSecrets = useCallback(async () => {
        setIsLoading(true);
        try {
            const response = await fetch('/api/secrets', { credentials: 'include' });

            // Check if redirected to login (200 OK but URL is /login)
            if (response.redirected && response.url.includes('/login')) {
                 navigate('/login');
                 return;
            }

            if (!response.ok) {
                if (response.status === 401) {
                    navigate('/login');
                    return;
                }
                throw new Error('Failed to fetch secrets');
            }

            const contentType = response.headers.get("content-type");
            if (contentType && contentType.indexOf("application/json") !== -1) {
                const data = await response.json();
                setSecrets(data || []);
            } else {
                 // Fallback if response is text/html (likely error or redirect caught as 200)
                 const text = await response.text();
                 console.error("Received non-JSON response:", text);
                 // If it looks like HTML, probably a redirect issue
                 if (text.trim().startsWith('<')) {
                      // Treat as auth failure? Or just empty list?
                      // For now, assume auth failure if weird HTML response on API
                      navigate('/login');
                      return;
                 }
                 throw new Error("Invalid server response");
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
        fetchSecrets();
    }, [fetchSecrets]);

    const handleCreateSecret = async (e: React.FormEvent) => {
        e.preventDefault();
        try {
            const url = editId ? `/api/secrets/${editId}` : '/api/secrets';
            const method = editId ? 'PUT' : 'POST';

            const response = await fetch(url, {
                method: method,
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(newSecret)
            });

            if (!response.ok) {
                 const errorData = await response.json();
                throw new Error(errorData.error || 'Failed to create secret');
            }

            setShowCreateModal(false);
            setNewSecret({ key: '', value: '', description: '' });
            setEditId(null);
            fetchSecrets();
        } catch (err: unknown) {
            if (err instanceof Error) {
                alert(err.message);
            } else {
                alert(String(err));
            }
        }
    };

    const handleEditClick = (secret: Secret) => {
        setEditId(secret.id);
        setNewSecret({
            key: secret.key,
            value: '', // Force re-entry
            description: secret.description || ''
        });
        setShowCreateModal(true);
    };

    const handleDeleteSecret = async (id: string) => {
        if (!confirm('Are you sure you want to delete this secret?')) return;

        try {
            const response = await fetch(`/api/secrets/${id}`, {
                method: 'DELETE',
            });

            if (!response.ok) {
                throw new Error('Failed to delete secret');
            }

            fetchSecrets();
        } catch (err: unknown) {
            if (err instanceof Error) {
                alert(err.message);
            } else {
                alert(String(err));
            }
        }
    };

    return (
        <div className="min-h-screen bg-slate-950 text-slate-200 font-sans selection:bg-cyan-500/30">
            <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
                {/* Header */}
                <div className="flex justify-between items-center mb-8">
                    <h1 className="text-3xl font-bold text-white tracking-tight">Secrets Management</h1>
                     <button
                        onClick={() => {
                            setEditId(null);
                            setNewSecret({ key: '', value: '', description: '' });
                            setShowCreateModal(true);
                        }}
                        className="btn-primary flex items-center gap-2"
                    >
                        <Plus className="w-4 h-4" />
                        Create Secret
                    </button>
                </div>

                {error && (
                    <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded relative mb-4" role="alert">
                        <strong className="font-bold">Error: </strong>
                        <span className="block sm:inline">{error}</span>
                    </div>
                )}

                {isLoading ? (
                    <div className="flex justify-center items-center h-64">
                        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-cyan-600"></div>
                    </div>
                ) : (
                    <div className="glass-panel overflow-hidden rounded-xl border border-slate-800/50">
                        <table className="min-w-full divide-y divide-slate-800/50">
                            <thead className="bg-slate-900/50">
                                <tr>
                                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">Key</th>
                                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">Description</th>
                                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">Created At</th>
                                    <th scope="col" className="px-6 py-3 text-right text-xs font-medium text-slate-400 uppercase tracking-wider">Actions</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-slate-800/50 bg-transparent">
                                {secrets.map((secret) => (
                                    <tr key={secret.id} className="hover:bg-slate-800/30 transition-colors">
                                         <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-white">{secret.key}</td>
                                         <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-400">{secret.description}</td>
                                         <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500">
                                            {new Date(secret.createdAt * 1000).toLocaleString()}
                                         </td>
                                         <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium flex justify-end gap-2">
                                            <button
                                                onClick={() => handleEditClick(secret)}
                                                className="text-slate-400 hover:text-white transition-colors p-1"
                                                title="Edit"
                                            >
                                                <Edit2 className="w-4 h-4" />
                                            </button>
                                            <button
                                                onClick={() => handleDeleteSecret(secret.id)}
                                                className="text-red-400 hover:text-red-300 transition-colors p-1"
                                                title="Delete"
                                            >
                                                <Trash className="w-4 h-4" />
                                            </button>
                                         </td>
                                    </tr>
                                ))}
                                {secrets.length === 0 && (
                                    <tr>
                                        <td colSpan={4} className="px-6 py-8 text-center text-sm text-slate-500">
                                            No secrets found.
                                        </td>
                                    </tr>
                                )}
                            </tbody>
                        </table>
                    </div>
                )}
            </main>

            {/* Create Secret Modal */}
            {showCreateModal && (
                <div className="fixed inset-0 z-10 overflow-y-auto">
                    <div className="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
                        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm transition-opacity" aria-hidden="true" onClick={() => setShowCreateModal(false)}></div>
                        <span className="hidden sm:inline-block sm:align-middle sm:h-screen" aria-hidden="true">&#8203;</span>
                        <div className="inline-block align-bottom bg-slate-900 rounded-xl border border-slate-700 text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full">
                            <form onSubmit={handleCreateSecret}>
                                <div className="px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
                                    <h3 className="text-lg leading-6 font-bold text-white mb-4">
                                        {editId ? 'Update Secret' : 'Create New Secret'}
                                    </h3>
                                    <div className="mb-4">
                                        <label className="block text-slate-300 text-sm font-medium mb-2" htmlFor="key">
                                            Key
                                        </label>
                                        <input
                                            className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-cyan-500 outline-none placeholder-slate-500"
                                            id="key"
                                            type="text"
                                            placeholder="e.g. SLACK_TOKEN"
                                            value={newSecret.key}
                                            onChange={(e) => setNewSecret({...newSecret, key: e.target.value})}
                                            required
                                            disabled={!!editId}
                                        />
                                    </div>
                                    <div className="mb-4">
                                        <label className="block text-slate-300 text-sm font-medium mb-2" htmlFor="value">
                                            Value
                                        </label>
                                        <textarea
                                            className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-cyan-500 outline-none placeholder-slate-500 font-mono text-sm"
                                            id="value"
                                            placeholder="Secret Value"
                                            value={newSecret.value}
                                            onChange={(e) => setNewSecret({...newSecret, value: e.target.value})}
                                            required
                                        />
                                    </div>
                                    <div className="mb-4">
                                        <label className="block text-slate-300 text-sm font-medium mb-2" htmlFor="description">
                                            Description (Optional)
                                        </label>
                                        <input
                                            className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-white focus:ring-2 focus:ring-cyan-500 outline-none placeholder-slate-500"
                                            id="description"
                                            type="text"
                                            placeholder="Description"
                                            value={newSecret.description}
                                            onChange={(e) => setNewSecret({...newSecret, description: e.target.value})}
                                        />
                                    </div>
                                </div>
                                <div className="bg-slate-800/50 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse border-t border-slate-800">
                                    <button
                                        type="submit"
                                        className="w-full inline-flex justify-center rounded-lg border border-transparent shadow-sm px-4 py-2 bg-cyan-600 text-base font-medium text-white hover:bg-cyan-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-cyan-500 sm:ml-3 sm:w-auto sm:text-sm"
                                    >
                                        {editId ? 'Update' : 'Create'}
                                    </button>
                                    <button
                                        type="button"
                                        onClick={() => setShowCreateModal(false)}
                                        className="mt-3 w-full inline-flex justify-center rounded-lg border border-slate-600 shadow-sm px-4 py-2 bg-transparent text-base font-medium text-slate-300 hover:bg-slate-800 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-cyan-500 sm:mt-0 sm:ml-3 sm:w-auto sm:text-sm"
                                    >
                                        Cancel
                                    </button>
                                </div>
                            </form>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

export default Secrets;
