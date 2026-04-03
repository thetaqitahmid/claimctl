import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { render } from "../test/utils";
import LoginPage from "./Login";
import * as authApi from "../store/api/auth";
import * as notificationHook from "../hooks/useNotification";

// Mock the hooks
vi.mock("../store/api/auth", async () => {
  const actual = await vi.importActual("../store/api/auth");
  return {
    ...actual,
    useLoginMutation: vi.fn(),
    useLoginLDAPMutation: vi.fn(),
  };
});

vi.mock("../hooks/useNotification", () => ({
  useNotificationContext: vi.fn(),
}));

describe("LoginPage", () => {
  const mockLogin = vi.fn();
  const mockLDAPLogin = vi.fn();
  const mockShowNotification = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    
    vi.mocked(authApi.useLoginMutation).mockReturnValue([
      mockLogin,
      { isLoading: false },
    ] as unknown as ReturnType<typeof authApi.useLoginMutation>);
    vi.mocked(authApi.useLoginLDAPMutation).mockReturnValue([
      mockLDAPLogin,
      { isLoading: false },
    ] as unknown as ReturnType<typeof authApi.useLoginLDAPMutation>);
    vi.mocked(notificationHook.useNotificationContext).mockReturnValue({
      showNotification: mockShowNotification,
    } as unknown as ReturnType<typeof notificationHook.useNotificationContext>);
  });

  it("renders login form", () => {
    render(<LoginPage />);
    expect(screen.getByText("title")).toBeInTheDocument();
    expect(screen.getByLabelText("emailLabel")).toBeInTheDocument();
    expect(screen.getByLabelText("passwordLabel")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "signIn" })).toBeInTheDocument();
  });

  it("toggles LDAP login", () => {
    render(<LoginPage />);
    const ldapToggle = screen.getByRole("checkbox");
    fireEvent.click(ldapToggle);
    
    expect(screen.getByText("ldapTitle")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "ldapSignIn" })).toBeInTheDocument();
  });

  it("calls login mutation on submit", async () => {
    mockLogin.mockReturnValue({
      unwrap: () => Promise.resolve({
        user: { name: "Test User", email: "test@example.com", role: "user" }
      })
    });

    render(<LoginPage />);
    
    fireEvent.change(screen.getByLabelText("emailLabel"), { target: { value: "test@example.com" } });
    fireEvent.change(screen.getByLabelText("passwordLabel"), { target: { value: "password" } });
    fireEvent.click(screen.getByRole("button", { name: "signIn" }));

    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledWith({ email: "test@example.com", password: "password" });
    });
  });

  it("calls LDAP login mutation when toggled", async () => {
    mockLDAPLogin.mockReturnValue({
      unwrap: () => Promise.resolve({
        user: { name: "LDAP User", email: "ldap@example.com", role: "user" }
      })
    });

    render(<LoginPage />);
    fireEvent.click(screen.getByRole("checkbox")); // Enable LDAP
    
    fireEvent.change(screen.getByLabelText("emailLabel"), { target: { value: "ldap@example.com" } });
    fireEvent.change(screen.getByLabelText("passwordLabel"), { target: { value: "password" } });
    fireEvent.click(screen.getByRole("button", { name: "ldapSignIn" }));

    await waitFor(() => {
      expect(mockLDAPLogin).toHaveBeenCalledWith({ email: "ldap@example.com", password: "password" });
    });
  });

  it("shows error message on login failure", async () => {
    mockLogin.mockReturnValue({
      unwrap: () => Promise.reject({
        data: { error: "Invalid credentials" }
      })
    });

    render(<LoginPage />);
    
    fireEvent.change(screen.getByLabelText("emailLabel"), { target: { value: "test@example.com" } });
    fireEvent.change(screen.getByLabelText("passwordLabel"), { target: { value: "wrong" } });
    fireEvent.click(screen.getByRole("button", { name: "signIn" }));

    await waitFor(() => {
      expect(screen.getByText("Invalid credentials")).toBeInTheDocument();
      expect(mockShowNotification).toHaveBeenCalledWith("error", expect.stringContaining("Invalid credentials"));
    });
  });
});
