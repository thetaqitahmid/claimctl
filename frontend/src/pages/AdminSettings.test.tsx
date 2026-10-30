import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import AdminSettings from "./AdminSettings";

const mocks = vi.hoisted(() => ({
  query: vi.fn(),
  update: vi.fn(),
  refetch: vi.fn(),
}));
vi.mock("../store/api/settings", () => ({
  useGetSettingsQuery: (...args: unknown[]) => mocks.query(...args),
  useUpdateSettingMutation: () => [mocks.update],
}));

const settings = [
  {
    key: "oidc_issuer",
    value: "https://identity.example.com",
    category: "auth",
    description: "Provider URL",
    is_secret: false,
  },
  {
    key: "oidc_client_id",
    value: "claimctl",
    category: "auth",
    description: "Application identifier",
    is_secret: false,
  },
  {
    key: "oidc_client_secret",
    value: "********",
    category: "auth",
    description: "Provider credential",
    is_secret: true,
  },
  {
    key: "oidc_scopes",
    value: "openid profile email",
    category: "auth",
    description: "Scopes",
    is_secret: false,
  },
  {
    key: "smtp_host",
    value: "mail.example.com",
    category: "notification",
    description: "Mail server",
    is_secret: false,
  },
  {
    key: "smtp_port",
    value: "587",
    category: "notification",
    description: "Mail port",
    is_secret: false,
  },
];

function setup(section = "authentication") {
  const router = createMemoryRouter(
    [
      { path: "/admin/settings/:section", element: <AdminSettings /> },
      { path: "/elsewhere", element: <h1>Elsewhere</h1> },
    ],
    { initialEntries: [`/admin/settings/${section}`] },
  );
  render(<RouterProvider router={router} />);
  return { user: userEvent.setup(), router };
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.query.mockReturnValue({
    data: settings,
    isLoading: false,
    error: null,
    refetch: mocks.refetch,
  });
  mocks.update.mockImplementation(() => ({
    unwrap: () => Promise.resolve({}),
  }));
});

