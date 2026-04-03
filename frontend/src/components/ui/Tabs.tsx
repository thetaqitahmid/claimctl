import React from 'react';

export interface Tab {
  id: string;
  label: string;
}

interface TabsProps {
  tabs: Tab[];
  activeTab: string;
  onTabChange: (id: string) => void;
  className?: string;
}

const Tabs: React.FC<TabsProps> = ({ tabs, activeTab, onTabChange, className = '' }) => {
  return (
    <div className={`flex gap-4 border-b border-slate-700/50 pb-1 ${className}`}>
      {tabs.map((tab) => (
        <button
          key={tab.id}
          onClick={() => onTabChange(tab.id)}
          className={`pb-3 px-4 text-sm font-medium transition-all duration-200 relative ${
            activeTab === tab.id
              ? "text-cyan-400"
              : "text-slate-400 hover:text-slate-200"
          }`}
        >
          {tab.label}
          {activeTab === tab.id && (
            <span className="absolute bottom-[-1px] left-0 w-full h-0.5 bg-cyan-400 rounded-full" />
          )}
        </button>
      ))}
    </div>
  );
};

export default Tabs;
