import { useEffect, useState } from "react";
import { Dialog, DialogPanel, DialogTitle } from "@headlessui/react";
import { User } from "../store/api/auth";
import {
  useCreateUserMutation,
  useUpdateUserMutation,
} from "../store/api/users";
import { useNotificationContext } from "../hooks/useNotification";

export const UserModal = ({
  user,
  onClose,
  isOpen,
}: {
  user?: User;
  onClose: () => void;
  isOpen: boolean;
}) => {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState("user");
  const [error, setError] = useState("");
  const [createUser, { isLoading: creating }] = useCreateUserMutation();
  const [updateUser, { isLoading: updating }] = useUpdateUserMutation();
  const { showNotification } = useNotificationContext();
  const pending = creating || updating;
  useEffect(() => {
    setName(user?.name || "");
    setEmail(user?.email || "");
    setPassword("");
    setRole(user?.role || "user");
    setError("");
  }, [user, isOpen]);

  const save = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!name.trim() || !email.trim()) {
      setError("Enter a name and email address.");
      return;
    }
    setError("");
    try {
      if (user)
        await updateUser({
          id: user.id,
          name: name.trim(),
          email: email.trim(),
          role,
        }).unwrap();
      else
        await createUser({
          name: name.trim(),
          email: email.trim(),
          password,
          role,
          status: "active",
        }).unwrap();
      showNotification("success", user ? "User updated" : "User created");
      setPassword("");
      onClose();
    } catch (err) {
      setError(
        (err as { data?: { error?: string } })?.data?.error ||
          "Could not save user. Your entries have been kept. Try again.",
      );
    }
  };

  return (
    <Dialog
      open={isOpen}
      onClose={() => {
        if (!pending) onClose();
      }}
      className="fixed inset-0 z-50"
    >
      <div className="fixed inset-0 bg-slate-950/80" aria-hidden="true" />
      <div className="fixed inset-0 flex items-center justify-center p-4">
        <DialogPanel className="admin-panel p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto">
          <DialogTitle className="text-xl font-semibold text-white">
            {user ? "Edit user" : "Add user"}
          </DialogTitle>
          <form onSubmit={save} className="space-y-5 mt-5">
            <label className="block text-sm">
              Name
              <input
                autoFocus
                className="admin-input mt-2"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                disabled={pending}
                autoComplete="name"
              />
            </label>
            <label className="block text-sm">
              Email address
              <input
                className="admin-input mt-2"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                disabled={pending}
                autoComplete="email"
              />
            </label>
            {!user && (
              <label className="block text-sm">
                Password
                <input
                  className="admin-input mt-2"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  disabled={pending}
                  autoComplete="new-password"
                />
              </label>
            )}
            <label className="block text-sm">
              System role
              <select
                className="admin-input mt-2"
                value={role}
                onChange={(e) => setRole(e.target.value)}
                disabled={pending}
              >
                <option value="user">User</option>
                <option value="admin">Administrator</option>
              </select>
              <span className="block text-sm text-slate-400 mt-2">
                Administrators can manage all users, spaces, and system
                configuration.
              </span>
            </label>
            {error && (
              <p role="alert" className="text-red-300">
                {error}
              </p>
            )}
            <div className="flex justify-end gap-3">
              <button
                type="button"
                className="btn-ghost"
                disabled={pending}
                onClick={onClose}
              >
                Cancel
              </button>
              <button className="btn-primary" disabled={pending}>
                {pending ? "Saving…" : user ? "Save changes" : "Add user"}
              </button>
            </div>
          </form>
        </DialogPanel>
      </div>
    </Dialog>
  );
};
