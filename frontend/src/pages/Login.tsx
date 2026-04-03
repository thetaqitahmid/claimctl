import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useLoginMutation, useLoginLDAPMutation } from "../store/api/auth";
import { useAppDispatch } from "../store/store";
import { setCredentials } from "../store/slices/authSlice";
import { useNotificationContext } from "../hooks/useNotification";

interface LoginError {
  data?: {
    error?: string;
  };
  message?: string;
}


const LoginPage = () => {
  const { t } = useTranslation(["pages", "common"]);
  const [email, setEmail] = useState<string>("");
  const [password, setPassword] = useState<string>("");
  const [errorMessage, setErrorMessage] = useState<string>("");
  const [isLDAP, setIsLDAP] = useState<boolean>(false);
  const [createLogin] = useLoginMutation();
  const [createLDAPLogin] = useLoginLDAPMutation();
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const { showNotification } = useNotificationContext();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMessage("");

    try {
      const loginFn = isLDAP ? createLDAPLogin : createLogin;
      const response = await loginFn({ email, password }).unwrap();
      dispatch(
        setCredentials({
          user: response.user.name,
          email: response.user.email,

          role: response.user.role,
        })
      );
      navigate("/", { replace: true });
    } catch (err) {
      const error = err as LoginError;
      console.error("Login failed:", err);
      let errorMsg = t("pages:login.invalidCredentials");

      if (error.data && error.data.error) {
          if (error.data.error.includes("Account is locked")) {
              errorMsg = "Account is locked after too many failed attempts.";
          } else {
             errorMsg = error.data.error;
          }
      } else if (error.message) {
           // Fallback if err.data is not present but err.message is
           if (error.message.includes("Account is locked")) {
              errorMsg = "Account is locked after too many failed attempts.";
           }
      }

      setErrorMessage(errorMsg);
      showNotification('error', `Login failed: ${errorMsg}`);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      {/* Background gradient overlay */}
      <div className="absolute inset-0 bg-gradient-to-br from-brand-queued/5 via-transparent to-brand-available/5 pointer-events-none"></div>

      <div className="relative w-full max-w-md">
        {/* Logo */}
        <div className="text-center mb-8">
          <h1 className="text-3xl font-light text-white tracking-tight">
            {t("common:appName")}
          </h1>
          <p className="text-slate-400 text-sm mt-1">Resource Management System</p>
        </div>

        {/* Login Card */}
        <div className="glass-panel rounded-2xl p-8">
          <h2 className="text-2xl font-semibold text-white text-center mb-2">
            {isLDAP ? t("pages:login.ldapTitle") : t("pages:login.title")}
          </h2>
          <p className="text-slate-400 text-center text-sm mb-6">
            {isLDAP ? t("pages:login.ldapSubtitle") : t("pages:login.subtitle")}
          </p>

          {/* LDAP Toggle */}
          <div className="flex justify-center mb-6">
            <label className="inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                className="sr-only peer"
                checked={isLDAP}
                onChange={() => setIsLDAP(!isLDAP)}
              />
              <div className="relative w-11 h-6 bg-slate-700 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-brand-queued/50 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-transparent after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-slate-300 after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-brand-queued"></div>
              <span className="ms-3 text-sm font-medium text-slate-300">
                {t("pages:login.ldapToggle")}
              </span>
            </label>
          </div>

          {/* Error Message */}
          {errorMessage && (
            <div className="mb-4 p-3 rounded-lg bg-brand-busy/10 border border-brand-busy/20">
              <p className="text-brand-busy text-sm text-center">{errorMessage}</p>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-5">
            <div>
              <label htmlFor="email" className="block text-sm font-medium text-slate-300 mb-2">
                {t("pages:login.emailLabel")}
              </label>
              <input
                id="email"
                type="email"
                name="username"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full px-4 py-3 bg-slate-800/50 border border-slate-700 rounded-lg text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-brand-queued/50 focus:border-brand-queued/50 transition-all duration-200"
                placeholder={t("pages:login.emailPlaceholder")}
              />
            </div>

            <div>
              <label htmlFor="password" id="password-label" className="block text-sm font-medium text-slate-300 mb-2">
                {t("pages:login.passwordLabel")}
              </label>
              <input
                id="password"
                type="password"
                name="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-4 py-3 bg-slate-800/50 border border-slate-700 rounded-lg text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-brand-queued/50 focus:border-brand-queued/50 transition-all duration-200"
                placeholder={t("pages:login.passwordPlaceholder")}
              />
            </div>

            <button
              type="submit"
              className="w-full py-3 bg-brand-queued hover:bg-brand-queued/90 text-white font-semibold rounded-lg transition-all duration-200 shadow-lg shadow-brand-queued/20 hover:shadow-brand-queued/30"
            >
              {isLDAP ? t("pages:login.ldapSignIn") : t("pages:login.signIn")}
            </button>

            {!isLDAP && (
              <div className="relative">
                <div className="absolute inset-0 flex items-center">
                  <div className="w-full border-t border-slate-700"></div>
                </div>
                <div className="relative flex justify-center text-sm">
                  <span className="px-2 bg-slate-800 text-slate-400">{t("pages:login.orContinueWith")}</span>
                </div>
              </div>
            )}

            {!isLDAP && (
              <button
                type="button"
                onClick={() => (window.location.href = "/api/auth/oidc/login")}
                className="w-full py-3 bg-white text-slate-900 font-semibold rounded-lg hover:bg-slate-100 transition-all duration-200 flex items-center justify-center gap-2"
              >
                <svg className="w-5 h-5" viewBox="0 0 24 24">
                  <path
                    fill="currentColor"
                    d="M12.545,10.239v3.821h5.445c-0.712,2.315-2.647,3.972-5.445,3.972c-3.332,0-6.033-2.701-6.033-6.032s2.701-6.032,6.033-6.032c1.498,0,2.866,0.549,3.921,1.453l2.814-2.814C17.503,2.988,15.139,2,12.545,2C7.021,2,2.543,6.477,2.543,12s4.478,10,10.002,10c8.396,0,10.249-7.85,9.426-11.748L12.545,10.239z"
                  />
                </svg>
                {t("pages:login.ssoSignIn")}
              </button>
            )}
          </form>
        </div>

        {/* Footer */}
        <p className="text-center text-slate-600 text-xs mt-6">
          {t("pages:login.footer", { year: new Date().getFullYear() })}
        </p>
      </div>
    </div>
  );
};

export default LoginPage;
