import React, { useEffect, useState } from 'react';
import { X, CheckCircle, AlertCircle, Info, AlertTriangle } from 'lucide-react';

export type ToastType = 'success' | 'error' | 'info' | 'warning';

interface ToastProps {
  type: ToastType;
  message: string;
  onClose: () => void;
}

const icons = {
  success: <CheckCircle className="w-5 h-5 text-emerald-400" />,
  error: <AlertCircle className="w-5 h-5 text-rose-400" />,
  info: <Info className="w-5 h-5 text-cyan-400" />,
  warning: <AlertTriangle className="w-5 h-5 text-amber-400" />,
};

const styles = {
  success: 'bg-emerald-950/90 border-emerald-500/30 text-emerald-100',
  error: 'bg-rose-950/90 border-rose-500/30 text-rose-100',
  info: 'bg-cyan-950/90 border-cyan-500/30 text-cyan-100',
  warning: 'bg-amber-950/90 border-amber-500/30 text-amber-100',
};

export const Toast: React.FC<ToastProps> = ({ type, message, onClose }) => {
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    // Small delay to allow enter animation
    const timer = setTimeout(() => setIsVisible(true), 10);
    return () => clearTimeout(timer);
  }, []);

  const handleClose = () => {
    setIsVisible(false);
    // Wait for exit animation
    setTimeout(onClose, 300);
  };

  return (
    <div
      className={`
        flex items-center gap-3 px-4 py-3 rounded-lg border shadow-lg backdrop-blur-md
        transition-all duration-300 transform translate-y-0 opacity-100
        ${styles[type]}
        ${isVisible ? 'translate-x-0 opacity-100' : 'translate-x-full opacity-0'}
        min-w-[300px] max-w-md
      `}
      role="alert"
    >
      <div className="flex-shrink-0">
        {icons[type]}
      </div>
      <p className="flex-1 text-sm font-medium">{message}</p>
      <button
        onClick={handleClose}
        className="flex-shrink-0 p-1 rounded-full hover:bg-white/10 transition-colors"
        aria-label="Close"
      >
        <X className="w-4 h-4 opacity-70" />
      </button>
    </div>
  );
};
