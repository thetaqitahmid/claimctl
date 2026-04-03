import React, { useState } from 'react';

interface HelpPopoverProps {
  label: string;
  children: React.ReactNode;
}

const HelpPopover: React.FC<HelpPopoverProps> = ({ label, children }) => {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className="mt-2">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="text-xs text-cyan-500/80 hover:text-cyan-400 flex items-center gap-1 transition-colors"
      >
        {isOpen ? '▼' : '▶'} {label}
      </button>
      {isOpen && (
        <div className="mt-2 p-3 bg-slate-800/80 rounded-lg text-xs text-slate-400 border border-slate-700/50 animate-fade-in space-y-2">
            {children}
        </div>
      )}
    </div>
  );
};

export default HelpPopover;
