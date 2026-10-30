import { useCallback } from "react";
import { useBeforeUnload, useBlocker } from "react-router-dom";
import ConfirmationModal from "../ConfirmationModal";

const UnsavedChangesGuard = ({
  dirty,
  pending = false,
}: {
  dirty: boolean;
  pending?: boolean;
}) => {
  const blocker = useBlocker(dirty);
  useBeforeUnload(
    useCallback(
      (event) => {
        if (dirty) {
          event.preventDefault();
          event.returnValue = "";
        }
      },
      [dirty],
    ),
  );

  return (
    <ConfirmationModal
      isOpen={blocker.state === "blocked"}
      title={pending ? "Operation in progress" : "Discard unsaved changes?"}
      message={
        pending
          ? "Please stay on this page until the operation completes. Leaving will not cancel a request already sent to the server."
          : "Your changes have not been saved. Stay on this page to continue editing."
      }
      confirmText={pending ? "Keep waiting" : "Discard and leave"}
      cancelText={pending ? "Stay on page" : "Keep editing"}
      isDestructive
      closeOnConfirm={false}
      onClose={() => {
        if (blocker.state === "blocked") blocker.reset();
      }}
      onConfirm={() => {
        if (blocker.state === "blocked") {
          if (pending) blocker.reset();
          else blocker.proceed();
        }
      }}
    />
  );
};

export default UnsavedChangesGuard;
