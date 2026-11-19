import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { render } from "../test/utils";
import SetupPage from "./Setup";
import * as authApi from "../store/api/auth";
import * as notificationHook from "../hooks/useNotification";

vi.mock("../store/api/auth", async () => {
  const actual = await vi.importActual("../store/api/auth");
  return {
    ...actual,
    useCreateInitialAdminMutation: vi.fn(),
    useGetSetupStatusQuery: vi.fn(),
    useLoginMutation: vi.fn(),
  };
});

vi.mock("../hooks/useNotification", () => ({
  useNotificationContext: vi.fn(),
}));

describe("SetupPage", () => {
  const mockCreateAdmin = vi.fn();
  const mockLogin = vi.fn();
  const mockRefetchSetupStatus = vi.fn();
  const mockShowNotification = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();

    vi.mocked(authApi.useCreateInitialAdminMutation).mockReturnValue([
      mockCreateAdmin,
      { isLoading: false },
    ] as unknown as ReturnType<typeof authApi.useCreateInitialAdminMutation>);
    vi.mocked(authApi.useGetSetupStatusQuery).mockReturnValue({
      refetch: mockRefetchSetupStatus,
    } as unknown as ReturnType<typeof authApi.useGetSetupStatusQuery>);
    vi.mocked(authApi.useLoginMutation).mockReturnValue([
      mockLogin,
      { isLoading: false },
    ] as unknown as ReturnType<typeof authApi.useLoginMutation>);
    vi.mocked(notificationHook.useNotificationContext).mockReturnValue({
      showNotification: mockShowNotification,
    } as unknown as ReturnType<typeof notificationHook.useNotificationContext>);
  });

  it("renders the setup form", () => {
    render(<SetupPage />);
    expect(screen.getByText("title")).toBeInTheDocument();
    expect(screen.getByLabelText("nameLabel")).toBeInTheDocument();
    expect(screen.getByLabelText("emailLabel")).toBeInTheDocument();
    expect(screen.getByLabelText("passwordLabel")).toBeInTheDocument();
    expect(screen.getByLabelText("confirmPasswordLabel")).toBeInTheDocument();
    expect(screen.getByLabelText("tokenLabel")).toBeInTheDocument();
  });

  it("blocks submission when passwords differ", async () => {
    render(<SetupPage />);

    fireEvent.change(screen.getByLabelText("nameLabel"), { target: { value: "Admin" } });
    fireEvent.change(screen.getByLabelText("emailLabel"), { target: { value: "admin@example.com" } });
    fireEvent.change(screen.getByLabelText("passwordLabel"), { target: { value: "Valid1!pass" } });
    fireEvent.change(screen.getByLabelText("confirmPasswordLabel"), { target: { value: "Different1!pass" } });
    fireEvent.change(screen.getByLabelText("tokenLabel"), { target: { value: "st_token" } });
    fireEvent.click(screen.getByRole("button", { name: "createAccount" }));

    await waitFor(() => {
      expect(screen.getByText("passwordMismatch")).toBeInTheDocument();
    });
    expect(mockCreateAdmin).not.toHaveBeenCalled();
  });

  it("creates the admin and signs in", async () => {
    mockCreateAdmin.mockReturnValue({
      unwrap: () => Promise.resolve({}),
    });
    mockRefetchSetupStatus.mockResolvedValue({ data: { needsSetup: false } });
    mockLogin.mockReturnValue({
      unwrap: () =>
        Promise.resolve({
          user: { name: "Admin", email: "admin@example.com", role: "admin" },
        }),
    });

    render(<SetupPage />);

    fireEvent.change(screen.getByLabelText("nameLabel"), { target: { value: "Admin" } });
    fireEvent.change(screen.getByLabelText("emailLabel"), { target: { value: "admin@example.com" } });
    fireEvent.change(screen.getByLabelText("passwordLabel"), { target: { value: "Valid1!pass" } });
    fireEvent.change(screen.getByLabelText("confirmPasswordLabel"), { target: { value: "Valid1!pass" } });
    fireEvent.change(screen.getByLabelText("tokenLabel"), { target: { value: "st_token" } });
    fireEvent.click(screen.getByRole("button", { name: "createAccount" }));

    await waitFor(() => {
      expect(mockCreateAdmin).toHaveBeenCalledWith({
        name: "Admin",
        email: "admin@example.com",
        password: "Valid1!pass",
        token: "st_token",
      });
      expect(mockLogin).toHaveBeenCalledWith({
        email: "admin@example.com",
        password: "Valid1!pass",
      });
    });
  });

  it("shows the backend error when setup fails", async () => {
    mockCreateAdmin.mockReturnValue({
      unwrap: () => Promise.reject({ data: { error: "Invalid or expired setup token" } }),
    });

    render(<SetupPage />);

    fireEvent.change(screen.getByLabelText("nameLabel"), { target: { value: "Admin" } });
    fireEvent.change(screen.getByLabelText("emailLabel"), { target: { value: "admin@example.com" } });
    fireEvent.change(screen.getByLabelText("passwordLabel"), { target: { value: "Valid1!pass" } });
    fireEvent.change(screen.getByLabelText("confirmPasswordLabel"), { target: { value: "Valid1!pass" } });
    fireEvent.change(screen.getByLabelText("tokenLabel"), { target: { value: "wrong" } });
    fireEvent.click(screen.getByRole("button", { name: "createAccount" }));

    await waitFor(() => {
      expect(screen.getByText("Invalid or expired setup token")).toBeInTheDocument();
    });
    expect(mockLogin).not.toHaveBeenCalled();
  });
});
