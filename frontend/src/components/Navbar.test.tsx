import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../test/utils";
import Navbar from "./Navbar";

vi.mock("../hooks/useNotification", () => ({
  useNotificationContext: () => ({ showNotification: vi.fn() }),
}));

describe("Navbar", () => {
  it("renders the app name", () => {
    renderWithProviders(<Navbar />);
    // Our mock t function returns the key or last part of it
    // t("common:appName") -> "common:appName"
    expect(screen.getByText(/appName/i)).toBeInTheDocument();
  });

  it("displays the user name", () => {
    renderWithProviders(<Navbar />, {
      preloadedState: {
        authSlice: {
          user: "Test User",
          email: "test@example.com",
          role: "user",
          authProvider: null,
        },
      },
    });
    expect(screen.getByText("Test User")).toBeInTheDocument();
  });

  it("opens user dropdown on click", async () => {
    renderWithProviders(<Navbar />, {
      preloadedState: {
        authSlice: {
          user: "Test User",
          email: "test@example.com",
          role: "user",
          authProvider: null,
        },
      },
    });

    const userButton = screen.getByText("Test User");
    await userEvent.click(userButton);

    expect(screen.getByText("profile")).toBeInTheDocument();
    expect(screen.getByText("logout")).toBeInTheDocument();
  });

  it("shows administration outside the account menu for admin users", async () => {
    renderWithProviders(<Navbar />, {
      preloadedState: {
        authSlice: {
          user: "Admin User",
          email: "admin@example.com",
          role: "admin",
          authProvider: null,
        },
      },
    });

    expect(
      screen.getByRole("link", { name: "Administration" }),
    ).toHaveAttribute("href", "/admin");
    await userEvent.click(screen.getByText("Admin User"));
    expect(screen.getByText("profile")).toBeInTheDocument();
    expect(screen.queryByText("secrets")).not.toBeInTheDocument();
    expect(screen.queryByText("settings")).not.toBeInTheDocument();
  });

  it("does not show admin links for regular users", async () => {
    renderWithProviders(<Navbar />, {
      preloadedState: {
        authSlice: {
          user: "Regular User",
          email: "test@example.com",
          role: "user",
          authProvider: null,
        },
      },
    });

    const userButton = screen.getByText("Regular User");
    await userEvent.click(userButton);

    expect(
      screen.queryByRole("link", { name: "Administration" }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText("secrets")).not.toBeInTheDocument();
  });
});
