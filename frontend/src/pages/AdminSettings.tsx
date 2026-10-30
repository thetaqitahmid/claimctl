import { useState } from "react";
import { Navigate, useParams } from "react-router-dom";
import {
  useGetSettingsQuery,
  useUpdateSettingMutation,
  AppSetting,
} from "../store/api/settings";
import UnsavedChangesGuard from "../components/ui/UnsavedChangesGuard";

const labels: Record<string, string> = {
  oidc_issuer: "Issuer URL",
  oidc_client_id: "Client ID",
  oidc_client_secret: "Client secret",
  oidc_redirect_url: "Redirect URL",
  oidc_scopes: "Scopes",
  oidc_admin_group: "Administrator group",
  oidc_groups_claim: "Groups claim",
  smtp_host: "SMTP host",
  smtp_port: "SMTP port",
  smtp_user: "SMTP username",
  smtp_pass: "SMTP password",
  smtp_from: "Sender email address",
  slack_bot_token: "Slack bot token",
  jwt_private_key: "JWT private key",
  jwt_public_key: "JWT public key",
};

const SettingsForm = ({
  settings,
  title,
  description,
}: {
  settings: AppSetting[];
  title: string;
  description: string;
}) => {
  const [saved, setSaved] = useState(() =>
    Object.fromEntries(
      settings.map((s) => [s.key, s.is_secret ? "" : s.value]),
    ),
  );
  const [values, setValues] = useState(saved);
  const [replacing, setReplacing] = useState<Record<string, boolean>>({});
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [status, setStatus] = useState("");
  const [saving, setSaving] = useState(false);
  const [configured, setConfigured] = useState(() =>
    Object.fromEntries(settings.map((s) => [s.key, Boolean(s.value)])),
  );
  const [updateSetting] = useUpdateSettingMutation();
  const changed = settings.filter((s) => values[s.key] !== saved[s.key]);

  const discard = () => {
    setValues(saved);
    setReplacing({});
    setErrors({});
    setStatus("");
  };

  const save = async (event: React.FormEvent) => {
    event.preventDefault();
    setSaving(true);
    setErrors({});
    setStatus("");
    const failures: Record<string, string> = {};
    for (const setting of changed) {
      try {
        await updateSetting({
          ...setting,
          value: values[setting.key],
        }).unwrap();
        const next = setting.is_secret ? "" : values[setting.key];
        setSaved((previous) => ({ ...previous, [setting.key]: next }));
        setValues((previous) => ({ ...previous, [setting.key]: next }));
        setConfigured((previous) => ({
          ...previous,
          [setting.key]: Boolean(values[setting.key]),
        }));
        setReplacing((previous) => ({ ...previous, [setting.key]: false }));
      } catch (error) {
        failures[setting.key] =
          (error as { data?: { error?: string } })?.data?.error ||
          "Could not save this setting. Try again.";
      }
    }
    setErrors(failures);
    setStatus(
      Object.keys(failures).length
        ? "Some changes could not be saved. Successful changes were saved; review the fields below and retry the remaining changes."
        : "Changes saved successfully.",
    );
    setSaving(false);
  };

  const groups =
    title === "Authentication"
      ? [
          {
            title: "Single sign on (OpenID Connect)",
            keys: settings.filter(
              (s) =>
                s.key.startsWith("oidc_") &&
                ![
                  "oidc_scopes",
                  "oidc_admin_group",
                  "oidc_groups_claim",
                ].includes(s.key),
            ),
            advanced: false,
          },
          {
            title: "Advanced authentication options",
            keys: settings.filter(
              (s) =>
                !s.key.startsWith("oidc_") ||
                [
                  "oidc_scopes",
                  "oidc_admin_group",
                  "oidc_groups_claim",
                ].includes(s.key),
            ),
            advanced: true,
          },
        ]
      : [
          {
            title: "Email delivery",
            keys: settings.filter((s) => s.key.startsWith("smtp_")),
            advanced: false,
          },
          {
            title: "Other notification integrations",
            keys: settings.filter((s) => !s.key.startsWith("smtp_")),
            advanced: false,
          },
        ];

  const renderField = (setting: AppSetting) => {
    const id = `setting-${setting.key}`;
    const label = labels[setting.key] || setting.key.replace(/_/g, " ");
    const secret = setting.is_secret;
    const multiline = setting.key.startsWith("jwt_");
    const type = secret
      ? "password"
      : setting.key === "smtp_port"
        ? "number"
        : setting.key === "smtp_from"
          ? "email"
          : setting.key.endsWith("_url") || setting.key === "oidc_issuer"
            ? "url"
            : "text";
    const props = {
      id,
      value: values[setting.key] || "",
      className: "admin-input",
      disabled: saving,
      onChange: (
        event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
      ) => {
        setValues((previous) => ({
          ...previous,
          [setting.key]: event.target.value,
        }));
        setStatus("");
      },
      "aria-describedby": `${id}-help${errors[setting.key] ? ` ${id}-error` : ""}`,
      "aria-invalid": Boolean(errors[setting.key]),
    };
    return (
      <div key={setting.key} className="max-w-2xl space-y-2">
        <label
          htmlFor={id}
          className="block text-sm font-medium text-slate-100"
        >
          {label}
        </label>
        <p id={`${id}-help`} className="text-sm text-slate-400">
          {setting.key === "oidc_redirect_url"
            ? "Optional. When empty, uses the application's base URL followed by /api/auth/oidc/callback."
            : setting.description}
        </p>
        {secret && (
          <div className="flex items-center gap-4 text-sm">
            <span className="text-slate-300">
              {configured[setting.key] ? "Configured" : "Not configured"}
            </span>
            <button
              type="button"
              className="text-cyan-300 underline"
              disabled={saving}
              onClick={() => {
                setReplacing((previous) => ({
                  ...previous,
                  [setting.key]: !previous[setting.key],
                }));
                setValues((previous) => ({ ...previous, [setting.key]: "" }));
              }}
            >
              {replacing[setting.key]
                ? "Cancel replacement"
                : configured[setting.key]
                  ? "Replace secret"
                  : "Set secret"}
            </button>
          </div>
        )}
        {(!secret || replacing[setting.key]) &&
          (multiline ? (
            <textarea
              {...props}
              rows={6}
              autoComplete="off"
              required={secret}
            />
          ) : (
            <input
              {...props}
              type={type}
              min={type === "number" ? 1 : undefined}
              max={type === "number" ? 65535 : undefined}
              step={type === "number" ? 1 : undefined}
              autoComplete={secret ? "new-password" : "off"}
              required={secret}
            />
          ))}
        {errors[setting.key] && (
          <p id={`${id}-error`} className="text-sm text-red-300" role="alert">
            {errors[setting.key]}
          </p>
        )}
      </div>
    );
  };

  return (
    <form onSubmit={save} className="space-y-6">
      <UnsavedChangesGuard
        dirty={changed.length > 0 || saving}
        pending={saving}
      />
      <header>
        <p className="text-sm text-slate-400 mb-2">
          Administration / System Settings
        </p>
        <h1 className="text-3xl font-semibold text-white">{title}</h1>
        <p className="text-slate-400 mt-2">{description}</p>
      </header>
      {settings.length === 0 ? (
        <div className="admin-panel p-6">
          No settings are available in this category.
        </div>
      ) : (
        groups
          .filter((g) => g.keys.length > 0)
          .map((group) =>
            group.advanced ? (
              <details
                key={group.title}
                className="admin-panel p-6"
                open={group.keys.some((s) => errors[s.key]) || undefined}
              >
                <summary className="cursor-pointer font-semibold text-white">
                  {group.title}
                </summary>
                <p className="text-sm text-amber-200 my-4">
                  Signing keys are sensitive. Changes are saved individually,
                  not as an atomic key pair. Keep a recovery path before
                  changing authentication configuration.
                </p>
                <div className="space-y-6 mt-6">
                  {group.keys.map(renderField)}
                </div>
              </details>
            ) : (
              <section key={group.title} className="admin-panel p-6">
                <h2 className="text-lg font-semibold text-white mb-6">
                  {group.title}
                </h2>
                <div className="space-y-6">{group.keys.map(renderField)}</div>
              </section>
            ),
          )
      )}
      <div className="sticky bottom-0 admin-panel p-4 flex flex-wrap items-center gap-3">
        <p className="text-sm text-slate-300 mr-auto" role="status">
          {status ||
            (changed.length
              ? `${changed.length} unsaved ${changed.length === 1 ? "change" : "changes"}`
              : "No unsaved changes")}
        </p>
        <button
          type="button"
          className="btn-ghost"
          disabled={!changed.length || saving}
          onClick={discard}
        >
          Discard changes
        </button>
        <button
          type="submit"
          className="btn-primary"
          disabled={!changed.length || saving}
        >
          {saving ? "Saving…" : "Save changes"}
        </button>
      </div>
    </form>
  );
};

