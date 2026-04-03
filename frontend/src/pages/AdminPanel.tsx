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
import SpaceManagement from "../components/SpaceManagement";
import GroupManagement from "../components/GroupManagement";
import ConfirmationModal from "../components/ConfirmationModal";
import { useNotificationContext } from "../hooks/useNotification";

const AdminPanel = () => {
  const [currentUser, setCurrentUser] = useState<userPropType | undefined>(
    undefined
  );
  const [userModalOpen, setUserModalOpen] = useState<boolean>(false);
  const { data: usersData, error, isLoading } = useGetUsersQuery();
  const users = usersData || [];
  const [deleteUser] = useDeleteUserMutation();
  const [editUser] = useUpdateUserMutation();

  const [userToDelete, setUserToDelete] = useState<userPropType | null>(null);
  const { showNotification } = useNotificationContext();

  const onDeleteAction = (user: userPropType) => {
    setUserToDelete(user);
  };

  const confirmDeleteUser = async () => {
    if (userToDelete) {
      try {
        await deleteUser(userToDelete.id).unwrap();
        showNotification('success', `User ${userToDelete.name} deleted successfully`);
      } catch (err) {
        const error = err as { data?: { error?: string }; message?: string };
        console.error("Failed to delete user", userToDelete, error);
        showNotification('error', `Failed to delete user: ${error?.data?.error || error.message || 'Unknown error'}`);
      } finally {
        setUserToDelete(null);
      }
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

  const handleUserStatus = async (user: userPropType) => {
    const newUser: userPropType = {
      ...user,
      status: user.status === "active" ? "inactive" : "active",
    };
    try {
      await editUser(newUser).unwrap();
      showNotification('success', `User status updated successfully`);
    } catch (err) {
      const error = err as { data?: { error?: string }; message?: string };
      console.error(error);
      showNotification('error', `Failed to update user status: ${error?.data?.error || error.message || 'Unknown error'}`);
    }
  };

  return (
    <div className="min-h-screen">
      {/* Main Content */}
      <main className="max-w-7xl mx-auto p-6 lg:p-8">
        <SpaceManagement />

        <GroupManagement />

        <div className="mb-8 flex items-center justify-between">
          <div>
            <h2 className="text-3xl font-bold text-white tracking-tight flex items-center gap-3">
              <UserIcon className="w-8 h-8 text-brand-queued" />
              Users
            </h2>
            <p className="text-slate-400 mt-1">Manage system accounts and permissions</p>
          </div>
          <button
            onClick={handleAddUser}
            className="btn-primary"
          >
            <Plus className="w-4 h-4" />
            Add User
          </button>
        </div>

        {/* Error State */}
        {error && (
          <div className="glass-panel p-4 rounded-xl border-brand-busy/20 bg-brand-busy/5 mb-6 text-center">
            <p className="text-brand-busy font-medium">Failed to fetch users</p>
          </div>
        )}

        {/* Loading State */}
        {isLoading ? (
          <div className="flex flex-col items-center justify-center p-20 glass-panel rounded-xl">
             <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-brand-queued mb-4"></div>
             <p className="text-slate-400 font-medium">Loading users...</p>
          </div>
        ) : (
          <UserTable
            data={users}
            onEdit={handleUserEdit}
            onToggleStatus={handleUserStatus}
            onDelete={onDeleteAction}
          />
        )}

        {/* Empty State */}
        {!isLoading && users.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 glass-panel rounded-xl">
            <UserIcon className="w-12 h-12 text-slate-700 mb-4" />
            <h3 className="text-lg font-medium text-white">No users found</h3>
            <p className="text-slate-400">Database is currently empty</p>
          </div>
        )}

        <UserModal
          user={currentUser}
          isOpen={userModalOpen}
          onClose={() => setUserModalOpen(false)}
        />

        <ConfirmationModal
          isOpen={!!userToDelete}
          onClose={() => setUserToDelete(null)}
          onConfirm={confirmDeleteUser}
          title="Delete User"
          message={<>Are you sure you want to delete user <strong className="text-white">{userToDelete?.name}</strong>? This action cannot be undone.</>}
          confirmText="Delete"
          cancelText="Cancel"
          isDestructive={true}
        />
      </main>
    </div>
  );
};

export default AdminPanel;
