import { describe, it, expect, vi } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import { render } from "../test/utils";
import Profile from "./Profile";

// Mock sub-components
vi.mock("../components/profile/ProfileReservations", () => ({
  default: () => <div data-testid="profile-reservations">ProfileReservations</div>,
}));
vi.mock("../components/profile/ProfileHistory", () => ({
  default: () => <div data-testid="profile-history">ProfileHistory</div>,
}));
vi.mock("../components/profile/ProfileNotifications", () => ({
  default: () => <div data-testid="profile-notifications">ProfileNotifications</div>,
}));
vi.mock("../components/profile/ProfileTokens", () => ({
  default: () => <div data-testid="profile-tokens">ProfileTokens</div>,
}));
vi.mock("../components/profile/ChangePasswordModal", () => ({
  default: ({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) => 
    isOpen ? <div data-testid="change-password-modal">ChangePasswordModal <button onClick={onClose}>Close</button></div> : null,
}));

describe("Profile Page", () => {
  it("renders user information", () => {
    render(<Profile />, {
      preloadedState: {
        authSlice: {
          user: "Test User",
          email: "test@example.com",
          role: "user",


        },
      },
    });
    
    expect(screen.getByText("Test User")).toBeInTheDocument();
    expect(screen.queryByText("Admin")).not.toBeInTheDocument();
  });

  it("shows Admin badge for admin users", () => {
    render(<Profile />, {
      preloadedState: {
        authSlice: {
          user: "Admin User",
          email: "admin@example.com",
          role: "admin",


        },
      },
    });
    
    expect(screen.getByText("Admin")).toBeInTheDocument();
  });

  it("switches tabs", () => {
    render(<Profile />, {
      preloadedState: {
        authSlice: {
          user: "Test User",
          email: "test@example.com",
          role: "user",


        },
      },
    });
    
    // Default tab is reservations
    expect(screen.getByTestId("profile-reservations")).toBeInTheDocument();
    
    // Switch to Activity Log
    fireEvent.click(screen.getByText("Activity Log"));
    expect(screen.getByTestId("profile-history")).toBeInTheDocument();
    expect(screen.queryByTestId("profile-reservations")).not.toBeInTheDocument();
    
    // Switch to Notifications
    fireEvent.click(screen.getByText("Notifications"));
    expect(screen.getByTestId("profile-notifications")).toBeInTheDocument();
    
    // Switch to API Tokens
    fireEvent.click(screen.getByText("API Tokens"));
    expect(screen.getByTestId("profile-tokens")).toBeInTheDocument();
  });

  it("opens change password modal", () => {
    render(<Profile />, {
      preloadedState: {
        authSlice: {
          user: "Test User",
          email: "test@example.com",
          role: "user",


        },
      },
    });
    
    const changePasswordButton = screen.getByText("Change Password");
    fireEvent.click(changePasswordButton);
    
    expect(screen.getByTestId("change-password-modal")).toBeInTheDocument();
    
    const closeButton = screen.getByText("Close");
    fireEvent.click(closeButton);
    
    expect(screen.queryByTestId("change-password-modal")).not.toBeInTheDocument();
  });
});
