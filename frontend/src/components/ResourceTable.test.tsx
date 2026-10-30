import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ResourceTable from "./ResourceTable";

vi.mock("./QueueListModal", () => ({ default: () => null }));
vi.mock("./HealthStatusIndicator", () => ({
  default: () => <span>Healthy</span>,
}));

const resource = {
  resource: {
    id: "r1",
    name: "Build server",
    type: "server",
    labels: [],
    last_modified: 0,
  },
  activeReservations: 0,
  queueLength: 0,
  nextUserId: "",
  nextQueuePosition: 0,
};
const callbacks = {
  onReserve: vi.fn(),
  onRelease: vi.fn(),
  onView: vi.fn(),
  onDelete: vi.fn(),
};

describe("ResourceTable action hierarchy", () => {
  it("shows a visible reserve action and hides administrator operations for users", async () => {
    const user = userEvent.setup();
    render(
      <ResourceTable
        data={[resource]}
        userReservations={[]}
        isAdmin={false}
        {...callbacks}
      />,
    );
    await user.click(screen.getByRole("button", { name: "reserve" }));
    expect(callbacks.onReserve).toHaveBeenCalledWith("r1");
    expect(
      screen.queryByRole("button", { name: "adminActions" }),
    ).not.toBeInTheDocument();
  });

  it("labels the queue action explicitly", () => {
    render(
      <ResourceTable
        data={[{ ...resource, activeReservations: 1 }]}
        userReservations={[]}
        isAdmin={false}
        {...callbacks}
      />,
    );
    expect(screen.getByRole("button", { name: "joinQueue" })).toHaveTextContent(
      "joinQueue",
    );
  });

  it("preserves the upstream busy guard for reservations and administrator actions", () => {
    const { rerender } = render(
      <ResourceTable data={[resource]} userReservations={[]} isAdmin busyResourceIds={["r1"]} {...callbacks} />,
    );
    expect(screen.getByRole("button", { name: "reserve" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "adminActions" })).toBeDisabled();
    rerender(
      <ResourceTable data={[resource]} userReservations={[]} isAdmin busyResourceIds={[]} {...callbacks} />,
    );
    expect(screen.getByRole("button", { name: "reserve" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "adminActions" })).toBeEnabled();
  });

  it("preserves the upstream reservation readiness guard", () => {
    const { rerender } = render(
      <ResourceTable data={[resource]} userReservations={[]} isAdmin={false} reservationsReady={false} {...callbacks} />,
    );
    expect(screen.getByRole("button", { name: "reserve" })).toBeDisabled();
    rerender(
      <ResourceTable data={[resource]} userReservations={[]} isAdmin={false} reservationsReady {...callbacks} />,
    );
    expect(screen.getByRole("button", { name: "reserve" })).toBeEnabled();
  });

  it("keeps less common administrator operations in a keyboard accessible menu", async () => {
    const user = userEvent.setup();
    render(
      <ResourceTable
        data={[resource]}
        userReservations={[]}
        isAdmin
        {...callbacks}
      />,
    );
    expect(
      screen.queryByRole("menuitem", { name: "delete" }),
    ).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "adminActions" }));
    await user.click(screen.getByRole("menuitem", { name: "delete" }));
    expect(callbacks.onDelete).toHaveBeenCalledWith("r1");
  });
});
