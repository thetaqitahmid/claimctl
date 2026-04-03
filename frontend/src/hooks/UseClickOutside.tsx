// Hook to handle clicks outside a specified element to close a dropdown or modal
// This hook is useful for dropdown menus, modals, or any component that should
// close when clicking outside of it.

import { useEffect } from "react";

export interface UseClickOutsideProps {
  ref: React.RefObject<HTMLDivElement>;
  isOpen: boolean;
  onClose: () => void;
}

export const useClickOutside = ({
  ref,
  isOpen,
  onClose,
}: UseClickOutsideProps) => {
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (ref.current && !ref.current.contains(event.target as Node)) {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
    }
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [onClose, isOpen, ref]);
};
