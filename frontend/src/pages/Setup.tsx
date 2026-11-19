import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import {
  useCreateInitialAdminMutation,
  useGetSetupStatusQuery,
  useLoginMutation,
} from "../store/api/auth";
import { useAppDispatch } from "../store/store";
import { setCredentials } from "../store/slices/authSlice";
import { useNotificationContext } from "../hooks/useNotification";

interface SetupError {
  data?: {
    error?: string;
  };
  message?: string;
}

const SetupPage = () => {
  const { t } = useTranslation(["pages", "common"]);
  const [name, setName] = useState<string>("");
  const [email, setEmail] = useState<string>("");
  const [password, setPassword] = useState<string>("");
  const [confirmPassword, setConfirmPassword] = useState<string>("");
  const [token, setToken] = useState<string>("");
  const [errorMessage, setErrorMessage] = useState<string>("");
  const [submitting, setSubmitting] = useState<boolean>(false);
  const [createInitialAdmin, { isLoading }] = useCreateInitialAdminMutation();
  const [login] = useLoginMutation();
  const { data: setupStatus, refetch: refetchSetupStatus } =
    useGetSetupStatusQuery();
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const { showNotification } = useNotificationContext();

  useEffect(() => {
    if (!submitting && setupStatus && !setupStatus.needsSetup) {
      navigate("/login", { replace: true });
    }
  }, [setupStatus, submitting, navigate]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMessage("");

    if (password !== confirmPassword) {
      setErrorMessage(t("pages:setup.passwordMismatch"));
      return;
    }

    setSubmitting(true);
    try {
      await createInitialAdmin({ name, email, password, token }).unwrap();
    } catch (err) {
      const error = err as SetupError;
      const errorMsg =
        error.data?.error ??
        error.message ??
        t("pages:setup.createFailed");
      setErrorMessage(errorMsg);
      showNotification("error", errorMsg);
      setSubmitting(false);
      await refetchSetupStatus();
      return;
    }

    await refetchSetupStatus();

    try {
      const response = await login({ email, password }).unwrap();
      dispatch(
        setCredentials({
          user: response.user.name,
          email: response.user.email,
          role: response.user.role,
          authProvider: response.user.authProvider ?? null,
        })
      );
      navigate("/", { replace: true });
    } catch {
      navigate("/login", { replace: true });
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-gradient-to-br from-brand-queued/5 via-transparent to-brand-available/5 pointer-events-none"></div>

      <div className="relative w-full max-w-md">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-light text-white tracking-tight">
            {t("common:appName")}
          </h1>
          <p className="text-slate-400 text-sm mt-1">Resource Management System</p>
        </div>

        <div className="glass-panel rounded-2xl p-8">
          <h2 className="text-2xl font-semibold text-white text-center mb-2">
            {t("pages:setup.title")}
          </h2>
          <p className="text-slate-400 text-center text-sm mb-6">
            {t("pages:setup.subtitle")}
          </p>

          {errorMessage && (
            <div className="mb-4 p-3 rounded-lg bg-brand-busy/10 border border-brand-busy/20">
              <p className="text-brand-busy text-sm text-center">{errorMessage}</p>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-5">
            <div>
              <label htmlFor="name" className="block text-sm font-medium text-slate-300 mb-2">
                {t("pages:setup.nameLabel")}
              </label>
              <input
                id="name"
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full px-4 py-3 bg-slate-800/50 border border-slate-700 rounded-lg text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-brand-queued/50 focus:border-brand-queued/50 transition-all duration-200"
                placeholder={t("pages:setup.namePlaceholder")}
                required
              />
            </div>

            <div>
              <label htmlFor="email" className="block text-sm font-medium text-slate-300 mb-2">
                {t("pages:setup.emailLabel")}
              </label>
              <input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full px-4 py-3 bg-slate-800/50 border border-slate-700 rounded-lg text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-brand-queued/50 focus:border-brand-queued/50 transition-all duration-200"
                placeholder={t("pages:setup.emailPlaceholder")}
                required
              />
            </div>

            <div>
              <label htmlFor="password" className="block text-sm font-medium text-slate-300 mb-2">
                {t("pages:setup.passwordLabel")}
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-4 py-3 bg-slate-800/50 border border-slate-700 rounded-lg text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-brand-queued/50 focus:border-brand-queued/50 transition-all duration-200"
                placeholder={t("pages:setup.passwordPlaceholder")}
                required
              />
            </div>

            <div>
              <label htmlFor="confirmPassword" className="block text-sm font-medium text-slate-300 mb-2">
                {t("pages:setup.confirmPasswordLabel")}
              </label>
              <input
                id="confirmPassword"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className="w-full px-4 py-3 bg-slate-800/50 border border-slate-700 rounded-lg text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-brand-queued/50 focus:border-brand-queued/50 transition-all duration-200"
                placeholder={t("pages:setup.confirmPasswordPlaceholder")}
                required
              />
            </div>

            <div>
              <label htmlFor="token" className="block text-sm font-medium text-slate-300 mb-2">
                {t("pages:setup.tokenLabel")}
              </label>
              <input
                id="token"
                type="text"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                className="w-full px-4 py-3 bg-slate-800/50 border border-slate-700 rounded-lg text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-brand-queued/50 focus:border-brand-queued/50 transition-all duration-200 font-mono text-sm"
                placeholder={t("pages:setup.tokenPlaceholder")}
                autoComplete="off"
                required
              />
              <p className="text-slate-500 text-xs mt-2">
                {t("pages:setup.tokenHelp")}
              </p>
            </div>

            <button
              type="submit"
              disabled={isLoading}
              className="w-full py-3 bg-brand-queued hover:bg-brand-queued/90 disabled:opacity-50 disabled:cursor-not-allowed text-white font-semibold rounded-lg transition-all duration-200 shadow-lg shadow-brand-queued/20 hover:shadow-brand-queued/30"
            >
              {isLoading
                ? t("common:loading")
                : t("pages:setup.createAccount")}
            </button>
          </form>
        </div>

        <p className="text-center text-slate-600 text-xs mt-6">
          {t("pages:login.footer", { year: new Date().getFullYear() })}
        </p>
      </div>
    </div>
  );
};

export default SetupPage;
