import { Fragment, useRef, useState } from "react";
import { Dialog, Transition } from "@headlessui/react";
import { useTranslation } from "react-i18next";
import { Clock, X } from "lucide-react";
import { ReservationOptions } from "../types";
import { useGetReservationPolicyQuery } from "../store/api/reservations";

interface ReservationModalProps {
  isOpen: boolean;
  onClose: () => void;
  resourceName: string;
  onConfirm: (options: ReservationOptions) => Promise<void>;
}

const PRESETS = ["30m", "1h", "4h", "24h"];

function durationSeconds(value: string) {
  const units: Record<string, number> = {
    d: 86400, h: 3600, m: 60, s: 1, ms: 0.001,
    us: 0.000001, "µs": 0.000001, ns: 0.000000001,
  };
  let seconds = 0;
  let consumed = "";
  for (const match of value.matchAll(/(\d+(?:\.\d+)?)(d|h|ms|m|us|µs|ns|s)/g)) {
    consumed += match[0];
    seconds += Number(match[1]) * units[match[2]];
  }
  return consumed === value && Number.isFinite(seconds) ? seconds : NaN;
}

export default function ReservationModal({
  isOpen, onClose, resourceName, onConfirm,
}: ReservationModalProps) {
  const { t } = useTranslation(["components", "common"]);
  const { data: policy, isFetching: isPolicyFetching, error: policyError, refetch } =
    useGetReservationPolicyQuery(undefined, { skip: !isOpen });
  const [selection, setSelection] = useState("default");
  const [custom, setCustom] = useState({ days: "0", hours: "0", minutes: "0" });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const submitting = useRef(false);
  const [error, setError] = useState("");
  const maxSeconds = policy ? durationSeconds(policy.maxDuration) : 0;
  const customSeconds = Number(custom.days) * 86400 + Number(custom.hours) * 3600 + Number(custom.minutes) * 60;
  const validCustom = Object.values(custom).every((value) => /^\d+$/.test(value) && Number.isSafeInteger(Number(value))) &&
    customSeconds > 0 && customSeconds <= maxSeconds;
  const canSubmit = !!policy && !policyError && !isPolicyFetching && !isSubmitting &&
    (selection !== "custom" || validCustom) &&
    (selection !== "indefinite" || policy.allowIndefinite);

  const close = () => { if (!submitting.current) onClose(); };
  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!canSubmit || submitting.current) return;
    submitting.current = true;
    setIsSubmitting(true);
    setError("");
    const options: ReservationOptions = selection === "indefinite"
      ? { indefinite: true }
      : selection === "default" ? {}
      : { duration: selection === "custom" ? `${customSeconds}s` : selection };
    try {
      await onConfirm(options);
      onClose();
    } catch (cause) {
      const status = (cause as { status?: number | string })?.status;
      const key = status === 409 ? "conflict" : status === 403 ? "forbidden"
        : status === 400 ? "invalid" : "failed";
      setError(t(`components:reservationModal.errors.${key}`));
    } finally {
      submitting.current = false;
      setIsSubmitting(false);
    }
  };

  return (
    <Transition appear show={isOpen} as={Fragment}>
      <Dialog as="div" className="relative z-50" onClose={close}>
        <div className="fixed inset-0 bg-black/80" aria-hidden="true" />
        <div className="fixed inset-0 overflow-y-auto">
          <div className="flex min-h-full items-center justify-center p-4">
            <Dialog.Panel className="w-full max-w-md rounded-2xl glass-panel p-6 shadow-xl border border-slate-700/50">
              <div className="flex justify-between items-start mb-6 gap-4">
                <Dialog.Title as="h2" className="text-lg font-medium text-white flex items-center gap-2">
                  <Clock className="w-5 h-5 shrink-0 text-brand-available" aria-hidden="true" />
                  {t("components:reservationModal.title", { resourceName })}
                </Dialog.Title>
                <button type="button" onClick={close} disabled={isSubmitting} aria-label={t("common:close")}
                  className="p-2 text-slate-400 hover:text-white disabled:opacity-50">
                  <X className="w-5 h-5" aria-hidden="true" />
                </button>
              </div>
              {policyError ? (
                <div role="alert" className="mb-4 text-red-400">
                  <p>{t("components:reservationModal.policyError")}</p>
                  <button type="button" onClick={() => refetch()} disabled={isPolicyFetching} className="underline p-2">
                    {t("common:refresh")}
                  </button>
                </div>
              ) : !policy ? <p role="status">{t("common:loading")}</p> : (
                <Dialog.Description className="mb-4 text-sm text-slate-300">
                  {t("components:reservationModal.policy", {
                    defaultDuration: policy.defaultDuration, maxDuration: policy.maxDuration,
                    maxLiveClaims: policy.maxLiveClaims,
                  })}
                </Dialog.Description>
              )}
              <form onSubmit={handleSubmit} aria-busy={isSubmitting}>
                <fieldset disabled={isSubmitting || !policy || !!policyError || isPolicyFetching} className="disabled:opacity-60">
                  <legend className="text-sm font-medium text-slate-400 mb-2">{t("components:reservationModal.duration")}</legend>
                  <div className="grid grid-cols-2 gap-2 mb-4">
                    {["default", ...PRESETS.filter((value) => durationSeconds(value) <= maxSeconds),
                      ...(policy?.allowIndefinite ? ["indefinite"] : []), "custom"].map((value) => (
                      <label key={value} className={`flex items-center gap-2 rounded-lg p-3 text-sm cursor-pointer border ${
                        selection === value ? "border-cyan-500 bg-cyan-600/20 text-white" : "border-slate-700 text-slate-300"
                      }`}>
                        <input type="radio" name="reservation-duration" value={value} checked={selection === value}
                          onChange={() => { setSelection(value); setError(""); }} />
                        {PRESETS.includes(value) ? t(`components:reservationModal.presets.${value}`)
                          : t(`components:reservationModal.${value}`)}
                      </label>
                    ))}
                  </div>
                  {selection === "custom" && (
                    <div className="mb-4">
                      <div className="grid grid-cols-3 gap-3">
                        {(["days", "hours", "minutes"] as const).map((unit) => (
                          <label key={unit} className="text-xs text-slate-400">
                            {t(`components:reservationModal.${unit}`)}
                            <input type="number" min="0" step="1" required value={custom[unit]}
                              aria-describedby={!validCustom ? "reservation-duration-error" : undefined}
                              aria-invalid={!validCustom}
                              onChange={(event) => setCustom({ ...custom, [unit]: event.target.value })}
                              className="mt-1 w-full rounded bg-slate-800 border border-slate-700 text-white text-sm px-2 py-2" />
                          </label>
                        ))}
                      </div>
                      {!validCustom && <p id="reservation-duration-error" className="text-sm text-amber-400 mt-2">
                        {t("components:reservationModal.invalidDuration", { maxDuration: policy?.maxDuration })}
                      </p>}
                    </div>
                  )}
                </fieldset>
                {error && <p role="alert" className="text-sm text-red-400 mb-4">{error}</p>}
                <div className="mt-6 flex justify-end gap-3">
                  <button type="button" onClick={close} disabled={isSubmitting} className="px-4 py-3 text-sm text-slate-300 disabled:opacity-50">
                    {t("common:cancel")}
                  </button>
                  <button type="submit" disabled={!canSubmit}
                    className="px-4 py-3 rounded-lg text-sm font-medium bg-brand-available text-white disabled:opacity-50 disabled:cursor-not-allowed">
                    {t(isSubmitting ? "components:reservationModal.submitting" : "components:reservationModal.confirm")}
                  </button>
                </div>
              </form>
            </Dialog.Panel>
          </div>
        </div>
      </Dialog>
    </Transition>
  );
}