describe("AdminSettings", () => {
  it("uses friendly grouped fields and does not expose secret values", () => {
    setup();
    expect(
      screen.getByRole("heading", { name: "Authentication" }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText("Client ID")).toHaveValue("claimctl");
    expect(screen.queryByText("oidc_client_id")).not.toBeInTheDocument();
    expect(screen.getByText("Configured")).toBeInTheDocument();
    expect(screen.queryByDisplayValue("********")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save changes" })).toBeDisabled();
  });

  it("saves only changed fields without resubmitting masked credentials", async () => {
    const { user } = setup();
    await user.clear(screen.getByLabelText("Client ID"));
    await user.type(screen.getByLabelText("Client ID"), "new-client");
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    expect(
      await screen.findByText("Changes saved successfully."),
    ).toBeInTheDocument();
    expect(mocks.update).toHaveBeenCalledOnce();
    expect(mocks.update).toHaveBeenCalledWith(
      expect.objectContaining({ key: "oidc_client_id", value: "new-client" }),
    );
    expect(screen.getByRole("button", { name: "Save changes" })).toBeDisabled();
  });

  it("requires explicit secret replacement and clears it from the form after saving", async () => {
    const { user } = setup();
    await user.click(screen.getByRole("button", { name: "Replace secret" }));
    const input = screen.getByLabelText("Client secret");
    expect(input).toHaveAttribute("type", "password");
    expect(input).toHaveValue("");
    await user.type(input, "new-secret");
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    expect(
      await screen.findByText("Changes saved successfully."),
    ).toBeInTheDocument();
    expect(mocks.update).toHaveBeenCalledWith(
      expect.objectContaining({
        key: "oidc_client_secret",
        value: "new-secret",
      }),
    );
    expect(screen.queryByDisplayValue("new-secret")).not.toBeInTheDocument();
  });

  it("retains failed fields and retries only unsaved values after partial success", async () => {
    mocks.update.mockImplementation((setting: { key: string }) => ({
      unwrap: () =>
        setting.key === "oidc_client_id"
          ? Promise.reject({ data: { error: "Invalid client" } })
          : Promise.resolve({}),
    }));
    const { user } = setup();
    await user.clear(screen.getByLabelText("Issuer URL"));
    await user.type(
      screen.getByLabelText("Issuer URL"),
      "https://new.example.com",
    );
    await user.clear(screen.getByLabelText("Client ID"));
    await user.type(screen.getByLabelText("Client ID"), "new-client");
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Invalid client",
    );
    expect(screen.getByRole("status")).toHaveTextContent(
      "Successful changes were saved",
    );
    expect(screen.getByLabelText("Client ID")).toHaveValue("new-client");
    mocks.update
      .mockClear()
      .mockImplementation(() => ({ unwrap: () => Promise.resolve({}) }));
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    await screen.findByText("Changes saved successfully.");
    expect(mocks.update).toHaveBeenCalledOnce();
    expect(mocks.update).toHaveBeenCalledWith(
      expect.objectContaining({ key: "oidc_client_id" }),
    );
  });

  it("discards edits and protects navigation until explicitly discarded", async () => {
    const { user, router } = setup();
    await user.type(screen.getByLabelText("Client ID"), "-edited");
    await act(async () => {
      await router.navigate("/elsewhere");
    });
    expect(
      await screen.findByRole("dialog", { name: "Discard unsaved changes?" }),
    ).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Keep editing" }));
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
    expect(screen.getByLabelText("Client ID")).toHaveValue("claimctl-edited");
    await act(async () => {
      await router.navigate("/elsewhere");
    });
    await user.click(screen.getByRole("button", { name: "Discard and leave" }));
    expect(
      await screen.findByRole("heading", { name: "Elsewhere" }),
    ).toBeInTheDocument();
  });

  it("prevents leaving or submitting again while saving", async () => {
    let finish!: () => void;
    mocks.update.mockReturnValue({
      unwrap: () =>
        new Promise<void>((resolve) => {
          finish = resolve;
        }),
    });
    const { user, router } = setup();
    await user.type(screen.getByLabelText("Client ID"), "-edited");
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    expect(screen.getByRole("button", { name: "Saving…" })).toBeDisabled();
    await act(async () => {
      await router.navigate("/elsewhere");
    });
    expect(
      await screen.findByRole("dialog", { name: "Operation in progress" }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Discard and leave" }),
    ).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Keep waiting" }));
    await act(async () => {
      finish();
    });
    await screen.findByText("Changes saved successfully.");
    expect(router.state.location.pathname).toBe(
      "/admin/settings/authentication",
    );
    expect(mocks.update).toHaveBeenCalledOnce();
  });

  it("restores saved values when discarding edits", async () => {
    const { user } = setup();
    await user.type(screen.getByLabelText("Client ID"), "-edited");
    await user.click(screen.getByRole("button", { name: "Discard changes" }));
    expect(screen.getByLabelText("Client ID")).toHaveValue("claimctl");
    expect(mocks.update).not.toHaveBeenCalled();
  });

  it("warns on browser unload only when dirty", async () => {
    const listener = vi.spyOn(window, "addEventListener");
    const { user } = setup();
    const callGuard = (event: Event) => {
      const registrations = listener.mock.calls.filter(
        ([type]) => type === "beforeunload",
      );
      const guard = registrations[registrations.length - 1][1] as EventListener;
      guard(event);
    };
    const clean = new Event("beforeunload", { cancelable: true });
    callGuard(clean);
    expect(clean.defaultPrevented).toBe(false);
    await user.type(screen.getByLabelText("Client ID"), "-edited");
    const dirty = new Event("beforeunload", { cancelable: true });
    callGuard(dirty);
    expect(dirty.defaultPrevented).toBe(true);
    listener.mockRestore();
  });

  it("uses numeric constraints for SMTP port", async () => {
    const { user } = setup("notifications");
    const port = screen.getByLabelText("SMTP port");
    expect(port).toHaveAttribute("type", "number");
    await user.clear(port);
    await user.type(port, "70000");
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    expect(mocks.update).not.toHaveBeenCalled();
  });

  it("renders errors as errors, not empty configuration", async () => {
    mocks.query.mockReturnValue({
      data: [],
      error: { status: 500 },
      isLoading: false,
      refetch: mocks.refetch,
    });
    const { user } = setup();
    expect(screen.getByRole("alert")).toHaveTextContent(
      "Could not load settings",
    );
    await user.click(screen.getByRole("button", { name: "Retry" }));
    expect(mocks.refetch).toHaveBeenCalledOnce();
    expect(
      screen.queryByText("No settings are available"),
    ).not.toBeInTheDocument();
  });

  it("does not restore the removed application backup interface", async () => {
    const { router } = setup("backup");
    expect(await screen.findByRole("heading", { name: "Authentication" })).toBeInTheDocument();
    expect(router.state.location.pathname).toBe("/admin/settings/authentication");
    expect(screen.queryByRole("button", { name: /backup|restore/i })).not.toBeInTheDocument();
    expect(document.querySelector('input[type="file"]')).not.toBeInTheDocument();
  });
});