const AdminSettings = () => {
  const { section } = useParams();
  const {
    data: settings = [],
    isLoading,
    error,
    refetch,
  } = useGetSettingsQuery();
  if (!["authentication", "notifications"].includes(section || ""))
    return <Navigate to="/admin/settings/authentication" replace />;
  if (isLoading)
    return (
      <p role="status" className="p-6">
        Loading settings…
      </p>
    );
  if (error && settings.length === 0)
    return (
      <div role="alert" className="admin-panel p-6">
        <p>Could not load settings. Your configuration has not been changed.</p>
        <button className="btn-ghost mt-4" onClick={() => refetch()}>
          Retry
        </button>
      </div>
    );
  return (
    <>
      {error && (
        <div role="alert" className="admin-panel p-4 mb-6">
          <p>Could not refresh settings. Your current edits have been kept.</p>
          <button className="btn-ghost mt-3" onClick={() => refetch()}>
            Retry refresh
          </button>
        </div>
      )}
      <SettingsForm
        key={section}
        settings={settings.filter(
          (s) =>
            s.category ===
            (section === "authentication" ? "auth" : "notification"),
        )}
        title={
          section === "authentication" ? "Authentication" : "Notifications"
        }
        description={
          section === "authentication"
            ? "Configure sign in and administrator access."
            : "Configure email delivery and notification integrations."
        }
      />
    </>
  );
};

export default AdminSettings;
