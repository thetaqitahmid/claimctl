import React, { useState, useCallback, useEffect } from "react";
import { Resource } from "../types";
import { useUpdateResourceMutation } from "../store/api/resources";
import { Edit3, X, Zap } from "lucide-react";

interface Webhook {
  id: string;
  name: string;
}

interface AssignedWebhook extends Webhook {
  events: string[];
}

interface EditResourceModalProps {
  resource: Resource | null;
  onClose: () => void;
  onSave: (updatedResource: Resource) => void;
}

const EditResourceModal: React.FC<EditResourceModalProps> = ({
  resource,
  onClose,
  onSave,
}) => {
  const [name, setName] = useState(resource?.name || "");
  const [type, setType] = useState(resource?.type || "");
  const [labels, setLabels] = useState(resource?.labels.join(", ") || "");
  const [properties, setProperties] = useState<{ [key: string]: string }>(resource?.properties || {});
  const [updateMutation] = useUpdateResourceMutation();

  // Webhooks state
  const [availableWebhooks, setAvailableWebhooks] = useState<Webhook[]>([]);
  const [assignedWebhooks, setAssignedWebhooks] = useState<AssignedWebhook[]>([]);
  const [isLoadingWebhooks, setIsLoadingWebhooks] = useState(false);
  const [selectedWebhookId, setSelectedWebhookId] = useState<string>("");
  const [selectedEvents, setSelectedEvents] = useState<string[]>([]);

  const fetchWebhooksData = useCallback(async () => {
      setIsLoadingWebhooks(true);
      try {
          // Fetch all webhooks
          const allRes = await fetch('/api/webhooks', { credentials: 'include' });
          if (allRes.ok) {
             const ct = allRes.headers.get("content-type");
             if (ct && ct.includes("application/json")) {
                 const allData = await allRes.json();
                 setAvailableWebhooks(allData || []);
             }
          }

          // Fetch assigned webhooks
          const assignedRes = await fetch(`/api/resources/${resource?.id}/webhooks`, { credentials: 'include' });
          if (assignedRes.ok) {
               const ct = assignedRes.headers.get("content-type");
               if (ct && ct.includes("application/json")) {
                   const assignedData = await assignedRes.json();
                   setAssignedWebhooks(assignedData || []);
               }
          }
      } catch (e) {
          console.error("Failed to fetch webhooks", e);
      } finally {
          setIsLoadingWebhooks(false);
      }
  }, [resource?.id]);

  // Fetch webhooks on mount
  useEffect(() => {
     if (resource) {
         fetchWebhooksData();
     }
  }, [resource, fetchWebhooksData]);

  const handleAddWebhook = async () => {
      if (!selectedWebhookId || selectedEvents.length === 0 || !resource) return;
      try {
          const res = await fetch(`/api/resources/${resource.id}/webhooks`, {
              method: 'POST',
              headers: {
                  'Content-Type': 'application/json',
              },
              body: JSON.stringify({
                  webhook_id: selectedWebhookId,
                  events: selectedEvents
              })
          });

          if (res.ok) {
              fetchWebhooksData();
              setSelectedWebhookId("");
              setSelectedEvents([]);
          } else {
              alert("Failed to assign webhook");
          }
      } catch (e) {
          console.error(e);
      }
  };

  const handleRemoveWebhook = async (webhookId: string) => {
      if (!confirm("Remove webhook assignment?") || !resource) return;
      try {
          const res = await fetch(`/api/resources/${resource.id}/webhooks/${webhookId}`, {
              method: 'DELETE',
              headers: {
                  'Content-Type': 'application/json',
              }
          });

          if (res.ok) {
              fetchWebhooksData();
          }
      } catch (e) {
          console.error(e);
      }
  };


  const handleSave = async () => {
    // ... existing save logic
    if (resource) {
      const updatedLabels = labels.split(",").map(l => l.trim()).filter(l => l !== "");

      const cleanProps: { [key: string]: string } = {};
      Object.entries(properties).forEach(([k, v]) => {
          if (k.trim()) cleanProps[k.trim()] = v.trim();
      });

      const updatedResource = { ...resource, name, type, labels: updatedLabels, properties: cleanProps };
      try {
        await updateMutation({
          id: updatedResource.id,
          name: updatedResource.name,
          type: updatedResource.type,
          labels: updatedResource.labels,
          properties: updatedResource.properties,
        });
        onSave(updatedResource);
        onClose();
      } catch (err) {
        console.error("Error updating resource:", err);
      }
    }
  };

  if (!resource) return null;

  const eventOptions = [
      "reservation.created",
      "reservation.cancelled",
      "reservation.activated",
      "reservation.completed"
  ];

  return (
     <div className="fixed inset-0 z-[100] flex items-center justify-center p-4 overflow-x-hidden overflow-y-auto outline-none focus:outline-none">
      <div
        className="fixed inset-0 bg-slate-950/80 backdrop-blur-md transition-opacity"
        onClick={onClose}
      />

      <div className="relative glass-panel w-full max-w-2xl rounded-2xl overflow-hidden shadow-2xl transition-all duration-200"> {/* Increased max-width */}
        <div className="flex justify-between items-center p-6 border-b border-slate-800/50">
          <div className="flex items-center gap-3">
             <div className="p-2 rounded-lg bg-brand-queued/10 text-brand-queued">
                <Edit3 className="h-5 w-5" />
             </div>
             <h3 className="text-xl font-semibold text-white">Edit Resource</h3>
          </div>
          <button
            onClick={onClose}
            className="p-2 hover:bg-slate-800 rounded-lg text-slate-400 hover:text-white transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="p-6 space-y-6 max-h-[70vh] overflow-y-auto"> {/* Added scroll */}
          <div className="space-y-4">
             {/* Name and Type Fields */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-2">
                <label htmlFor="edit-name" className="text-sm font-medium text-slate-300">
                    Resource Name
                </label>
                <input
                    id="edit-name"
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    className="w-full bg-slate-900/50 border border-slate-700 rounded-lg px-4 py-2 text-white placeholder-slate-500 focus:outline-none focus:border-brand-queued focus:ring-1 focus:ring-brand-queued transition-all"
                />
                </div>

                <div className="space-y-2">
                <label htmlFor="edit-type" className="text-sm font-medium text-slate-300">
                    Resource Type
                </label>
                <input
                    id="edit-type"
                    type="text"
                    value={type}
                    onChange={(e) => setType(e.target.value)}
                    className="w-full bg-slate-900/50 border border-slate-700 rounded-lg px-4 py-2 text-white placeholder-slate-500 focus:outline-none focus:border-brand-queued focus:ring-1 focus:ring-brand-queued transition-all"
                />
                </div>
            </div>

            {/* Properties Section */}
            <div className="space-y-4 border-t border-slate-800/50 pt-4">
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
              </div>
            </div>

             {/* Webhooks Section */}
             <div className="space-y-4 border-t border-slate-800/50 pt-4">
                 <h4 className="text-sm font-medium text-slate-300 flex items-center gap-2">
                    <Zap className="w-4 h-4 text-yellow-400" /> Webhooks
                 </h4>

                 {isLoadingWebhooks ? (
                     <div className="text-xs text-slate-500">Loading webhooks...</div>
                 ) : (
                     <div className="space-y-3">
                         {/* Assigned Webhooks List */}
                         {assignedWebhooks.map((link) => (
                             <div key={link.id} className="flex items-center justify-between bg-slate-800/40 p-3 rounded-lg border border-slate-700/50">
                                 <div>
                                     <div className="text-sm text-white font-medium">{link.name}</div>
                                     <div className="text-xs text-slate-400 flex gap-1 mt-1">
                                         {link.events?.map((ev: string) => (
                                             <span key={ev} className="bg-slate-700 px-1.5 rounded">{ev.split('.')[1]}</span>
                                         ))}
                                     </div>
                                 </div>
                                 <button
                                    onClick={() => handleRemoveWebhook(link.id)}
                                    className="text-slate-500 hover:text-red-400 p-1"
                                 >
                                     <X className="w-4 h-4" />
                                 </button>
                             </div>
                         ))}

                         {/* Add New Webhook */}
                         <div className="bg-slate-900/30 p-3 rounded-lg border border-slate-800/50 space-y-3">
                             <div className="flex gap-2">
                                 <select
                                    className="flex-1 bg-slate-900 border border-slate-700 rounded text-sm text-white px-2 py-1"
                                    value={selectedWebhookId}
                                    onChange={(e) => setSelectedWebhookId(e.target.value)}
                                 >
                                     <option value="">Select Webhook...</option>
                                     {availableWebhooks.filter(w => !assignedWebhooks.find(aw => aw.id === w.id)).map(w => (
                                         <option key={w.id} value={w.id}>{w.name}</option>
                                     ))}
                                 </select>
                                 <button
                                    onClick={handleAddWebhook}
                                    disabled={!selectedWebhookId || selectedEvents.length === 0}
                                    className="bg-cyan-600 hover:bg-cyan-700 disabled:opacity-50 disabled:cursor-not-allowed text-white text-xs px-3 py-1 rounded"
                                 >
                                     Add
                                 </button>
                             </div>
                             {selectedWebhookId && (
                                 <div className="flex flex-wrap gap-2">
                                     {eventOptions.map(ev => (
                                         <label key={ev} className="flex items-center gap-1.5 text-xs text-slate-300 cursor-pointer hover:text-white">
                                             <input
                                                type="checkbox"
                                                checked={selectedEvents.includes(ev)}
                                                onChange={(e) => {
                                                    if (e.target.checked) setSelectedEvents([...selectedEvents, ev]);
                                                    else setSelectedEvents(selectedEvents.filter(x => x !== ev));
                                                }}
                                                className="rounded border-slate-700 bg-slate-800"
                                             />
                                             {ev.replace('reservation.', '')}
                                         </label>
                                     ))}
                                 </div>
                             )}
                         </div>
                     </div>
                 )}
             </div>

            <div className="space-y-2 border-t border-slate-800/50 pt-4">
              <label htmlFor="edit-labels" className="text-sm font-medium text-slate-300">
                Labels (comma-separated)
              </label>
              <input
                id="edit-labels"
                type="text"
                value={labels}
                onChange={(e) => setLabels(e.target.value)}
                className="w-full bg-slate-900/50 border border-slate-700 rounded-lg px-4 py-2 text-white placeholder-slate-500 focus:outline-none focus:border-brand-queued focus:ring-1 focus:ring-brand-queued transition-all"
              />
            </div>
          </div>
        </div>

        <div className="p-6 pt-2 flex flex-col sm:flex-row-reverse gap-3 bg-slate-900/50">
          <button
            onClick={handleSave}
            disabled={!name.trim()}
            className="btn-primary w-full sm:w-auto px-8 py-2.5 disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-brand-queued/10"
          >
            Save Changes
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

export default EditResourceModal;
