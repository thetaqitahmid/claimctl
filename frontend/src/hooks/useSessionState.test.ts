import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import useSessionState from "./useSessionState";

describe("useSessionState", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it("returns default value when sessionStorage is empty", () => {
    const { result } = renderHook(() =>
      useSessionState("key", "default")
    );
    expect(result.current[0]).toBe("default");
  });

  it("returns stored value from sessionStorage", () => {
    sessionStorage.setItem("key", JSON.stringify("stored"));
    const { result } = renderHook(() =>
      useSessionState("key", "default")
    );
    expect(result.current[0]).toBe("stored");
  });

  it("writes to sessionStorage on state change", () => {
    const { result } = renderHook(() =>
      useSessionState("key", "initial")
    );

    act(() => {
      result.current[1]("updated");
    });

    expect(result.current[0]).toBe("updated");
    expect(sessionStorage.getItem("key")).toBe(
      JSON.stringify("updated")
    );
  });

  it("supports functional updates", () => {
    const { result } = renderHook(() =>
      useSessionState("counter", 0)
    );

    act(() => {
      result.current[1]((prev) => prev + 1);
    });

    expect(result.current[0]).toBe(1);
    expect(sessionStorage.getItem("counter")).toBe("1");
  });

  it("falls back to default on invalid JSON", () => {
    sessionStorage.setItem("key", "not-valid-json");
    const { result } = renderHook(() =>
      useSessionState("key", "fallback")
    );
    expect(result.current[0]).toBe("fallback");
  });

  it("handles null values correctly", () => {
    const { result } = renderHook(() =>
      useSessionState<string | null>("key", null)
    );
    expect(result.current[0]).toBeNull();

    act(() => {
      result.current[1]("value");
    });
    expect(result.current[0]).toBe("value");

    act(() => {
      result.current[1](null);
    });
    expect(result.current[0]).toBeNull();
    expect(sessionStorage.getItem("key")).toBe("null");
  });
});
