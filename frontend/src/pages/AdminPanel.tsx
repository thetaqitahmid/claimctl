import { useState } from "react";
import { User as UserIcon, Plus } from "lucide-react";
import {
  useGetUsersQuery,
  useDeleteUserMutation,
  useUpdateUserMutation,
} from "../store/api/users";
import { User as userPropType } from "../store/api/auth";
import { UserModal } from "../components/UserModal";
import UserTable from "../components/UserTable";
import ConfirmationModal from "../components/ConfirmationModal";
import { useNotificationContext } from "../hooks/useNotification";

const AdminPanel = () => {
  const [currentUser, setCurrentUser] = useState<userPropType | undefined>(
    undefined,
  );
  const [userModalOpen, setUserModalOpen] = useState<boolean>(false);
  const { data: usersData, error, isLoading, refetch } = useGetUsersQuery();
  const users = usersData || [];
  const [deleteUser] = useDeleteUserMutation();
  const [editUser] = useUpdateUserMutation();

  const [userToDelete, setUserToDelete] = useState<userPropType | null>(null);
  const [userToToggle, setUserToToggle] = useState<userPropType | null>(null);
  const { showNotification } = useNotificationContext();

  const onDeleteAction = (user: userPropType) => {
    setUserToDelete(user);
  };

  const confirmDeleteUser = async () => {
    if (userToDelete) {
      try {
        await deleteUser(userToDelete.id).unwrap();
        showNotification(
          "success",
          `User ${userToDelete.name} deleted successfully`,
        );
      } catch (err) {
        const error = err as { data?: { error?: string }; message?: string };
        console.error("Failed to delete user", userToDelete, error);
        showNotification(
          "error",
          `Failed to delete user: ${error?.data?.error || error.message || "Unknown error"}`,
        );
        throw err;
      }
      setUserToDelete(null);
    }
  };

  const handleUserEdit = (user: userPropType) => {
    setCurrentUser(user);
    setUserModalOpen(true);
  };

  const handleAddUser = () => {
    setCurrentUser(undefined);
    setUserModalOpen(true);
  };

  const handleUserStatus = async () => {
    if (!userToToggle) return;
    await editUser({
      id: userToToggle.id,
      status: userToToggle.status === "active" ? "inactive" : "active",
    }).unwrap();
    showNotification("success", "User status updated successfully");
  };

  return (
    <div>
      <p className="text-sm text-slate-400 mb-2">
        Administration / People and access
      </p>
      <section aria-label="User management">
        <div className="mb-8 flex flex-wrap gap-4 items-center justify-between">
          <div>
            <h1 className="text-3xl font-semibold text-white tracking-tight flex items-center gap-3">
              <UserIcon className="w-8 h-8 text-brand-queued" />
              Users
            </h1>
            <p className="text-slate-400 mt-1">
              Manage system accounts and permissions
            </p>
          </div>
          <button onClick={handleAddUser} className="btn-primary">
            <Plus className="w-4 h-4" />
            Add User
          </button>
        </div>

        {/* Error State */}
        {error && (
          <div role="alert" className="admin-panel p-6 mb-6">
            <p className="text-red-300 font-medium">Could not load users.</p>
            <button className="btn-ghost mt-3" onClick={() => refetch()}>
              Retry
            </button>
          </div>
        )}

        {/* Loading State */}
        {isLoading ? (
          <div className="flex flex-col items-center justify-center p-20 glass-panel rounded-xl">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-brand-queued mb-4"></div>
            <p className="text-slate-400 font-medium">Loading users...</p>
          </div>
        ) : !error && users.length > 0 ? (
          <UserTable
            data={users}
            onEdit={handleUserEdit}
            onToggleStatus={setUserToToggle}
            onDelete={onDeleteAction}
          />
        ) : null}

        {!isLoading && !error && users.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 glass-panel rounded-xl">
            <UserIcon className="w-12 h-12 text-slate-700 mb-4" />
            <h3 className="text-lg font-medium text-white">No users found</h3>
            <p className="text-slate-400 mb-4">
              Add an account to get started.
            </p>
            <button className="btn-primary" onClick={handleAddUser}>
              Add first user
            </button>
          </div>
        )}

        <UserModal
          user={currentUser}
          isOpen={userModalOpen}
          onClose={() => setUserModalOpen(false)}
        />

        <ConfirmationModal
          isOpen={Boolean(userToToggle)}
          onClose={() => setUserToToggle(null)}
          onConfirm={handleUserStatus}
          title={
            userToToggle?.status === "active"
              ? "Deactivate user"
              : "Activate user"
          }
          message={
            <>
              Change account access for <strong>{userToToggle?.name}</strong>?{" "}
              {userToToggle?.status === "active"
                ? "They will no longer be able to sign in while inactive."
                : "They will be able to sign in again."}
            </>
          }
          confirmText={
            userToToggle?.status === "active" ? "Deactivate" : "Activate"
          }
          isDestructive={userToToggle?.status === "active"}
        />
        <ConfirmationModal
          isOpen={!!userToDelete}
          onClose={() => setUserToDelete(null)}
          onConfirm={confirmDeleteUser}
          title="Delete User"
          message={
            <>
              Are you sure you want to delete user{" "}
              <strong className="text-white">{userToDelete?.name}</strong>? This
              action cannot be undone.
            </>
          }
          confirmText="Delete"
          cancelText="Cancel"
          isDestructive={true}
        />
      </section>
    </div>
  );
};

export default AdminPanel;
