import { useState } from "react";
import { Dialog, DialogPanel, DialogTitle } from "@headlessui/react";
import { UserGroup, GroupMember } from "../types";
import {
  useGetGroupMembersQuery,
  useAddUserToGroupMutation,
  useRemoveUserFromGroupMutation,
} from "../store/api/groups";
import { useGetUsersQuery } from "../store/api/users";
import ConfirmationModal from "./ConfirmationModal";

export const GroupMembersModal = ({
  group,
  onClose,
  isOpen,
}: {
  group: UserGroup | null;
  onClose: () => void;
  isOpen: boolean;
}) => {
  const [search, setSearch] = useState("");
  const [message, setMessage] = useState("");
  const [actionError, setActionError] = useState("");
  const [removing, setRemoving] = useState<GroupMember | null>(null);
  const {
    data: members = [],
    isLoading,
    error,
    refetch,
  } = useGetGroupMembersQuery(group?.id || "", { skip: !group || !isOpen });
  const {
    data: users = [],
    isLoading: usersLoading,
    error: usersError,
    refetch: refetchUsers,
  } = useGetUsersQuery(undefined, { skip: !isOpen });
  const [addUser, { isLoading: adding }] = useAddUserToGroupMutation();
  const [removeUser, { isLoading: deleting }] =
    useRemoveUserFromGroupMutation();
  const available = users.filter(
    (user) =>
      !members.some((m) => m.id === user.id) &&
      `${user.name} ${user.email}`.toLowerCase().includes(search.toLowerCase()),
  );

  const add = async (userId: string, name: string) => {
    if (!group) return;
    setActionError("");
    setMessage("");
    try {
      await addUser({ groupId: group.id, userId }).unwrap();
      setMessage(`${name} added to ${group.name}.`);
    } catch (err) {
      setActionError(
        (err as { data?: { error?: string } })?.data?.error ||
          "Could not add member. Try again.",
      );
    }
  };

  const close = () => {
    if (adding || deleting) return;
    setSearch("");
    setMessage("");
    setActionError("");
    onClose();
  };

  return (
    <Dialog
      open={isOpen && Boolean(group)}
      onClose={close}
      className="fixed inset-0 z-50"
    >
      <div className="fixed inset-0 bg-slate-950/80" aria-hidden="true" />
      <div className="fixed inset-0 flex items-center justify-center p-4">
        <DialogPanel className="admin-panel p-6 w-full max-w-4xl max-h-[90vh] overflow-y-auto">
          <div className="flex items-start justify-between gap-3 mb-4">
            <div>
              <DialogTitle className="text-xl font-semibold text-white">
                Members of {group?.name}
              </DialogTitle>
              <p className="text-sm text-slate-400 mt-2">
                Members inherit access to spaces granted to this group.
              </p>
            </div>
            <button
              className="btn-ghost"
              onClick={close}
              disabled={adding || deleting}
            >
              Close
            </button>
          </div>
          {message && (
            <p role="status" className="text-emerald-300 my-3">
              {message}
            </p>
          )}
          {actionError && (
            <p role="alert" className="text-red-300 my-3">
              {actionError}
            </p>
          )}
          {isLoading ? (
            <p role="status">Loading members…</p>
          ) : error ? (
            <div role="alert">
              <p>Could not load members.</p>
              <button className="btn-ghost mt-3" onClick={() => refetch()}>
                Retry
              </button>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <section>
                <h3 className="font-semibold mb-4">
                  Current members ({members.length})
                </h3>
                {members.length === 0 ? (
                  <p className="text-slate-400">No members yet.</p>
                ) : (
                  <ul className="space-y-3">
                    {members.map((member) => (
                      <li
                        key={member.id}
                        className="border border-slate-700 p-3 rounded-lg flex flex-wrap items-center justify-between gap-3"
                      >
                        <div className="min-w-0 break-all">
                          <p className="text-white">{member.name}</p>
                          <p className="text-sm text-slate-400">
                            {member.email}
                          </p>
                        </div>
                        <button
                          className="btn-ghost text-red-300"
                          disabled={adding || deleting}
                          aria-label={`Remove ${member.name}`}
                          onClick={() => setRemoving(member)}
                        >
                          Remove
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
              </section>
              <section>
                <h3 className="font-semibold mb-4">Add members</h3>
                <label className="block text-sm mb-4">
                  Search users
                  <input
                    className="admin-input mt-2"
                    type="search"
                    placeholder="Name or email"
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                  />
                </label>
                {usersLoading ? (
                  <p role="status">Loading users…</p>
                ) : usersError ? (
                  <div role="alert">
                    <p>Could not load users.</p>
                    <button
                      className="btn-ghost mt-3"
                      onClick={() => refetchUsers()}
                    >
                      Retry
                    </button>
                  </div>
                ) : available.length === 0 ? (
                  <p className="text-slate-400">
                    No matching users available to add.
                  </p>
                ) : (
                  <ul className="space-y-3 max-h-96 overflow-y-auto">
                    {available.map((user) => (
                      <li
                        key={user.id}
                        className="border border-slate-700 p-3 rounded-lg flex flex-wrap items-center justify-between gap-3"
                      >
                        <div className="min-w-0 break-all">
                          <p className="text-white">{user.name}</p>
                          <p className="text-sm text-slate-400">{user.email}</p>
                        </div>
                        <button
                          className="btn-ghost"
                          disabled={adding || deleting}
                          aria-label={`Add ${user.name}`}
                          onClick={() => add(user.id, user.name)}
                        >
                          Add
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
              </section>
            </div>
          )}
          <ConfirmationModal
            isOpen={Boolean(removing)}
            onClose={() => setRemoving(null)}
            title="Remove group member"
            message={
              <>
                Remove <strong>{removing?.name}</strong> from {group?.name}?
                They will lose access provided by this group.
              </>
            }
            confirmText="Remove member"
            isDestructive
            onConfirm={async () => {
              if (group && removing) {
                await removeUser({
                  groupId: group.id,
                  userId: removing.id,
                }).unwrap();
                setMessage(`${removing.name} removed.`);
              }
            }}
          />
        </DialogPanel>
      </div>
    </Dialog>
  );
};
