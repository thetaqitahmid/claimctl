import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useGetQueueForResourceQuery } from "../store/api/reservations";
import "../test/i18n";
import QueueListModal from "./QueueListModal";

vi.unmock("react-i18next");
vi.mock("../store/api/reservations", () => ({ useGetQueueForResourceQuery: vi.fn() }));
const refetch = vi.fn();
const entries = [{
  id: "claim1", userId: "user1", userName: "Ada", userEmail: "ada@example.test",
  status: "pending", queuePosition: 2, createdAt: Math.floor(Date.now() / 1000) - 60, startTime: 0,
}];
function mockQuery(error?: object, isFetching = false) {
  vi.mocked(useGetQueueForResourceQuery).mockReturnValue({
    data: entries, isLoading: false, isFetching, error, refetch,
  } as ReturnType<typeof useGetQueueForResourceQuery>);
}

describe("QueueListModal", () => {
  beforeEach(() => { vi.clearAllMocks(); mockQuery(); });

  it("exposes a named dialog, positions, and an accessible close button", async () => {
    const onClose = vi.fn();
    render(<QueueListModal isOpen resourceId="r1" resourceName="GPU node" onClose={onClose} />);
    expect(screen.getByRole("dialog", { name: "Queue for GPU node" })).toBeInTheDocument();
    expect(screen.getByText("Position #2")).toBeInTheDocument();
    expect(screen.getByText("Ada")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("supports keyboard dismissal", async () => {
    const onClose = vi.fn();
    render(<QueueListModal isOpen resourceId="r1" resourceName="GPU node" onClose={onClose} />);
    await userEvent.keyboard("{Escape}");
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("retains entries on refresh errors and offers an explicit retry", async () => {
    const { rerender } = render(<QueueListModal isOpen resourceId="r1" resourceName="GPU node" onClose={vi.fn()} />);
    mockQuery({ status: 500 });
    rerender(<QueueListModal isOpen resourceId="r1" resourceName="GPU node" onClose={vi.fn()} />);
    expect(screen.getByRole("alert")).toHaveTextContent("Displayed entries may be stale");
    expect(screen.getByText("Ada")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Refresh" }));
    expect(refetch).toHaveBeenCalledOnce();
  });
});
