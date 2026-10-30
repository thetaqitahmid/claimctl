import { useEffect, useState } from "react";
import { Dialog, DialogPanel, DialogTitle } from "@headlessui/react";
import { UserGroup } from "../types";
import {
  useCreateGroupMutation,
  useUpdateGroupMutation,
} from "../store/api/groups";
import { useNotificationContext } from "../hooks/useNotification";

export const GroupModal = ({
  group,
  onClose,
  isOpen,
}: {
  group?: UserGroup;
  onClose: () => void;
  isOpen: boolean;
}) => {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState("");
  const [createGroup, { isLoading: creating }] = useCreateGroupMutation();
  const [updateGroup, { isLoading: updating }] = useUpdateGroupMutation();
  const { showNotification } = useNotificationContext();
  const pending = creating || updating;
  useEffect(() => {
    setName(group?.name || "");
    setDescription(group?.description || "");
    setError("");
  }, [group, isOpen]);

  const save = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!name.trim()) {
      setError("Enter a group name.");
      return;
    }
    setError("");
    try {
      if (group)
        await updateGroup({
          id: group.id,
          name: name.trim(),
          description: description.trim(),
        }).unwrap();
      else
        await createGroup({
          name: name.trim(),
          description: description.trim(),
        }).unwrap();
      showNotification(
        "success",
        group ? "Group updated" : "Group created. Add members to get started.",
      );
      onClose();
    } catch (err) {
      setError(
        (err as { data?: { error?: string } })?.data?.error ||
          "Could not save group. Your entries have been kept. Try again.",
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
        <DialogPanel className="admin-panel p-6 w-full max-w-lg">
          <DialogTitle className="text-xl font-semibold text-white">
            {group ? "Edit group" : "Create group"}
          </DialogTitle>
          <form onSubmit={save} className="space-y-5 mt-5">
            <label className="block text-sm">
              Group name
              <input
                autoFocus
                className="admin-input mt-2"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                disabled={pending}
              />
            </label>
            <label className="block text-sm">
              Description (optional)
              <textarea
                className="admin-input mt-2"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={3}
                disabled={pending}
              />
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
                {pending ? "Saving…" : group ? "Save changes" : "Create group"}
              </button>
            </div>
          </form>
        </DialogPanel>
      </div>
    </Dialog>
  );
};
