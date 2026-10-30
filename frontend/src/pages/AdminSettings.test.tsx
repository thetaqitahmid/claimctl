import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import { render } from "../test/utils";
import AdminSettings from "./AdminSettings";

vi.mock("../store/api/settings", async (importOriginal) => {
  const original = await importOriginal<typeof import("../store/api/settings")>();
  return {
    ...original,
    useGetSettingsQuery: () => ({
      data: [
        { key: "oidc_issuer", category: "auth", value: "https://invalid.test" },
        { key: "smtp_host", category: "notification", value: "smtp.invalid.test" },
      ],
      isLoading: false,
      refetch: vi.fn(),
    }),
    useUpdateSettingMutation: () => [vi.fn()],
  };
});

vi.mock("../hooks/useNotification", () => ({
  useNotificationContext: () => ({ showNotification: vi.fn() }),
}));

describe("Admin settings", () => {
  it("keeps settings tabs without application backup or restore controls", () => {
    render(<AdminSettings />, {
      preloadedState: {
        authSlice: {
          user: "Admin",
          email: "admin@invalid.test",
          role: "admin",
          authProvider: "local",
        },
      },
    });

    expect(screen.getByRole("button", { name: "auth" })).toBeInTheDocument();
    expect(screen.getByText("oidc_issuer")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "notification" }));
    expect(screen.getByText("smtp_host")).toBeInTheDocument();
    expect(screen.queryByText("oidc_issuer")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /backup|restore/i })).not.toBeInTheDocument();
    expect(document.querySelector('input[type="file"]')).not.toBeInTheDocument();
  });
});
