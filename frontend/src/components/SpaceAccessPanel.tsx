import { useState } from "react";
import { Space } from "../types";
import {
  useGetSpacePermissionsQuery,
  useAddSpacePermissionMutation,
  useRemoveSpacePermissionMutation,
} from "../store/api/spaces";
import { useGetGroupsQuery } from "../store/api/groups";
import { useGetUsersQuery } from "../store/api/users";
import { useNotificationContext } from "../hooks/useNotification";
import ConfirmationModal from "./ConfirmationModal";

export const SpaceAccessPanel = ({ space }: { space: Space }) => {
  const {
    data: permissions = [],
    isLoading,
    error,
    refetch,
  } = useGetSpacePermissionsQuery(space.id, {
    skip: space.name === "Default Space",
  });
  const {
    data: groups = [],
    error: groupsError,
    isLoading: groupsLoading,
    refetch: refetchGroups,
  } = useGetGroupsQuery();
  const {
    data: users = [],
    error: usersError,
    isLoading: usersLoading,
    refetch: refetchUsers,
  } = useGetUsersQuery();
  const [addPermission, { isLoading: adding }] =
    useAddSpacePermissionMutation();
  const [removePermission] = useRemoveSpacePermissionMutation();
  const [search, setSearch] = useState("");
  const [type, setType] = useState<"groups" | "users">("groups");
  const [actionError, setActionError] = useState("");
  const [removing, setRemoving] = useState<{
    groupId?: string;
    userId?: string;
    name: string;
  } | null>(null);
  const { showNotification } = useNotificationContext();

  const grant = async (id: string, name: string) => {
    setActionError("");
    try {
      await addPermission({
        spaceId: space.id,
        ...(type === "groups" ? { groupId: id } : { userId: id }),
      }).unwrap();
      showNotification("success", `Access granted to ${name}`);
    } catch (err) {
      setActionError(
        (err as { data?: { error?: string } })?.data?.error ||
          "Could not grant access. Please try again.",
      );
    }
  };

  if (space.name === "Default Space")
    return (
      <section className="admin-panel p-6">
        <h2 className="text-lg font-semibold text-white">Public access</h2>
        <p className="mt-3 text-slate-300">
          The Default Space is public. Individual access grants are not
          required.
        </p>
      </section>
    );
  if (isLoading) return <p role="status">Loading access…</p>;
  if (error)
    return (
      <div role="alert" className="admin-panel p-6">
        <p>Could not load access. No access changes have been made.</p>
        <button className="btn-ghost mt-3" onClick={() => refetch()}>
          Retry
        </button>
      </div>
    );

  const available = (
    type === "groups"
      ? groups.map((g) => ({ id: g.id, name: g.name, detail: g.description }))
      : users.map((u) => ({ id: u.id, name: u.name, detail: u.email }))
  ).filter(
    (item) =>
      !permissions.some(
        (p) => (type === "groups" ? p.groupId : p.userId) === item.id,
      ) &&
      `${item.name} ${item.detail}`
        .toLowerCase()
        .includes(search.toLowerCase()),
  );

  return (
    <div className="space-y-6">
      <p className="admin-panel p-4 text-sm text-slate-300">
        Group members inherit access from their groups. Direct grants apply to
        one person. This list shows explicit grants, not every person with
        effective access. Administrators retain access.
      </p>
      <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
        <section className="admin-panel p-5">
          <h2 className="text-lg font-semibold text-white mb-4">
            Current access ({permissions.length}{" "}
            {permissions.length === 1 ? "grant" : "grants"})
          </h2>
          {permissions.length === 0 ? (
            <p className="text-slate-400">
              No access granted. Only administrators can access this space.
            </p>
          ) : (
            <ul className="space-y-3">
              {permissions.map((permission) => {
                const name = permission.groupId
                  ? permission.groupName ||
                    groups.find((g) => g.id === permission.groupId)?.name ||
                    "Unknown group"
                  : permission.userEmail ||
                    users.find((u) => u.id === permission.userId)?.email ||
                    "Unknown user";
                return (
                  <li
                    key={permission.id}
                    className="border border-slate-700 rounded-lg p-3 flex flex-wrap items-center justify-between gap-3"
                  >
                    <div className="min-w-0 break-words">
                      <p className="font-medium text-white">{name}</p>
                      <p className="text-sm text-slate-400">
                        {permission.groupId
                          ? "Through group membership"
                          : "Direct user access"}
                      </p>
                    </div>
                    <button
                      className="btn-ghost text-red-300"
                      aria-label={`Remove access for ${name}`}
                      disabled={adding}
                      onClick={() =>
                        setRemoving({
                          groupId: permission.groupId,
                          userId: permission.userId,
                          name,
                        })
                      }
                    >
                      Remove
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
        </section>
        <section className="admin-panel p-5">
          <h2 className="text-lg font-semibold text-white mb-4">
            Grant access
          </h2>
          <label className="block text-sm mb-4">
            Grant access to
            <select
              className="admin-input mt-2"
              value={type}
              disabled={adding}
              onChange={(e) => {
                setType(e.target.value as "groups" | "users");
                setSearch("");
              }}
            >
              <option value="groups">Groups</option>
              <option value="users">Individual users</option>
            </select>
          </label>
          <label className="block text-sm mb-4">
            Search {type}
            <input
              className="admin-input mt-2"
              type="search"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search by name or email"
            />
          </label>
          {actionError && (
            <p role="alert" className="text-red-300 mb-4">
              {actionError}
            </p>
          )}
          {(type === "groups" ? groupsLoading : usersLoading) ? (
            <p role="status">Loading {type}…</p>
          ) : (type === "groups" ? groupsError : usersError) ? (
            <div role="alert">
              <p>Could not load {type}.</p>
              <button
                className="btn-ghost mt-3"
                onClick={() =>
                  type === "groups" ? refetchGroups() : refetchUsers()
                }
              >
                Retry
              </button>
            </div>
          ) : available.length === 0 ? (
            <p className="text-slate-400">
              No matching {type} available to add.
            </p>
          ) : (
            <ul className="space-y-3 max-h-96 overflow-y-auto">
              {available.map((item) => (
                <li
                  key={item.id}
                  className="border border-slate-700 rounded-lg p-3 flex flex-wrap items-center justify-between gap-3"
                >
                  <div className="min-w-0 break-words">
                    <p className="text-white">{item.name}</p>
                    <p className="text-sm text-slate-400">{item.detail}</p>
                  </div>
                  <button
                    className="btn-ghost"
                    aria-label={`Grant access to ${item.name}`}
                    disabled={adding}
                    onClick={() => grant(item.id, item.name)}
                  >
                    Grant access
                  </button>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
      <ConfirmationModal
        isOpen={Boolean(removing)}
        onClose={() => setRemoving(null)}
        title="Remove access"
        message={
          <>
            Remove the grant for <strong>{removing?.name}</strong>? Other group
            memberships or direct grants may still provide access.
          </>
        }
        confirmText="Remove access"
        isDestructive
        onConfirm={async () => {
          if (removing) {
            await removePermission({
              spaceId: space.id,
              ...(removing.groupId
                ? { groupId: removing.groupId }
                : { userId: removing.userId }),
            }).unwrap();
            showNotification("success", "Access grant removed");
          }
        }}
      />
    </div>
  );
};
