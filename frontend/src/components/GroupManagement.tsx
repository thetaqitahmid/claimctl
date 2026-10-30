import { useState } from "react";
import { Plus } from "lucide-react";
import { useGetGroupsQuery, useDeleteGroupMutation } from "../store/api/groups";
import { UserGroup } from "../types";
import { GroupModal } from "./GroupModal";
import { GroupMembersModal } from "./GroupMembersModal";
import ConfirmationModal from "./ConfirmationModal";
import { useNotificationContext } from "../hooks/useNotification";

const GroupManagement = () => {
  const { data: groups = [], isLoading, error, refetch } = useGetGroupsQuery();
  const [deleteGroup] = useDeleteGroupMutation();
  const [selectedGroup, setSelectedGroup] = useState<UserGroup>();
  const [isGroupModalOpen, setIsGroupModalOpen] = useState(false);
  const [activeGroupId, setActiveGroupId] = useState<string | null>(null);
  const [groupToDelete, setGroupToDelete] = useState<UserGroup | null>(null);
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(0);
  const { showNotification } = useNotificationContext();
  const filtered = groups.filter((group) =>
    `${group.name} ${group.description}`
      .toLowerCase()
      .includes(search.toLowerCase()),
  );
  const pageCount = Math.max(1, Math.ceil(filtered.length / 12));
  const currentPage = Math.min(page, pageCount - 1);

  const openCreate = () => {
    setSelectedGroup(undefined);
    setIsGroupModalOpen(true);
  };
  const handleDelete = async () => {
    if (!groupToDelete) return;
    await deleteGroup(groupToDelete.id).unwrap();
    showNotification("success", `Group ${groupToDelete.name} deleted`);
  };

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <p className="text-sm text-slate-400 mb-2">
            Administration / People and access
          </p>
          <h1 className="text-3xl font-semibold text-white">Groups</h1>
          <p className="text-slate-400 mt-2">
            Organize people into groups, then grant groups access to spaces.
          </p>
        </div>
        <button className="btn-primary" onClick={openCreate}>
          <Plus size={18} />
          Create group
        </button>
      </header>
      <label className="block max-w-lg text-sm text-slate-300">
        Search groups
        <input
          className="admin-input mt-2"
          type="search"
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            setPage(0);
          }}
          placeholder="Name or description"
        />
      </label>
      {isLoading ? (
        <p role="status">Loading groups…</p>
      ) : error ? (
        <div role="alert" className="admin-panel p-6">
          <p>Could not load groups.</p>
          <button className="btn-ghost mt-3" onClick={() => refetch()}>
            Retry
          </button>
        </div>
      ) : groups.length === 0 ? (
        <div className="admin-panel p-10 text-center">
          <h2 className="text-lg font-medium text-white">No groups yet</h2>
          <p className="text-slate-400 my-3">
            Create a group to manage access for a team.
          </p>
          <button className="btn-primary mx-auto" onClick={openCreate}>
            Create first group
          </button>
        </div>
      ) : filtered.length === 0 ? (
        <p className="admin-panel p-6">No groups match your search.</p>
      ) : (
        <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
          {filtered
            .slice(currentPage * 12, (currentPage + 1) * 12)
            .map((group) => (
              <article key={group.id} className="admin-panel p-5">
                <h2 className="text-lg font-semibold text-white">
                  {group.name}
                </h2>
                <p className="text-sm text-slate-400 mt-2 mb-5">
                  {group.description || "No description provided."}
                </p>
                <div className="flex flex-wrap gap-2">
                  <button
                    className="btn-ghost"
                    aria-label={`Manage members of ${group.name}`}
                    onClick={() => setActiveGroupId(group.id)}
                  >
                    Members
                  </button>
                  <button
                    className="btn-ghost"
                    aria-label={`Edit ${group.name}`}
                    onClick={() => {
                      setSelectedGroup(group);
                      setIsGroupModalOpen(true);
                    }}
                  >
                    Edit
                  </button>
                  <button
                    className="btn-ghost text-red-300"
                    aria-label={`Delete ${group.name}`}
                    onClick={() => setGroupToDelete(group)}
                  >
                    Delete
                  </button>
                </div>
              </article>
            ))}
        </div>
      )}
      {!error && filtered.length > 0 && (
        <nav
          aria-label="Group pagination"
          className="flex flex-wrap items-center gap-3 text-sm"
        >
          <p className="mr-auto">
            {filtered.length} groups · Page {currentPage + 1} of {pageCount}
          </p>
          <button
            className="btn-ghost"
            disabled={currentPage === 0}
            onClick={() => setPage(currentPage - 1)}
          >
            Previous
          </button>
          <button
            className="btn-ghost"
            disabled={currentPage === pageCount - 1}
            onClick={() => setPage(currentPage + 1)}
          >
            Next
          </button>
        </nav>
      )}
      <GroupModal
        group={selectedGroup}
        isOpen={isGroupModalOpen}
        onClose={() => setIsGroupModalOpen(false)}
      />
      <GroupMembersModal
        key={activeGroupId || "closed"}
        group={groups.find((g) => g.id === activeGroupId) || null}
        isOpen={Boolean(activeGroupId)}
        onClose={() => setActiveGroupId(null)}
      />
      <ConfirmationModal
        isOpen={Boolean(groupToDelete)}
        onClose={() => setGroupToDelete(null)}
        onConfirm={handleDelete}
        title="Delete group"
        message={
          <>
            Delete <strong>{groupToDelete?.name}</strong>? Members will lose
            access granted through this group. Their user accounts will not be
            deleted.
          </>
        }
        confirmText="Delete group"
        isDestructive
      />
    </div>
  );
};

export default GroupManagement;
