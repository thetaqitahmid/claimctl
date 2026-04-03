import React, { useState } from "react";
import { UserGroup } from "../types";
import { useGetGroupMembersQuery, useAddUserToGroupMutation, useRemoveUserFromGroupMutation } from "../store/api/groups";
import { useGetUsersQuery } from "../store/api/users";
import { X, Trash2, UserPlus, Search, User } from "lucide-react";

interface GroupMembersModalProps {
  group: UserGroup | null;
  onClose: () => void;
  isOpen: boolean;
}

interface FlatGroupMember {
  id: string;
  name: string;
  email: string;
}

export const GroupMembersModal: React.FC<GroupMembersModalProps> = ({
  group,
  onClose,
  isOpen,
}) => {
  const [searchTerm, setSearchTerm] = useState("");
  const { data: membersData, isLoading: isLoadingMembers } = useGetGroupMembersQuery(group?.id || "", {
    skip: !group,
  });
  const members = membersData || [];

  const { data: allUsersData } = useGetUsersQuery();
  const allUsers = allUsersData || [];

  const [addUser] = useAddUserToGroupMutation();
  const [removeUser] = useRemoveUserFromGroupMutation();

  if (!isOpen || !group) return null;

  const existingMemberIds = new Set(members.map((m) => (m as unknown as FlatGroupMember).id));

  // Filter users to show only those not in the group and matching search
  const availableUsers = allUsers.filter(
    (user) =>
      !existingMemberIds.has(user.id) &&
      (user.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
       user.email.toLowerCase().includes(searchTerm.toLowerCase()))
  );

  const handleAddUser = async (userId: string) => {
    try {
      await addUser({ groupId: group.id, userId }).unwrap();
    } catch (error) {
      console.error("Failed to add user:", error);
    }
  };

  const handleRemoveUser = async (userId: string) => {
    try {
      await removeUser({ groupId: group.id, userId }).unwrap();
    } catch (error) {
      console.error("Failed to remove user:", error);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm transition-opacity duration-300">
      <div
        className="w-full max-w-2xl bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl overflow-hidden transform transition-all duration-300 scale-100 flex flex-col max-h-[80vh]"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800 bg-slate-900/50 flex-shrink-0">
          <div>
            <h2 className="text-xl font-semibold text-white flex items-center gap-2">
              <UserPlus className="w-5 h-5 text-emerald-400" />
              Manage Members
            </h2>
            <p className="text-sm text-slate-400 mt-1">
              Group: <span className="text-white font-medium">{group.name}</span>
            </p>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white transition-colors p-1 rounded-lg hover:bg-slate-800"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="flex flex-1 overflow-hidden">
          {/* Left: Current Members */}
          <div className="w-1/2 border-r border-slate-800 flex flex-col">
            <div className="p-4 border-b border-slate-800 bg-slate-900/30">
              <h3 className="text-sm font-medium text-slate-300">Current Members ({members.length})</h3>
            </div>
            <div className="flex-1 overflow-y-auto p-4 space-y-2">
              {isLoadingMembers ? (
                <div className="text-center py-4 text-slate-500">Loading members...</div>
              ) : members.length === 0 ? (
                <div className="text-center py-8 text-slate-500 italic">No members in this group</div>
              ) : (
                members.map((item) => {
                  const member = item as unknown as FlatGroupMember;
                  return (
                  <div key={member.id} className="flex items-center justify-between p-3 rounded-xl bg-slate-800/50 border border-slate-700/50 group hover:border-slate-600 transition-all">
                    <div className="flex items-center gap-3">
                      <div className="w-8 h-8 rounded-full bg-cyan-500/10 flex items-center justify-center text-cyan-400">
                        <User className="w-4 h-4" />
                      </div>
                      <div>
                        <div className="text-sm font-medium text-white">{member.name || `User ${member.id}`}</div>
                        <div className="text-xs text-slate-400">{member.email}</div>
                      </div>
                    </div>
                    <button
                      onClick={() => handleRemoveUser(member.id)}
                      className="p-1.5 rounded-lg text-slate-400 hover:text-red-400 hover:bg-red-400/10 transition-colors"
                      title="Remove member"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                  );
                })
              )}
            </div>
          </div>

          {/* Right: Add Users */}
          <div className="w-1/2 flex flex-col bg-slate-950/30">
            <div className="p-4 border-b border-slate-800 bg-slate-900/30">
              <h3 className="text-sm font-medium text-slate-300 mb-3">Add Users</h3>
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-500" />
                <input
                  type="text"
                  placeholder="Search users..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="w-full pl-9 pr-4 py-2 bg-slate-950 border border-slate-800 rounded-lg text-sm text-slate-200 focus:outline-none focus:border-cyan-500/50 placeholder:text-slate-600"
                />
              </div>
            </div>
            <div className="flex-1 overflow-y-auto p-4 space-y-2">
              {availableUsers.length === 0 ? (
                <div className="text-center py-8 text-slate-500 italic">
                  {searchTerm ? "No matching users found" : "No available users to add"}
                </div>
              ) : (
                availableUsers.map((user) => (
                  <div key={user.id} className="flex items-center justify-between p-3 rounded-xl bg-slate-800/30 border border-slate-800 hover:border-slate-700 hover:bg-slate-800/50 transition-all">
                    <div className="overflow-hidden mr-2">
                      <div className="text-sm font-medium text-slate-200 truncate">{user.name}</div>
                      <div className="text-xs text-slate-500 truncate">{user.email}</div>
                    </div>
                    <button
                      onClick={() => handleAddUser(user.id)}
                      className="p-1.5 rounded-lg text-emerald-400 hover:bg-emerald-400/10 transition-colors shrink-0"
                      title="Add to group"
                    >
                      <UserPlus className="w-4 h-4" />
                    </button>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
