import React, { useState, useEffect } from "react";
import { User as UserProps } from "../store/api/auth";
import { useCreateUserMutation, useUpdateUserMutation } from "../store/api/users";
import { X, Save, UserPlus, FileEdit, Shield, Mail, Lock, User } from "lucide-react";

interface UserModalProps {
  user?: UserProps;
  onClose: () => void;
  isOpen: boolean;
}

export const UserModal: React.FC<UserModalProps> = ({
  user,
  onClose,
  isOpen,
}) => {
  const isEditing = !!user;
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState(""); // Only required for creating users
  const [role, setRole] = useState("user");

  const [error, setError] = useState("");

  const [createUser, { isLoading: isCreating }] = useCreateUserMutation();
  const [updateUser, { isLoading: isUpdating }] = useUpdateUserMutation();

  useEffect(() => {
    if (user) {
      setName(user.name);
      setEmail(user.email);
      setRole(user.role);

    } else {
      // Reset form for "Add User"
      setName("");
      setEmail("");
      setPassword("");
      setRole("user");

    }
    setError("");
  }, [user, isOpen]);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    // Basic Validation
    if (!name || !email) {
      setError("Name and Email are required.");
      return;
    }
    if (!isEditing && !password) {
      setError("Password is required for new users.");
      return;
    }

    try {
      if (isEditing && user) {
        await updateUser({
          id: user.id,
          name,
          email,
          role,

        }).unwrap();
      } else {
        await createUser({
          name,
          email,
          password,
          role,

          status: "active", // Default status
        }).unwrap();
      }
      onClose();
    } catch (err) {
      console.error("Failed to save user:", err);
      setError("Failed to save user. Please try again.");
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm transition-opacity duration-300">
      <div
        className="w-full max-w-md bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl overflow-hidden transform transition-all duration-300 scale-100"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800 bg-slate-900/50">
          <h2 className="text-xl font-semibold text-white flex items-center gap-2">
            {isEditing ? (
              <>
                <FileEdit className="w-5 h-5 text-cyan-400" />
                Edit User
              </>
            ) : (
              <>
                <UserPlus className="w-5 h-5 text-emerald-400" />
                Add New User
              </>
            )}
          </h2>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white transition-colors p-1 rounded-lg hover:bg-slate-800"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Form body */}
        <form onSubmit={handleSubmit} className="p-6 space-y-5">
           {error && (
            <div className="p-3 text-sm text-red-400 bg-red-500/10 border border-red-500/20 rounded-lg">
              {error}
            </div>
          )}

          <div className="space-y-4">
            {/* Name */}
            <div>
              <label className="block text-sm font-medium text-slate-400 mb-1.5 flex items-center gap-2">
                <User className="w-3.5 h-3.5" /> Name
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full px-4 py-2.5 bg-slate-950 border border-slate-800 rounded-xl text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500/50 transition-all placeholder:text-slate-600"
                placeholder="John Doe"
              />
            </div>

            {/* Email */}
            <div>
              <label className="block text-sm font-medium text-slate-400 mb-1.5 flex items-center gap-2">
                <Mail className="w-3.5 h-3.5" /> Email Address
              </label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full px-4 py-2.5 bg-slate-950 border border-slate-800 rounded-xl text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500/50 transition-all placeholder:text-slate-600"
                placeholder="john@example.com"
              />
            </div>

             {/* Password - Only for creating new users */}
             {!isEditing && (
              <div>
                <label className="block text-sm font-medium text-slate-400 mb-1.5 flex items-center gap-2">
                  <Lock className="w-3.5 h-3.5" /> Password
                </label>
                <input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full px-4 py-2.5 bg-slate-950 border border-slate-800 rounded-xl text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500/50 transition-all placeholder:text-slate-600"
                  placeholder="••••••••"
                />
                <p className="text-xs text-slate-500 mt-1">Must contain at least 8 characters.</p>
              </div>
            )}

            {/* Role & Admin */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                 <label className="block text-sm font-medium text-slate-400 mb-1.5 flex items-center gap-2">
                  <Shield className="w-3.5 h-3.5" /> Role
                </label>
                <select
                  value={role}
                  onChange={(e) => setRole(e.target.value)}
                  className="w-full px-4 py-2.5 bg-slate-950 border border-slate-800 rounded-xl text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500/50 transition-all cursor-pointer"
                >
                  <option value="user">User</option>
                  <option value="admin">Admin</option>
                </select>
              </div>


            </div>
          </div>

          {/* Footer Actions */}
          <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2.5 text-sm font-medium text-slate-300 hover:text-white bg-slate-800 hover:bg-slate-700/80 rounded-xl transition-all"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isCreating || isUpdating}
              className="flex items-center gap-2 px-6 py-2.5 text-sm font-medium text-white bg-gradient-to-r from-cyan-600 to-cyan-600 hover:from-cyan-500 hover:to-cyan-500 rounded-xl shadow-lg shadow-cyan-500/20 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isCreating || isUpdating ? (
                <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
              ) : (
                <Save className="w-4 h-4" />
              )}
              {isEditing ? "Save Changes" : "Create User"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
