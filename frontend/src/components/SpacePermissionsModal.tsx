import React, { useState } from "react";
import { Space } from "../types";
import { useGetSpacePermissionsQuery, useAddSpacePermissionMutation, useRemoveSpacePermissionMutation } from "../store/api/spaces";
import { useGetGroupsQuery } from "../store/api/groups";
import { useGetUsersQuery } from "../store/api/users";
import { X, Trash2, CheckCircle, Search, User, Users, Shield } from "lucide-react";

interface SpacePermissionsModalProps {
  space: Space | null;
  onClose: () => void;
  isOpen: boolean;
}

export const SpacePermissionsModal: React.FC<SpacePermissionsModalProps> = ({
  space,
  onClose,
  isOpen,
}) => {
  const [activeTab, setActiveTab] = useState<"groups" | "users">("groups");
  const [searchTerm, setSearchTerm] = useState("");

  const { data: permissionsData, isLoading: isLoadingPermissions } = useGetSpacePermissionsQuery(space?.id || "", {
    skip: !space,
  });
  const permissions = permissionsData || [];

  const { data: groupsData } = useGetGroupsQuery();
  const groups = groupsData || [];

  const { data: usersData } = useGetUsersQuery();
  const users = usersData || [];

  const [addPermission] = useAddSpacePermissionMutation();
  const [removePermission] = useRemoveSpacePermissionMutation();

  if (!isOpen || !space) return null;

  const existingGroupIds = new Set(permissions.filter(p => p.groupId).map(p => p.groupId));
  const existingUserIds = new Set(permissions.filter(p => p.userId).map(p => p.userId));

  // Filter available items
  const availableGroups = groups.filter(
    (g) => !existingGroupIds.has(g.id) && g.name.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const availableUsers = users.filter(
    (u) =>
      !existingUserIds.has(u.id) &&
      (u.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
       u.email.toLowerCase().includes(searchTerm.toLowerCase()))
  );

  const handleAddGroup = async (groupId: string) => {
    try {
      await addPermission({ spaceId: space.id, groupId }).unwrap();
    } catch (error) {
      console.error("Failed to add group permission:", error);
    }
  };

  const handleAddUser = async (userId: string) => {
    try {
      await addPermission({ spaceId: space.id, userId }).unwrap();
    } catch (error) {
      console.error("Failed to add user permission:", error);
    }
  };

  const handleRemove = async (type: "group" | "user", id: string) => {
    try {
      if (type === "group") {
        await removePermission({ spaceId: space.id, groupId: id }).unwrap();
      } else {
        await removePermission({ spaceId: space.id, userId: id }).unwrap();
      }
    } catch (error) {
      console.error("Failed to remove permission:", error);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm transition-opacity duration-300">
      <div
        className="w-full max-w-3xl bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl overflow-hidden transform transition-all duration-300 scale-100 flex flex-col max-h-[85vh]"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800 bg-slate-900/50 flex-shrink-0">
          <div>
            <h2 className="text-xl font-semibold text-white flex items-center gap-2">
              <Shield className="w-5 h-5 text-cyan-400" />
              Space Permissions
            </h2>
            <p className="text-sm text-slate-400 mt-1">
              Manage access for: <span className="text-white font-medium">{space.name}</span>
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
          {/* Left: Current Access */}
          <div className="w-1/2 border-r border-slate-800 flex flex-col">
            <div className="p-4 border-b border-slate-800 bg-slate-900/30">
              <h3 className="text-sm font-medium text-slate-300">Current Access ({permissions.length})</h3>
              <p className="text-xs text-slate-500 mt-1">Users and groups with access to this space</p>
            </div>
            <div className="flex-1 overflow-y-auto p-4 space-y-2">
              {isLoadingPermissions ? (
                <div className="text-center py-4 text-slate-500">Loading permissions...</div>
              ) : permissions.length === 0 ? (
                <div className="text-center py-8 text-slate-500 italic">No permissions assigned (Private)</div>
              ) : (
                permissions.map((p) => (
                  <div key={p.id} className="flex items-center justify-between p-3 rounded-xl bg-slate-800/50 border border-slate-700/50 group hover:border-slate-600 transition-all">
                    <div className="flex items-center gap-3">
                      <div className={`w-8 h-8 rounded-full flex items-center justify-center ${p.groupId ? "bg-cyan-500/10 text-cyan-400" : "bg-emerald-500/10 text-emerald-400"}`}>
                        {p.groupId ? <Users className="w-4 h-4" /> : <User className="w-4 h-4" />}
                      </div>
                      <div>
                        {p.groupId ? (
                           <>
                              <div className="text-sm font-medium text-white">{p.groupName || `Group ${p.groupId}`}</div>
                              <div className="text-xs text-slate-400">Group Access</div>
                           </>
                        ) : (
                           <>
                              <div className="text-sm font-medium text-white">{p.userEmail || `User ${p.userId}`}</div>
                              <div className="text-xs text-slate-400">Individual Access</div>
                           </>
                        )}
                      </div>
                    </div>
                    <button
                      onClick={() => handleRemove(p.groupId ? "group" : "user", p.groupId || p.userId!)}
                      className="p-1.5 rounded-lg text-slate-400 hover:text-red-400 hover:bg-red-400/10 transition-colors"
                      title="Remove access"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                ))
              )}
            </div>
          </div>

          {/* Right: Add Access */}
          <div className="w-1/2 flex flex-col bg-slate-950/30">
            <div className="p-4 border-b border-slate-800 bg-slate-900/30">
              <h3 className="text-sm font-medium text-slate-300 mb-3">Grant Access</h3>

              {/* Type Toggle */}
              <div className="flex p-1 bg-slate-900 rounded-lg border border-slate-800 mb-3">
                <button
                  onClick={() => setActiveTab("groups")}
                  className={`flex-1 flex items-center justify-center gap-2 py-1.5 text-xs font-medium rounded-md transition-all ${activeTab === "groups" ? "bg-slate-800 text-white shadow-sm" : "text-slate-400 hover:text-slate-300"}`}
                >
                  <Users className="w-3.5 h-3.5" /> Groups
                </button>
                <button
                  onClick={() => setActiveTab("users")}
                   className={`flex-1 flex items-center justify-center gap-2 py-1.5 text-xs font-medium rounded-md transition-all ${activeTab === "users" ? "bg-slate-800 text-white shadow-sm" : "text-slate-400 hover:text-slate-300"}`}
                >
                  <User className="w-3.5 h-3.5" /> Users
                </button>
              </div>

              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-500" />
                <input
                  type="text"
                  placeholder={activeTab === "groups" ? "Search groups..." : "Search users..."}
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="w-full pl-9 pr-4 py-2 bg-slate-950 border border-slate-800 rounded-lg text-sm text-slate-200 focus:outline-none focus:border-cyan-500/50 placeholder:text-slate-600"
                />
              </div>
            </div>

            <div className="flex-1 overflow-y-auto p-4 space-y-2">
               {activeTab === "groups" ? (
                  availableGroups.length === 0 ? (
                    <div className="text-center py-8 text-slate-500 italic">No available groups</div>
                  ) : (
                    availableGroups.map((g) => (
                      <div key={g.id} className="flex items-center justify-between p-3 rounded-xl bg-slate-800/30 border border-slate-800 hover:border-slate-700 hover:bg-slate-800/50 transition-all">
                        <div className="overflow-hidden mr-2">
                          <div className="text-sm font-medium text-slate-200 truncate">{g.name}</div>
                          <div className="text-xs text-slate-500 truncate">{g.description || "No description"}</div>
                        </div>
                        <button
                          onClick={() => handleAddGroup(g.id)}
                          className="p-1.5 rounded-lg text-cyan-400 hover:bg-cyan-400/10 transition-colors shrink-0"
                          title="Grant Group Access"
                        >
                          <CheckCircle className="w-4 h-4" />
                        </button>
                      </div>
                    ))
                  )
               ) : (
                  availableUsers.length === 0 ? (
                    <div className="text-center py-8 text-slate-500 italic">No available users</div>
                  ) : (
                    availableUsers.map((u) => (
                      <div key={u.id} className="flex items-center justify-between p-3 rounded-xl bg-slate-800/30 border border-slate-800 hover:border-slate-700 hover:bg-slate-800/50 transition-all">
                         <div className="overflow-hidden mr-2">
                          <div className="text-sm font-medium text-slate-200 truncate">{u.name}</div>
                          <div className="text-xs text-slate-500 truncate">{u.email}</div>
                        </div>
                        <button
                          onClick={() => handleAddUser(u.id)}
                          className="p-1.5 rounded-lg text-cyan-400 hover:bg-cyan-400/10 transition-colors shrink-0"
                          title="Grant User Access"
                        >
                          <CheckCircle className="w-4 h-4" />
                        </button>
                      </div>
                    ))
                  )
               )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
