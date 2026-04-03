import React, { useState } from "react";
import {
  useGetSpacesQuery,
  useCreateSpaceMutation,
  useDeleteSpaceMutation,
} from "../store/api/spaces";
import { Space } from "../types";
import { Trash2, Plus, Box, Shield } from "lucide-react";
import { SpacePermissionsModal } from "./SpacePermissionsModal";

const SpaceManagement: React.FC = () => {
  const { data: spaces, isLoading, error } = useGetSpacesQuery();
  const [createSpace] = useCreateSpaceMutation();
  const [deleteSpace] = useDeleteSpaceMutation();
  const [newSpaceName, setNewSpaceName] = useState("");
  const [newSpaceDescription, setNewSpaceDescription] = useState("");

  const [selectedSpace, setSelectedSpace] = useState<Space | null>(null);
  const [isPermissionsModalOpen, setIsPermissionsModalOpen] = useState(false);

  const handleCreateSpace = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newSpaceName) return;
    try {
      await createSpace({
        name: newSpaceName,
        description: newSpaceDescription,
      }).unwrap();
      setNewSpaceName("");
      setNewSpaceDescription("");
    } catch (err) {
      console.error("Failed to create space:", err);
    }
  };

  const handleDeleteSpace = async (id: string) => {
    if (confirm("Are you sure? All resources in this space will be deleted.")) {
      try {
        await deleteSpace(id).unwrap();
      } catch (err) {
        console.error("Failed to delete space:", err);
      }
    }
  };

  const handlePermissions = (space: Space) => {
    setSelectedSpace(space);
    setIsPermissionsModalOpen(true);
  };

  return (
    <div className="space-y-8 mb-12">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-3xl font-bold text-white tracking-tight flex items-center gap-3">
            <Box className="w-8 h-8 text-brand-queued" />
            Spaces
          </h2>
          <p className="text-slate-400 mt-1">Manage organizational units and resource boundaries</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Create Form */}
        <div className="lg:col-span-1">
          <div className="glass-panel p-6 rounded-xl border border-slate-800/50">
            <h3 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
              <Plus className="w-4 h-4 text-brand-queued" />
              New Space
            </h3>
            <form onSubmit={handleCreateSpace} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-slate-400 uppercase tracking-wider mb-1.5">
                  Space Name
                </label>
                <input
                  type="text"
                  placeholder="e.g. Engineering, Marketing"
                  value={newSpaceName}
                  onChange={(e) => setNewSpaceName(e.target.value)}
                  className="w-full rounded-lg bg-slate-950/50 border border-slate-800 px-4 py-2.5 text-sm text-slate-200 focus:outline-none focus:border-brand-queued transition-colors"
                  required
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-400 uppercase tracking-wider mb-1.5">
                  Description
                </label>
                <textarea
                  placeholder="Optional details..."
                  value={newSpaceDescription}
                  onChange={(e) => setNewSpaceDescription(e.target.value)}
                  className="w-full rounded-lg bg-slate-950/50 border border-slate-800 px-4 py-2.5 text-sm text-slate-200 focus:outline-none focus:border-brand-queued transition-colors h-24 resize-none"
                />
              </div>
              <button
                type="submit"
                className="btn-primary w-full py-3"
              >
                Create Space
              </button>
            </form>
          </div>
        </div>

        {/* Spaces List */}
        <div className="lg:col-span-2">
          <div className="glass-panel rounded-xl overflow-hidden border border-slate-800/50 shadow-2xl">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-slate-800/50 bg-slate-900/50">
                    <th className="px-6 py-4 font-semibold text-slate-300 uppercase tracking-wider text-[11px]">Name</th>
                    <th className="px-6 py-4 font-semibold text-slate-300 uppercase tracking-wider text-[11px]">Description</th>
                    <th className="px-6 py-4 font-semibold text-slate-300 uppercase tracking-wider text-[11px] text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800/50 text-slate-400">
                  {isLoading ? (
                    <tr>
                      <td colSpan={3} className="px-6 py-12 text-center">
                        <div className="flex flex-col items-center gap-2">
                          <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-brand-queued"></div>
                          <span className="text-slate-500 italic">Loading spaces...</span>
                        </div>
                      </td>
                    </tr>
                  ) : error ? (
                    <tr>
                      <td colSpan={3} className="px-6 py-12 text-center text-brand-busy italic">
                        Error loading spaces.
                      </td>
                    </tr>
                  ) : spaces && spaces.length > 0 ? (
                    spaces.map((space: Space) => (
                      <tr key={space.id} className="group hover:bg-white/[0.02] transition-colors">
                        <td className="px-6 py-4 whitespace-nowrap align-middle">
                          <span className="font-semibold text-white">{space.name}</span>
                        </td>
                        <td className="px-6 py-4 align-middle">
                          <span className="text-slate-400 line-clamp-1">{space.description || "—"}</span>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap align-middle text-right">
                          <div className="flex items-center justify-end gap-2">
                            {space.name !== "Default Space" ? (
                              <>
                                <button
                                  onClick={() => handlePermissions(space)}
                                  className="p-2 rounded-lg bg-cyan-500/10 text-cyan-400 hover:bg-cyan-500/20 border border-cyan-500/20 transition-all shadow-sm"
                                  title="Manage Permissions"
                                >
                                  <Shield className="w-4 h-4" />
                                </button>
                                <button
                                  onClick={() => handleDeleteSpace(space.id)}
                                  className="p-2 rounded-lg bg-brand-busy/10 text-brand-busy hover:bg-brand-busy/20 border border-brand-busy/20 transition-all shadow-sm"
                                  title="Delete Space"
                                >
                                  <Trash2 className="w-4 h-4" />
                                </button>
                              </>
                            ) : (
                              <span className="text-xs font-medium text-slate-600 bg-slate-800/40 px-2 py-1 rounded border border-slate-700/50 uppercase tracking-tighter">
                                Public
                              </span>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={3} className="px-6 py-12 text-center text-slate-500 italic">
                        No spaces found.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>

      <div className="h-[1px] w-full bg-slate-800/50 my-12 shadow-[0_0_15px_rgba(0,0,0,0.5)]"></div>

      <SpacePermissionsModal
        isOpen={isPermissionsModalOpen}
        onClose={() => setIsPermissionsModalOpen(false)}
        space={selectedSpace}
      />
    </div>
  );
};

export default SpaceManagement;
