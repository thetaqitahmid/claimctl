import React, { useState, useRef } from "react";
import {
  Download,
  Upload,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Loader2,
  Database,
} from "lucide-react";
import { useAppSelector } from "../store/store";

const BackupRestore: React.FC = () => {
  const authData = useAppSelector((state) => state.authSlice);
  const [restoreStatus, setRestoreStatus] = useState<
    "idle" | "uploading" | "success" | "error"
  >("idle");
  const [restoreMessage, setRestoreMessage] = useState("");
  const [backupStatus, setBackupStatus] = useState<
    "idle" | "downloading" | "success" | "error"
  >("idle");
  const [showConfirm, setShowConfirm] = useState(false);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  if (authData.role !== "admin") {
    return <div className="p-8 text-center text-red-500">Access Denied</div>;
  }

  const handleBackup = async () => {
    setBackupStatus("downloading");
    try {
      const response = await fetch("/api/admin/backup", {
        credentials: "include",
      });
      if (!response.ok) {
        throw new Error(`Backup failed: ${response.statusText}`);
      }
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      const date = new Date().toISOString().slice(0, 19).replace(/:/g, "");
      a.download = `claimctl-backup-${date}.json`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      setBackupStatus("success");
      setTimeout(() => setBackupStatus("idle"), 3000);
    } catch (err: unknown) {
      setBackupStatus("error");
      console.error(err instanceof Error ? err.message : "Backup failed");
      setTimeout(() => setBackupStatus("idle"), 5000);
    }
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0] || null;
    setSelectedFile(file);
    setRestoreStatus("idle");
    setRestoreMessage("");
    if (file) {
      setShowConfirm(true);
    }
  };

  const handleRestore = async () => {
    if (!selectedFile) return;
    setShowConfirm(false);
    setRestoreStatus("uploading");

    try {
      const formData = new FormData();
      formData.append("file", selectedFile);

      const response = await fetch("/api/admin/restore", {
        method: "POST",
        credentials: "include",
        body: formData,
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || "Restore failed");
      }

      setRestoreStatus("success");
      setRestoreMessage(data.message || "Backup restored successfully");
      setSelectedFile(null);
      if (fileInputRef.current) fileInputRef.current.value = "";
    } catch (err: unknown) {
      setRestoreStatus("error");
      setRestoreMessage(err instanceof Error ? err.message : "Restore failed");
    }
  };

  const cancelRestore = () => {
    setShowConfirm(false);
    setSelectedFile(null);
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

  return (
    <div className="space-y-6">
      {/* Backup Section */}
      <div className="glass-panel rounded-xl p-6 border border-slate-700/50">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-cyan-500/10">
              <Database className="w-5 h-5 text-cyan-400" />
            </div>
            <div>
              <h3 className="text-lg font-medium text-white">Create Backup</h3>
              <p className="text-sm text-slate-400 mt-0.5">
                Download a full snapshot of all application data as JSON
              </p>
            </div>
          </div>
          <button
            onClick={handleBackup}
            disabled={backupStatus === "downloading"}
            className="flex items-center gap-2 px-5 py-2.5 rounded-lg text-sm font-medium
                            bg-cyan-500/10 hover:bg-cyan-500/20 text-cyan-400 border border-cyan-500/20
                            transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {backupStatus === "downloading" ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : backupStatus === "success" ? (
              <CheckCircle className="w-4 h-4 text-emerald-400" />
            ) : (
              <Download className="w-4 h-4" />
            )}
            {backupStatus === "downloading"
              ? "Downloading..."
              : backupStatus === "success"
                ? "Downloaded"
                : "Download Backup"}
          </button>
        </div>
        <div className="mt-4 p-3 rounded-lg bg-slate-800/30 border border-slate-700/30">
          <p className="text-xs text-slate-500">
            Includes: resources, users, reservations, spaces, groups, webhooks,
            secrets, settings, health configs, and history. Secrets are exported
            in encrypted form and require the same encryption key on the target
            instance.
          </p>
        </div>
      </div>

      {/* Restore Section */}
      <div className="glass-panel rounded-xl p-6 border border-slate-700/50">
        <div className="flex items-center gap-3 mb-4">
          <div className="p-2 rounded-lg bg-amber-500/10">
            <Upload className="w-5 h-5 text-amber-400" />
          </div>
          <div>
            <h3 className="text-lg font-medium text-white">
              Restore from Backup
            </h3>
            <p className="text-sm text-slate-400 mt-0.5">
              Upload a backup file to replace all existing data
            </p>
          </div>
        </div>

        <div className="flex items-center gap-4">
          <label className="flex-1">
            <input
              ref={fileInputRef}
              type="file"
              accept=".json"
              onChange={handleFileSelect}
              className="block w-full text-sm text-slate-400
                                file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border file:border-slate-600
                                file:text-sm file:font-medium file:bg-slate-800 file:text-slate-300
                                hover:file:bg-slate-700 file:transition-colors file:cursor-pointer"
            />
          </label>
        </div>

        {/* Status Messages */}
        {restoreStatus === "uploading" && (
          <div className="mt-4 flex items-center gap-2 text-cyan-400 text-sm">
            <Loader2 className="w-4 h-4 animate-spin" />
            Restoring backup...
          </div>
        )}
        {restoreStatus === "success" && (
          <div className="mt-4 flex items-center gap-2 text-emerald-400 text-sm">
            <CheckCircle className="w-4 h-4" />
            {restoreMessage}
          </div>
        )}
        {restoreStatus === "error" && (
          <div className="mt-4 flex items-center gap-2 text-red-400 text-sm">
            <XCircle className="w-4 h-4" />
            {restoreMessage}
          </div>
        )}

        {/* Warning */}
        <div className="mt-4 p-3 rounded-lg bg-amber-500/5 border border-amber-500/20">
          <div className="flex items-start gap-2">
            <AlertTriangle className="w-4 h-4 text-amber-400 mt-0.5 shrink-0" />
            <p className="text-xs text-amber-400/80">
              Restoring a backup will permanently replace all existing data. All
              active sessions will be invalidated and users will need to log in
              again.
            </p>
          </div>
        </div>
      </div>

      {/* Confirmation Modal */}
      {showConfirm && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center">
          <div className="glass-panel rounded-2xl p-6 max-w-md w-full mx-4 border border-slate-700/50">
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2 rounded-full bg-red-500/10">
                <AlertTriangle className="w-6 h-6 text-red-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">
                Confirm Restore
              </h3>
            </div>
            <p className="text-slate-300 text-sm mb-2">
              You are about to restore from:
            </p>
            <p className="text-cyan-400 text-sm font-mono mb-4 truncate">
              {selectedFile?.name}
            </p>
            <div className="p-3 rounded-lg bg-red-500/5 border border-red-500/20 mb-6">
              <p className="text-xs text-red-400">
                This will permanently delete all existing data and replace it
                with the contents of this backup file. This action cannot be
                undone.
              </p>
            </div>
            <div className="flex justify-end gap-3">
              <button
                onClick={cancelRestore}
                className="px-4 py-2 rounded-lg text-sm text-slate-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleRestore}
                className="px-4 py-2 rounded-lg text-sm font-medium
                                    bg-red-500/10 hover:bg-red-500/20 text-red-400
                                    border border-red-500/20 transition-colors"
              >
                Yes, Restore
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default BackupRestore;
