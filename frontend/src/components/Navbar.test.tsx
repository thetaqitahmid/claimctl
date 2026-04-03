import { describe, it, expect } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import { renderWithProviders } from "../test/utils";
import Navbar from "./Navbar";

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

        },
      },
    });
    expect(screen.getByText("Test User")).toBeInTheDocument();
  });

  it("opens user dropdown on click", () => {
    renderWithProviders(<Navbar />, {
      preloadedState: {
        authSlice: {
          user: "Test User",

          email: "test@example.com",
          role: "user",

        },
      },
    });
    
    const userButton = screen.getByText("Test User");
    fireEvent.click(userButton);
    
    // t("components:navbar.profile") -> "profile"
    expect(screen.getByText("profile")).toBeInTheDocument();
    // t("components:navbar.logout") -> "logout"
    expect(screen.getByText("logout")).toBeInTheDocument();
  });

  it("shows admin links for admin users", () => {
    renderWithProviders(<Navbar />, {
      preloadedState: {
        authSlice: {
          user: "Admin User",

          email: "admin@example.com",
          role: "admin",

        },
      },
    });
    
    const userButton = screen.getByText("Admin User");
    fireEvent.click(userButton);
    
    // t("components:navbar.adminPanel") -> "adminPanel"
    expect(screen.getByText("adminPanel")).toBeInTheDocument();
    expect(screen.getByText("secrets")).toBeInTheDocument();
    expect(screen.getByText("webhooks")).toBeInTheDocument();
    expect(screen.getByText("settings")).toBeInTheDocument();
  });

  it("does not show admin links for regular users", () => {
    renderWithProviders(<Navbar />, {
      preloadedState: {
        authSlice: {
          user: "Regular User",

          email: "test@example.com",
          role: "user",

        },
      },
    });
    
    const userButton = screen.getByText("Regular User");
    fireEvent.click(userButton);
    
    expect(screen.queryByText("adminPanel")).not.toBeInTheDocument();
    expect(screen.queryByText("secrets")).not.toBeInTheDocument();
  });
});
