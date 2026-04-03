import React, { useState } from "react";
import { CopyPlus, X } from "lucide-react";

import { useGetSpacesQuery } from "../store/api/spaces";
import { Space } from "../types";

interface addResourceProp {
  isOpen: boolean;
  onClose: () => void;
  onSave: (name: string, type: string, labels: string[], properties: { [key: string]: string }, spaceId?: string) => void;
  preSelectedSpaceId?: string;
}

const AddResourcePopup: React.FC<addResourceProp> = ({
  isOpen,
  onClose,
  onSave,
  preSelectedSpaceId,
}) => {
  const [name, setName] = useState("");
  const [type, setType] = useState("");
  const [labels, setLabels] = useState("");
  const [properties, setProperties] = useState<{ [key: string]: string }>({});
  const [selectedSpaceId, setSelectedSpaceId] = useState<string | undefined>(preSelectedSpaceId);

  React.useEffect(() => {
    if (isOpen) {
        setSelectedSpaceId(preSelectedSpaceId);
    }
  }, [isOpen, preSelectedSpaceId]);
  const { data: spaces } = useGetSpacesQuery();

  const handleSave = () => {
    if (!name.trim()) return;

    // Clean up empty keys/values if needed (optional)
    const cleanProps: { [key: string]: string } = {};
    Object.entries(properties).forEach(([k, v]) => {
        if (k.trim()) cleanProps[k.trim()] = v.trim();
    });

    onSave(
      name.trim(),
      type.trim(),
      labels.split(",").map((label) => label.trim()).filter(l => l !== ""),
      cleanProps,
      selectedSpaceId
    );
    setName("");
    setType("");
    setLabels("");
    setProperties({});
    setSelectedSpaceId(undefined);
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-4 overflow-x-hidden overflow-y-auto outline-none focus:outline-none">
      <div
        className="fixed inset-0 bg-slate-950/80 backdrop-blur-md transition-opacity"
        onClick={onClose}
      />

      <div className="relative glass-panel w-full max-w-lg rounded-2xl overflow-hidden shadow-2xl transition-all duration-200">
        <div className="flex justify-between items-center p-6 border-b border-slate-800/50">
          <div className="flex items-center gap-3">
             <div className="p-2 rounded-lg bg-brand-queued/10 text-brand-queued">
                <CopyPlus className="h-5 w-5" />
             </div>
             <h3 className="text-xl font-semibold text-white">Add New Resource</h3>
          </div>
          <button
            onClick={onClose}
            className="p-2 hover:bg-slate-800 rounded-lg text-slate-400 hover:text-white transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="p-6 space-y-5">
           <p className="text-slate-400 text-sm">
             Configure the details for the new resource. Use commas to separate multiple labels.
           </p>

          <div className="space-y-4">
            <div className="space-y-2">
              <label htmlFor="name" className="text-sm font-medium text-slate-300">
                Resource Name
              </label>
              <input
                id="name"
                type="text"
                autoFocus
                placeholder="e.g. Lab Server 42"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full bg-slate-900/50 border border-slate-700 rounded-lg px-4 py-2 text-white placeholder-slate-500 focus:outline-none focus:border-brand-queued focus:ring-1 focus:ring-brand-queued transition-all"
              />
            </div>

            <div className="space-y-2">
              <label htmlFor="type" className="text-sm font-medium text-slate-300">
                Resource Type
              </label>
              <input
                id="type"
                type="text"
                placeholder="e.g. Server, License, Cloud"
                value={type}
                onChange={(e) => setType(e.target.value)}
                className="w-full bg-slate-900/50 border border-slate-700 rounded-lg px-4 py-2 text-white placeholder-slate-500 focus:outline-none focus:border-brand-queued focus:ring-1 focus:ring-brand-queued transition-all"
              />
            </div>

            <div className="space-y-2">
              <label htmlFor="labels" className="text-sm font-medium text-slate-300">
                Labels (comma-separated)
              </label>
              <input
                id="labels"
                type="text"
                placeholder="e.g. high-priority, dev, stable"
                value={labels}
                onChange={(e) => setLabels(e.target.value)}
                className="w-full bg-slate-900/50 border border-slate-700 rounded-lg px-4 py-2 text-white placeholder-slate-500 focus:outline-none focus:border-brand-queued focus:ring-1 focus:ring-brand-queued transition-all"
              />
            </div>

            <div className="space-y-4">
              <div className="flex justify-between items-center">
                 <label className="text-sm font-medium text-slate-300">Properties (max 10)</label>
                 <button
                    onClick={() => {
                        if (Object.keys(properties).length < 10) {
                            setProperties({...properties, "": ""});
                        }
                    }}
                    disabled={Object.keys(properties).length >= 10}
                    type="button"
                    className="text-xs text-brand-queued hover:text-brand-queued/80 disabled:opacity-50 disabled:cursor-not-allowed"
                 >
                    + Add Property
                 </button>
              </div>
              <div className="space-y-2 max-h-40 overflow-y-auto pr-2">
                 {Object.entries(properties).map(([key, value], index) => (
                    <div key={index} className="flex gap-2 items-center">
                        <input
                            type="text"
                            placeholder="Key"
                            value={key}
                            onChange={(e) => {
                                const newKey = e.target.value;
                                const newProps = { ...properties };
                                delete newProps[key];
                                newProps[newKey] = value;
                                setProperties(newProps);
                            }}
                            className="w-1/3 bg-slate-900/50 border border-slate-700 rounded-lg px-3 py-1.5 text-sm text-white focus:border-brand-queued focus:outline-none"
                        />
                        <input
                            type="text"
                            placeholder="Value"
                            value={value}
                            onChange={(e) => {
                                setProperties({
                                    ...properties,
                                    [key]: e.target.value
                                });
                            }}
                           className="flex-1 bg-slate-900/50 border border-slate-700 rounded-lg px-3 py-1.5 text-sm text-white focus:border-brand-queued focus:outline-none"
                        />
                        <button
                            onClick={() => {
                                const newProps = { ...properties };
                                delete newProps[key];
                                setProperties(newProps);
                            }}
                            className="p-1.5 text-red-400 hover:bg-slate-800 rounded-md"
                        >
                            <X className="h-4 w-4" />
                        </button>
                    </div>
                 ))}
                 {Object.keys(properties).length === 0 && (
                    <p className="text-xs text-slate-500 italic text-center py-2">No properties added.</p>
                 )}
              </div>
            </div>

            <div className="space-y-2">
              <label htmlFor="space" className="text-sm font-medium text-slate-300">
                Space (Optional, defaults to Default Space)
              </label>
              <select
                id="space"
                value={selectedSpaceId || ""}
                onChange={(e) => setSelectedSpaceId(e.target.value || undefined)}
                className="w-full bg-slate-900/50 border border-slate-700 rounded-lg px-4 py-2 text-white placeholder-slate-500 focus:outline-none focus:border-brand-queued focus:ring-1 focus:ring-brand-queued transition-all appearance-none"
              >
                <option value="">Default Space</option>
                {spaces?.map((space: Space) => (
                  <option key={space.id} value={space.id}>
                    {space.name}
                  </option>
                ))}
              </select>
            </div>
          </div>
        </div>

        <div className="p-6 pt-2 flex flex-col sm:flex-row-reverse gap-3">
          <button
            onClick={handleSave}
            disabled={!name.trim()}
            className="btn-primary w-full sm:w-auto px-8 py-2.5 disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-brand-queued/10"
          >
            Create Resource
          </button>
          <button
            onClick={onClose}
            className="btn-ghost w-full sm:w-auto px-8 py-2.5"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
};

export default AddResourcePopup;
