import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useGetResourcesQuery } from "../store/api/resources";
import { useClaimReservationMutation, useGetUserReservationsQuery } from "../store/api/reservations";
import { ResourceWithStatus } from "../types";
import "../test/i18n";
import ResourceList from "./ResourceList";

vi.unmock("react-i18next");
const mocks = vi.hoisted(() => ({
  claim: vi.fn(), complete: vi.fn(), notify: vi.fn(), refetch: vi.fn(),
}));
vi.mock("../store/store", () => ({ useAppSelector: () => false }));
vi.mock("../hooks/useNotification", () => ({ useNotificationContext: () => ({ showNotification: mocks.notify }) }));
vi.mock("../store/api/resources", () => ({
  useGetResourcesQuery: vi.fn(),
  useDeleteResourceMutation: () => [vi.fn()],
  useCreateResourceMutation: () => [vi.fn()],
  useSetMaintenanceModeMutation: () => [vi.fn()],
}));
vi.mock("../store/api/reservations", () => ({
  useClaimReservationMutation: vi.fn(),
  useCancelReservationMutation: () => [vi.fn()],
  useCompleteReservationMutation: () => [mocks.complete],
  useCancelAllReservationsMutation: () => [vi.fn()],
  useGetUserReservationsQuery: vi.fn(),
  useGetReservationPolicyQuery: () => ({
    data: { defaultDuration: "1h", maxDuration: "24h", maxLiveClaims: 8, allowIndefinite: false },
    isFetching: false,
  }),
}));
vi.mock("../store/api/spaces", () => ({ useGetSpacesQuery: () => ({ data: [
  { id: "one", name: "Default Space" }, { id: "two", name: "Lab" },
] }) }));
vi.mock("./AddResource", () => ({ default: () => null }));
vi.mock("./ResourceDetailsModal", () => ({ default: () => null }));

const resource = (id: string, name: string, spaceId: string): ResourceWithStatus => ({
  resource: { id, name, spaceId, type: "server", labels: [], last_modified: 0 },
  activeReservations: 0, queueLength: 0, nextUserId: "", nextQueuePosition: 0,
});
const resources = [resource("r1", "GPU node", "one"), resource("r2", "Lab node", "two")];
function mockResources(isFetching = false, error?: object) {
  vi.mocked(useGetResourcesQuery).mockReturnValue({
    data: resources, isLoading: false, isFetching, error, refetch: mocks.refetch,
  } as ReturnType<typeof useGetResourcesQuery>);
}
function mockReservations(isFetching = false, active = false, error?: object) {
  vi.mocked(useGetUserReservationsQuery).mockReturnValue({
    data: active ? [{ id: "claim1", resourceId: "r1", status: "active", queuePosition: 0 }] : [],
    isLoading: false, isFetching, error, refetch: mocks.refetch,
  } as ReturnType<typeof useGetUserReservationsQuery>);
}

describe("ResourceList", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
    mockResources();
    mockReservations();
    vi.mocked(useClaimReservationMutation).mockReturnValue([mocks.claim, {}] as unknown as ReturnType<typeof useClaimReservationMutation>);
    mocks.claim.mockImplementation(() => ({ unwrap: () => Promise.resolve({ status: "active", queuePosition: 0 }) }));
  });

  it("retains the table and filters during refresh, mutations, and fetch errors", async () => {
    const { rerender } = render(<ResourceList />);
    const filter = screen.getByRole("textbox", { name: "Filter names..." });
    fireEvent.change(filter, { target: { value: "GPU" } });
    const table = screen.getByRole("table");
    mockResources(true);
    rerender(<ResourceList />);
    expect(screen.getByRole("table")).toBe(table);
    expect(filter).toHaveValue("GPU");
    expect(screen.getByRole("status")).toHaveTextContent("Refreshing availability");
    mockResources(false, { status: 500 });
    rerender(<ResourceList />);
    expect(screen.getByRole("table")).toBe(table);
    expect(screen.getByRole("alert")).toHaveTextContent("Previously loaded availability may be out of date");
    await userEvent.click(screen.getByRole("button", { name: "Refresh" }));
    expect(mocks.refetch).toHaveBeenCalled();
  });

  it("provides an associated mobile space selector and recovers a stale selection", async () => {
    sessionStorage.setItem("rm_selectedSpaceId", JSON.stringify("removed-space"));
    render(<ResourceList />);
    const selector = screen.getByRole("combobox", { name: "Spaces" });
    await waitFor(() => expect(selector).toHaveValue("one"));
    fireEvent.change(selector, { target: { value: "two" } });
    expect(screen.getByText("Lab node")).toBeInTheDocument();
    expect(screen.queryByText("GPU node")).not.toBeInTheDocument();
    expect(selector.closest(".md\\:hidden")?.parentElement).toContainElement(screen.getByRole("table"));
  });

  it("retries ambiguous submissions with the same key and uses a new key for changed input", async () => {
    const user = userEvent.setup();
    mocks.claim.mockImplementation(() => ({ unwrap: () => Promise.reject({ status: "FETCH_ERROR" }) }));
    render(<ResourceList />);
    await user.click(screen.getByRole("button", { name: "Reserve" }));
    await user.click(screen.getByRole("button", { name: "Confirm Reservation" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Retry with the same selection");
    await user.click(screen.getByRole("button", { name: "Confirm Reservation" }));
    await waitFor(() => expect(mocks.claim).toHaveBeenCalledTimes(2));
    const first = mocks.claim.mock.calls[0][0];
    expect(first.request).toEqual({ resourceId: "r1" });
    expect(first.idempotencyKey).toBeTruthy();
    expect(mocks.claim.mock.calls[1][0]).toEqual(first);
    await user.click(screen.getByRole("radio", { name: "30 minutes" }));
    mocks.claim.mockImplementation(() => ({ unwrap: () => Promise.resolve({ status: "pending", queuePosition: 2 }) }));
    await user.click(screen.getByRole("button", { name: "Confirm Reservation" }));
    expect(mocks.claim.mock.calls[2][0].idempotencyKey).not.toBe(first.idempotencyKey);
    expect(mocks.claim.mock.calls[2][0].request).toEqual({ resourceId: "r1", duration: "30m" });
    expect(mocks.notify).toHaveBeenCalledWith("success", expect.stringContaining("position 2"));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });

  it("keeps the table mounted and blocks duplicate release while a mutation is pending", async () => {
    mockReservations(false, true);
    let resolve!: () => void;
    mocks.complete.mockImplementation(() => ({ unwrap: () => new Promise<void>((done) => { resolve = done; }) }));
    const user = userEvent.setup();
    render(<ResourceList />);
    const table = screen.getByRole("table");
    await user.dblClick(screen.getByRole("button", { name: "Release" }));
    expect(mocks.complete).toHaveBeenCalledOnce();
    expect(screen.getByRole("button", { name: "Release" })).toBeDisabled();
    expect(screen.getByRole("table")).toBe(table);
    resolve();
    await waitFor(() => expect(screen.getByRole("button", { name: "Release" })).toBeEnabled());
  });

  it("does not offer an incorrect reserve action while personal reservations are unavailable", () => {
    mockReservations(false, false, { status: 500 });
    render(<ResourceList />);
    expect(screen.getByRole("button", { name: "Reserve" })).toBeDisabled();
    expect(screen.getByRole("alert")).toHaveTextContent("Reservation actions are disabled");
  });
});
