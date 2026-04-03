interface ConfirmDeleteDialogProps<T extends { name: string }> {
  object: T;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmDeleteDialog<T extends { name: string }>({
  object,
  onConfirm,
  onCancel,
}: ConfirmDeleteDialogProps<T>) {
  return (
    <div className="fixed inset-0 flex items-center justify-center bg-black bg-opacity-50">
      <p className="text-white">
        Are you sure you want to delete {object.name}?
      </p>
      <button onClick={onConfirm}>Delete</button>
      <button onClick={onCancel}>Cancel</button>
    </div>
  );
}
