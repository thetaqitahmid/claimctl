import { useState } from "react";
import { Users, Plus, Edit, Trash2, UserPlus, Info } from "lucide-react";
import { useGetGroupsQuery, useDeleteGroupMutation } from "../store/api/groups";
import { UserGroup } from "../types";
import { GroupModal } from "./GroupModal";
import { GroupMembersModal } from "./GroupMembersModal";

const GroupManagement = () => {
  const { data: groupsData, isLoading, error } = useGetGroupsQuery();
  const groups = groupsData || [];
  const [deleteGroup] = useDeleteGroupMutation();

  const [selectedGroup, setSelectedGroup] = useState<UserGroup | undefined>(undefined);
  const [isGroupModalOpen, setIsGroupModalOpen] = useState(false);
  const [isMembersModalOpen, setIsMembersModalOpen] = useState(false);

  // For editing/members modals, we need to know which group is active
  const [activeGroupId, setActiveGroupId] = useState<string | null>(null);

  const activeGroup = groups.find(g => g.id === activeGroupId) || null;

  const handleCreate = () => {
    setSelectedGroup(undefined);
    setIsGroupModalOpen(true);
  };

  const handleEdit = (group: UserGroup) => {
    setSelectedGroup(group);
    setIsGroupModalOpen(true);
  };

  const handleManageMembers = (group: UserGroup) => {
    setActiveGroupId(group.id);
    setIsMembersModalOpen(true);
  };

  const handleDelete = async (group: UserGroup) => {
    if (window.confirm(`Are you sure you want to delete group "${group.name}"?`)) {
      try {
        await deleteGroup(group.id).unwrap();
      } catch (err) {
        console.error("Failed to delete group:", err);
      }
    }
  };

  const closeModal = () => {
    setIsGroupModalOpen(false);
    setSelectedGroup(undefined);
  };

  return (
    <div className="mb-12">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-3xl font-bold text-white tracking-tight flex items-center gap-3">
            <Users className="w-8 h-8 text-brand-queued" />
            User Groups
          </h2>
          <p className="text-slate-400 mt-1">Manage user groups and access control</p>
        </div>
        <button
          onClick={handleCreate}
          className="btn-primary"
        >
          <Plus className="w-4 h-4" />
          Create Group
        </button>
      </div>

      {/* Error State */}
      {error && (
        <div className="glass-panel p-4 rounded-xl border-brand-busy/20 bg-brand-busy/5 mb-6 text-center">
          <p className="text-brand-busy font-medium">Failed to fetch groups</p>
        </div>
      )}

      {/* Loading State */}
      {isLoading ? (
        <div className="flex flex-col items-center justify-center p-20 glass-panel rounded-xl">
           <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-brand-queued mb-4"></div>
           <p className="text-slate-400 font-medium">Loading groups...</p>
        </div>
      ) : groups.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 glass-panel rounded-xl border-dashed border-2 border-slate-800">
          <Users className="w-12 h-12 text-slate-700 mb-4" />
          <h3 className="text-lg font-medium text-white">No groups found</h3>
          <p className="text-slate-400 mb-4">Create groups to manage permissions effectively</p>
          <button onClick={handleCreate} className="btn-secondary">
            Create First Group
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {groups.map((group) => (
            <div key={group.id} className="glass-panel p-5 rounded-xl hover:bg-slate-800/50 transition-all group">
              <div className="flex justify-between items-start mb-3">
                <div className="p-2 rounded-lg bg-cyan-500/10 text-cyan-400">
                  <Users className="w-6 h-6" />
                </div>
                <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button
                    onClick={() => handleManageMembers(group)}
                    className="p-1.5 rounded-lg text-slate-400 hover:text-emerald-400 hover:bg-emerald-400/10 transition-colors"
                    title="Manage Members"
                  >
                    <UserPlus className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => handleEdit(group)}
                    className="p-1.5 rounded-lg text-slate-400 hover:text-cyan-400 hover:bg-cyan-400/10 transition-colors"
                    title="Edit Group"
                  >
                    <Edit className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => handleDelete(group)}
                    className="p-1.5 rounded-lg text-slate-400 hover:text-red-400 hover:bg-red-400/10 transition-colors"
                    title="Delete Group"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>

              <h3 className="text-lg font-semibold text-white mb-1">{group.name}</h3>
              <p className="text-sm text-slate-400 line-clamp-2 h-10 mb-4">
                {group.description || <span className="italic text-slate-600">No description</span>}
              </p>

              <div className="flex items-center text-xs text-slate-500 gap-4 pt-3 border-t border-slate-800">
                <div className="flex items-center gap-1">
                  <Info className="w-3 h-3" />
                  ID: {group.id}
                </div>
                {/* We could show member count here if the API returned it directly */}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Modals */}
      <GroupModal
        isOpen={isGroupModalOpen}
        onClose={closeModal}
        group={selectedGroup}
      />

      <GroupMembersModal
        isOpen={isMembersModalOpen}
        onClose={() => setIsMembersModalOpen(false)}
        group={activeGroup}
      />
    </div>
  );
};

export default GroupManagement;
