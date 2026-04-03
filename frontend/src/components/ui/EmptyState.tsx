import React from 'react';

interface EmptyStateProps {
  message: string;
  icon?: React.ReactNode;
}

const EmptyState: React.FC<EmptyStateProps> = ({ message, icon }) => {
  return (
    <div className="text-center py-12 bg-slate-800/30 rounded-2xl border border-dashed border-slate-700">
        {icon && <div className="mb-2 text-slate-500">{icon}</div>}
      <p className="text-slate-400">{message}</p>
    </div>
  );
};

export default EmptyState;
