import { useState } from "react";
import { Link } from "react-router-dom";
import { Dialog, DialogPanel, DialogTitle } from "@headlessui/react";
import { Plus } from "lucide-react";
import {
  useGetSpacesQuery,
  useCreateSpaceMutation,
  useDeleteSpaceMutation,
} from "../store/api/spaces";
import { Space } from "../types";
import ConfirmationModal from "./ConfirmationModal";
import { useNotificationContext } from "../hooks/useNotification";

const SpaceManagement = () => {
  const { data: spaces = [], isLoading, error, refetch } = useGetSpacesQuery();
  const [createSpace, { isLoading: creating }] = useCreateSpaceMutation();
  const [deleteSpace] = useDeleteSpaceMutation();
  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [createError, setCreateError] = useState("");
  const [spaceToDelete, setSpaceToDelete] = useState<Space | null>(null);
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(0);
  const { showNotification } = useNotificationContext();
  const filtered = spaces.filter((space) =>
    `${space.name} ${space.description}`
      .toLowerCase()
      .includes(search.toLowerCase()),
  );
  const pageCount = Math.max(1, Math.ceil(filtered.length / 10));
  const currentPage = Math.min(page, pageCount - 1);

  const handleCreate = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!name.trim()) {
      setCreateError("Enter a space name.");
      return;
    }
    setCreateError("");
    try {
      await createSpace({
        name: name.trim(),
        description: description.trim(),
      }).unwrap();
      setShowCreate(false);
      setName("");
      setDescription("");
      showNotification(
        "success",
        "Space created. Open Access to grant access to users or groups.",
      );
    } catch (err) {
      setCreateError(
        (err as { data?: { error?: string } })?.data?.error ||
          "Could not create space. Your entries have been kept. Please try again.",
      );
    }
  };

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <p className="text-sm text-slate-400 mb-2">
            Administration / People and access
          </p>
          <h1 className="text-3xl font-semibold text-white">Spaces</h1>
          <p className="text-slate-400 mt-2">
            Organize resources and control which people and groups can access
            them.
          </p>
        </div>
        <button
          className="btn-primary"
          onClick={() => {
            setCreateError("");
            setShowCreate(true);
          }}
        >
          <Plus size={18} />
          Create space
        </button>
      </header>
      <p className="admin-panel p-4 text-sm text-slate-300">
        The Default Space is public. Other spaces use explicit user and group
        grants. Administrators retain access.
      </p>
      <label className="block max-w-lg text-sm text-slate-300">
        Search spaces
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
        <p role="status">Loading spaces…</p>
      ) : error ? (
        <div role="alert" className="admin-panel p-6">
          <p>Could not load spaces.</p>
          <button className="btn-ghost mt-3" onClick={() => refetch()}>
            Retry
          </button>
        </div>
      ) : spaces.length === 0 ? (
        <div className="admin-panel p-8">
          <h2 className="text-lg text-white">No spaces yet</h2>
          <p className="mt-2 mb-4">Create a space to organize resources.</p>
          <button className="btn-primary" onClick={() => setShowCreate(true)}>
            Create first space
          </button>
        </div>
      ) : (
        <>
          <div className="admin-panel overflow-x-auto">
            <table className="w-full text-sm text-left">
              <thead className="bg-slate-800">
                <tr>
                  <th className="p-4" scope="col">
                    Name
                  </th>
                  <th className="p-4" scope="col">
                    Description
                  </th>
                  <th className="p-4" scope="col">
                    Access
                  </th>
                  <th className="p-4" scope="col">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800">
                {filtered.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="p-8 text-center">
                      No spaces match your search.
                    </td>
                  </tr>
                ) : (
                  filtered
                    .slice(currentPage * 10, (currentPage + 1) * 10)
                    .map((space) => (
                      <tr key={space.id}>
                        <td className="p-4 font-medium text-white">
                          {space.name}
                        </td>
                        <td className="p-4 text-slate-300">
                          {space.description || "No description"}
                        </td>
                        <td className="p-4">
                          {space.name === "Default Space"
                            ? "Public"
                            : "Restricted"}
                        </td>
                        <td className="p-4">
                          <div className="flex flex-wrap gap-2">
                            <Link
                              className="btn-ghost"
                              aria-label={`Manage access to ${space.name}`}
                              to={`/admin/spaces/${space.id}`}
                            >
                              Access
                            </Link>
                            {space.name !== "Default Space" && (
                              <button
                                className="btn-ghost text-red-300"
                                aria-label={`Delete ${space.name}`}
                                onClick={() => setSpaceToDelete(space)}
                              >
                                Delete
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))
                )}
              </tbody>
            </table>
          </div>
          <nav
            aria-label="Space pagination"
            className="flex flex-wrap items-center gap-3 text-sm"
          >
            <p className="mr-auto">
              {filtered.length} spaces · Page {currentPage + 1} of {pageCount}
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
        </>
      )}
      <Dialog
        open={showCreate}
        onClose={() => {
          if (!creating) setShowCreate(false);
        }}
        className="fixed inset-0 z-50"
      >
        <div className="fixed inset-0 bg-slate-950/80" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="admin-panel p-6 w-full max-w-lg">
            <DialogTitle className="text-xl font-semibold text-white">
              Create space
            </DialogTitle>
            <form className="space-y-5 mt-5" onSubmit={handleCreate}>
              <label className="block text-sm">
                Space name
                <input
                  autoFocus
                  className="admin-input mt-2"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                  disabled={creating}
                />
              </label>
              <label className="block text-sm">
                Description (optional)
                <textarea
                  className="admin-input mt-2"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={3}
                  disabled={creating}
                />
              </label>
              {createError && (
                <p role="alert" className="text-red-300">
                  {createError}
                </p>
              )}
              <div className="flex justify-end gap-3">
                <button
                  type="button"
                  className="btn-ghost"
                  disabled={creating}
                  onClick={() => setShowCreate(false)}
                >
                  Cancel
                </button>
                <button className="btn-primary" disabled={creating}>
                  {creating ? "Creating…" : "Create space"}
                </button>
              </div>
            </form>
          </DialogPanel>
        </div>
      </Dialog>
      <ConfirmationModal
        isOpen={Boolean(spaceToDelete)}
        onClose={() => setSpaceToDelete(null)}
        title="Delete space"
        message={
          <>
            Delete <strong>{spaceToDelete?.name}</strong>? All resources in this
            space will also be deleted. This cannot be undone.
          </>
        }
        confirmText="Delete space"
        isDestructive
        onConfirm={async () => {
          if (spaceToDelete) {
            await deleteSpace(spaceToDelete.id).unwrap();
            showNotification("success", "Space deleted");
          }
        }}
      />
    </div>
  );
};

export default SpaceManagement;
