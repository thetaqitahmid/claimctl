import React, { useState } from "react";
import {
  Dialog,
  DialogPanel,
  DialogTitle,
  Description,
} from "@headlessui/react";
import { X } from "lucide-react";

interface ConfirmationModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void | Promise<void>;
  title: string;
  message: React.ReactNode;
  confirmText?: string;
  cancelText?: string;
  isDestructive?: boolean;
  confirmationWord?: string;
  closeOnConfirm?: boolean;
}

const ConfirmationModal = ({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmText = "Confirm",
  cancelText = "Cancel",
  isDestructive = false,
  confirmationWord,
  closeOnConfirm = true,
}: ConfirmationModalProps) => {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const [confirmation, setConfirmation] = useState("");

  const close = () => {
    if (pending) return;
    setError("");
    setConfirmation("");
    onClose();
  };

  const confirm = async () => {
    if (pending) return;
    setPending(true);
    setError("");
    try {
      await onConfirm();
      setConfirmation("");
      if (closeOnConfirm) onClose();
    } catch (err) {
      setError(
        (err as { data?: { error?: string } })?.data?.error ||
          (err instanceof Error
            ? err.message
            : "The operation failed. Please try again."),
      );
    } finally {
      setPending(false);
    }
  };

  return (
    <Dialog open={isOpen} onClose={close} className="fixed inset-0 z-50">
      <div className="fixed inset-0 bg-slate-950/80" aria-hidden="true" />
      <div className="fixed inset-0 flex items-center justify-center p-4 overflow-y-auto">
        <DialogPanel className="bg-slate-900 border border-slate-700 rounded-xl shadow-2xl max-w-md w-full p-6">
          <div className="flex justify-between items-start mb-4 gap-4">
            <DialogTitle className="text-xl font-semibold text-white">
              {title}
            </DialogTitle>
            <button
              onClick={close}
              disabled={pending}
              aria-label="Close confirmation"
              className="text-slate-300 p-1"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
          <Description as="div" className="text-slate-300 mb-6 leading-relaxed">
            {message}
          </Description>
          {confirmationWord && (
            <label className="block text-sm text-slate-300 mb-6">
              Type <strong>{confirmationWord}</strong> to confirm
              <input
                className="admin-input mt-2"
                value={confirmation}
                onChange={(e) => setConfirmation(e.target.value)}
                disabled={pending}
                autoComplete="off"
              />
            </label>
          )}
          {error && (
            <p role="alert" className="text-red-300 text-sm mb-4">
              {error}
            </p>
          )}
          <div className="flex justify-end gap-3">
            <button
              data-autofocus
              onClick={close}
              disabled={pending}
              className="btn-ghost"
            >
              {cancelText}
            </button>
            <button
              onClick={confirm}
              disabled={
                pending ||
                Boolean(confirmationWord && confirmation !== confirmationWord)
              }
              className={`px-4 py-2 rounded-lg text-sm font-medium text-white ${isDestructive ? "bg-red-700 hover:bg-red-800" : "bg-cyan-700 hover:bg-cyan-800"}`}
            >
              {pending ? "Working…" : confirmText}
            </button>
          </div>
        </DialogPanel>
      </div>
    </Dialog>
  );
};

export default ConfirmationModal;
