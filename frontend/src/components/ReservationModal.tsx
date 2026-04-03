import { Fragment, useState } from 'react';
import { Dialog, Transition } from '@headlessui/react';
import { Clock, X } from 'lucide-react';

interface ReservationModalProps {
  isOpen: boolean;
  onClose: () => void;
  resourceName: string;
  onConfirm: (duration: string | null) => void;
}

const DURATIONS = [
  { label: 'Indefinite', value: null },
  { label: '30 Minutes', value: '30m' },
  { label: '1 Hour', value: '1h' },
  { label: '4 Hours', value: '4h' },
  { label: '1 Day', value: '24h' },
];

export default function ReservationModal({
  isOpen,
  onClose,
  resourceName,
  onConfirm,
}: ReservationModalProps) {
  const [selectedDuration, setSelectedDuration] = useState<string | null>(null);
  const [isCustomDuration, setIsCustomDuration] = useState(false);
  const [customDays, setCustomDays] = useState(0);
  const [customHours, setCustomHours] = useState(0);
  const [customMinutes, setCustomMinutes] = useState(0);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (isCustomDuration) {
      // Send separate components if > 0, e.g. "1d 2h 30m" which our backend helper will parse.
      // actually backend helper is simple: "1d2h30m" (no spaces)
      let durationStr = "";
      if (customDays > 0) durationStr += `${customDays}d`;
      if (customHours > 0) durationStr += `${customHours}h`;
      if (customMinutes > 0) durationStr += `${customMinutes}m`;

      onConfirm(durationStr);
    } else {
      onConfirm(selectedDuration);
    }
    onClose();
  };

  const handleDurationSelect = (value: string | null) => {
    setSelectedDuration(value);
    setIsCustomDuration(value === 'custom');
    // Reset custom inputs when switching away from custom
    if (value !== 'custom') {
      setCustomDays(0);
      setCustomHours(0);
      setCustomMinutes(0);
    }
  };

  return (
    <Transition appear show={isOpen} as={Fragment}>
      <Dialog as="div" className="relative z-50" onClose={onClose}>
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-black/80" />
        </Transition.Child>

        <div className="fixed inset-0 overflow-y-auto">
          <div className="flex min-h-full items-center justify-center p-4 text-center">
            <Transition.Child
              as={Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0 scale-95"
              enterTo="opacity-100 scale-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100 scale-100"
              leaveTo="opacity-0 scale-95"
            >
              <Dialog.Panel className="w-full max-w-md transform overflow-hidden rounded-2xl glass-panel p-6 text-left align-middle shadow-xl transition-all border border-slate-700/50">
                <div className="flex justify-between items-start mb-6">
                  <Dialog.Title
                    as="h3"
                    className="text-lg font-medium leading-6 text-white flex items-center gap-2"
                  >
                    <Clock className="w-5 h-5 text-brand-available" />
                    Reserve Resource
                  </Dialog.Title>
                  <button
                    onClick={onClose}
                    className="text-slate-400 hover:text-white transition-colors"
                  >
                    <X className="w-5 h-5" />
                  </button>
                </div>

                <form onSubmit={handleSubmit}>
                  <div className="space-y-4">
                    <div>
                      <p className="text-sm text-slate-300 mb-2">
                        Reserving <span className="font-semibold text-white">{resourceName}</span>
                      </p>
                      <label className="block text-sm font-medium text-slate-400 mb-2">
                        Duration
                      </label>
                      <div className="grid grid-cols-2 gap-2 mb-4">
                        {DURATIONS.map((duration) => (
                          <button
                            key={duration.label}
                            type="button"
                            onClick={() => handleDurationSelect(duration.value)}
                            className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
                              !isCustomDuration && selectedDuration === duration.value
                                ? 'bg-brand-available text-white shadow-lg shadow-brand-available/20'
                                : 'bg-slate-800/50 text-slate-400 hover:bg-slate-800 hover:text-white border border-slate-700/50'
                            }`}
                          >
                            {duration.label}
                          </button>
                        ))}
                         <button
                            type="button"
                            onClick={() => handleDurationSelect('custom')}
                            className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
                              isCustomDuration
                                ? 'bg-brand-available text-white shadow-lg shadow-brand-available/20'
                                : 'bg-slate-800/50 text-slate-400 hover:bg-slate-800 hover:text-white border border-slate-700/50'
                            }`}
                          >
                            Custom
                          </button>
                      </div>

                      {isCustomDuration && (
                        <div className="bg-slate-900/50 p-4 rounded-lg border border-slate-700/50 animate-in fade-in slide-in-from-top-2 duration-200">
                            <div className="grid grid-cols-3 gap-4">
                                <div>
                                    <label className="block text-xs font-medium text-slate-400 mb-1">Days</label>
                                    <input
                                        type="number"
                                        min="0"
                                        value={customDays}
                                        onChange={(e) => setCustomDays(Math.max(0, parseInt(e.target.value) || 0))}
                                        className="w-full rounded bg-slate-800 border-slate-700 text-white text-sm px-2 py-1 focus:ring-1 focus:ring-brand-available focus:outline-none"
                                    />
                                </div>
                                <div>
                                    <label className="block text-xs font-medium text-slate-400 mb-1">Hours</label>
                                    <input
                                        type="number"
                                        min="0"
                                        value={customHours}
                                        onChange={(e) => setCustomHours(Math.max(0, parseInt(e.target.value) || 0))}
                                        className="w-full rounded bg-slate-800 border-slate-700 text-white text-sm px-2 py-1 focus:ring-1 focus:ring-brand-available focus:outline-none"
                                    />
                                </div>
                                <div>
                                    <label className="block text-xs font-medium text-slate-400 mb-1">Minutes</label>
                                    <input
                                        type="number"
                                        min="0"
                                        value={customMinutes}
                                        onChange={(e) => setCustomMinutes(Math.max(0, parseInt(e.target.value) || 0))}
                                        className="w-full rounded bg-slate-800 border-slate-700 text-white text-sm px-2 py-1 focus:ring-1 focus:ring-brand-available focus:outline-none"
                                    />
                                </div>
                            </div>
                        </div>
                      )}
                    </div>
                  </div>

                  <div className="mt-8 flex justify-end gap-3">
                    <button
                      type="button"
                      className="px-4 py-2 rounded-lg text-sm font-medium text-slate-300 hover:text-white transition-colors"
                      onClick={onClose}
                    >
                      Cancel
                    </button>
                    <button
                      type="submit"
                      className="px-4 py-2 rounded-lg text-sm font-medium bg-brand-available text-white hover:bg-brand-available/90 focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-available focus-visible:ring-offset-2 transition-all shadow-lg shadow-brand-available/20"
                    >
                      Confirm Reservation
                    </button>
                  </div>
                </form>
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition>
  );
}
