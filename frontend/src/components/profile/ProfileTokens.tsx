import { useState } from "react";
import { format } from "date-fns";
import { useGetTokensQuery, useCreateTokenMutation, useRevokeTokenMutation } from "../../store/api/tokens";
import LoadingSpinner from "../ui/LoadingSpinner";
import EmptyState from "../ui/EmptyState";

const ProfileTokens = () => {
    const { data: tokens, isLoading: tokensLoading } = useGetTokensQuery(undefined);
    const [createToken] = useCreateTokenMutation();
    const [revokeToken] = useRevokeTokenMutation();

    const [isTokenModalOpen, setIsTokenModalOpen] = useState(false);
    const [newTokenName, setNewTokenName] = useState('');
    const [newTokenExpiry, setNewTokenExpiry] = useState(''); // "" = never
    const [generatedToken, setGeneratedToken] = useState<string | null>(null);

    const handleGenerateToken = async () => {
        if (!newTokenName) return;
        try {
            const res = await createToken({
                name: newTokenName,
                expires_in: newTokenExpiry || undefined
            }).unwrap();
            setGeneratedToken(res.token);
            setNewTokenName('');
            setNewTokenExpiry(''); // reset
        } catch (err) {
            console.error("Failed to generate token", err);
        }
    };

    const handleCloseTokenModal = () => {
        setIsTokenModalOpen(false);
        setGeneratedToken(null);
        setNewTokenName('');
        setNewTokenExpiry('');
    };

    return (
        <div className="animate-fade-in">
            <div className="flex justify-between items-center mb-6">
                <h2 className="text-xl font-light text-white">Access Tokens</h2>
                <button
                    onClick={() => setIsTokenModalOpen(true)}
                    className="bg-cyan-500/10 hover:bg-cyan-500/20 text-cyan-400 text-sm font-medium py-2 px-4 rounded-lg transition-colors border border-cyan-500/20"
                >
                    + Generate Token
                </button>
            </div>

            {tokensLoading ? (
                <LoadingSpinner />
            ) : tokens && tokens.length > 0 ? (
                <div className="glass-panel rounded-xl overflow-hidden border border-slate-700/50">
                    <table className="w-full text-left text-sm text-slate-400">
                        <thead className="bg-slate-800/50 text-slate-200 uppercase text-xs font-medium tracking-wider">
                        <tr>
                            <th className="px-6 py-4">Name</th>
                            <th className="px-6 py-4">Created At</th>
                            <th className="px-6 py-4">Expires</th>
                            <th className="px-6 py-4">Last Used</th>
                            <th className="px-6 py-4 text-right">Actions</th>
                        </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-700/50">
                        {tokens.map((token) => (
                            <tr key={token.id} className="hover:bg-slate-800/30 transition-colors">
                                <td className="px-6 py-4 font-medium text-white">{token.name}</td>
                                <td className="px-6 py-4">{format(new Date(token.createdAt), "PPP p")}</td>
                                <td className="px-6 py-4">
                                    {token.expiresAt ? format(new Date(token.expiresAt), "PPP p") : "Never"}
                                </td>
                                <td className="px-6 py-4">
                                    {token.lastUsedAt ? format(new Date(token.lastUsedAt), "PPP p") : "Never"}
                                </td>
                                <td className="px-6 py-4 text-right">
                                    <button
                                        onClick={() => revokeToken(token.id)}
                                        className="text-red-400 hover:text-red-300 hover:underline text-xs"
                                    >
                                        Revoke
                                    </button>
                                </td>
                            </tr>
                        ))}
                        </tbody>
                    </table>
                </div>
            ) : (
                <EmptyState message="No active tokens found. Generate one to access the API." />
            )}

            {/* Token Generation Modal */}
            {isTokenModalOpen && (
                <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
                    <div className="bg-slate-900 border border-slate-700 rounded-xl max-w-md w-full p-6 shadow-2xl animate-fade-in-up">
                        <h3 className="text-xl font-semibold text-white mb-4">
                            {generatedToken ? "Token Generated" : "Generate New API Token"}
                        </h3>

                        {generatedToken ? (
                            <div className="space-y-4">
                                <div className="bg-amber-500/10 border border-amber-500/20 text-amber-200 text-sm p-3 rounded-lg">
                                    Make sure to copy your personal access token now. You won’t be able to see it again!
                                </div>
                                <div className="bg-slate-950 p-4 rounded-lg font-mono text-cyan-400 break-all border border-slate-800">
                                    {generatedToken}
                                </div>
                                <div className="flex justify-end pt-2">
                                    <button
                                        onClick={handleCloseTokenModal}
                                        className="bg-cyan-500 hover:bg-cyan-600 text-white font-medium py-2 px-4 rounded-lg transition-colors"
                                    >
                                        Done
                                    </button>
                                </div>
                            </div>
                        ) : (
                            <div className="space-y-4">
                                <div>
                                    <label className="block text-sm font-medium text-slate-300 mb-2">Token Name</label>
                                    <input
                                        type="text"
                                        className="w-full bg-slate-800 border border-slate-700 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-cyan-500 transition-colors"
                                        placeholder="e.g. CI Pipeline, Development"
                                        value={newTokenName}
                                        onChange={(e) => setNewTokenName(e.target.value)}
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-300 mb-2">Expiration</label>
                                    <select
                                        className="w-full bg-slate-800 border border-slate-700 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-cyan-500 transition-colors"
                                        value={newTokenExpiry}
                                        onChange={(e) => setNewTokenExpiry(e.target.value)}
                                    >
                                        <option value="">Never</option>
                                        <option value="720h">30 Days</option>
                                        <option value="2160h">90 Days</option>
                                        <option value="4320h">6 Months</option>
                                        <option value="8760h">1 Year</option>
                                    </select>
                                </div>
                                <div className="flex justify-end gap-3 pt-4">
                                    <button
                                        onClick={() => setIsTokenModalOpen(false)}
                                        className="text-slate-400 hover:text-white font-medium py-2 px-4 transition-colors"
                                    >
                                        Cancel
                                    </button>
                                    <button
                                        onClick={handleGenerateToken}
                                        disabled={!newTokenName}
                                        className="bg-cyan-500 hover:bg-cyan-600 disabled:opacity-50 disabled:cursor-not-allowed text-white font-medium py-2 px-4 rounded-lg transition-colors"
                                    >
                                        Generate
                                    </button>
                                </div>
                            </div>
                        )}
                    </div>
                </div>
            )}
        </div>
    );
};

export default ProfileTokens;
