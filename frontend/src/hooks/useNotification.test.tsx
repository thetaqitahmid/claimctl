import { describe, it, expect } from "vitest";
import { renderHook } from "@testing-library/react";
import { useNotificationContext } from "./useNotification";
import {
  NotificationContext,
  NotificationContextType,
} from "../store/context/NotificationContext";
import React from "react";

describe("useNotificationContext", () => {
  it("throws error when used outside of NotificationProvider", () => {
    expect(() => {
      renderHook(() => useNotificationContext());
    }).toThrow(
      "useNotificationContext must be used within a NotificationProvider",
    );
  });

  it("returns context when used within NotificationProvider", () => {
    const mockContext: NotificationContextType = {
      notifications: [],
      showNotification: () => {},
      removeNotification: () => {},
    };

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <NotificationContext.Provider value={mockContext}>
        {children}
      </NotificationContext.Provider>
    );

    const { result } = renderHook(() => useNotificationContext(), { wrapper });

    expect(result.current).toBe(mockContext);
    expect(result.current.notifications).toEqual([]);
  });
});
