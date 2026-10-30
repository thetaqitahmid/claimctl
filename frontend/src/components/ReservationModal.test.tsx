import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useGetReservationPolicyQuery } from "../store/api/reservations";
import { ReservationPolicy } from "../types";
import "../test/i18n";
import ReservationModal from "./ReservationModal";

vi.unmock("react-i18next");
vi.mock("../store/api/reservations", () => ({ useGetReservationPolicyQuery: vi.fn() }));

const policy: ReservationPolicy = {
  defaultDuration: "1h0m0s", maxDuration: "24h0m0s", maxLiveClaims: 8, allowIndefinite: false,
};
const refetch = vi.fn();
function mockPolicy(data: ReservationPolicy | undefined = policy, error?: object) {
  vi.mocked(useGetReservationPolicyQuery).mockReturnValue({
    data, error, isFetching: false, refetch,
  } as ReturnType<typeof useGetReservationPolicyQuery>);
}

function setup(onConfirm = vi.fn().mockResolvedValue(undefined)) {
  const onClose = vi.fn();
  render(<ReservationModal isOpen onClose={onClose} resourceName="GPU node" onConfirm={onConfirm} />);
  return { onConfirm, onClose, user: userEvent.setup() };
}

describe("ReservationModal", () => {
  beforeEach(() => { vi.clearAllMocks(); mockPolicy(); });

  it("uses the server default, not an implicit indefinite reservation", async () => {
    const { user, onConfirm, onClose } = setup();
    expect(screen.getByRole("dialog", { name: "Reserve GPU node" })).toBeInTheDocument();
    expect(screen.queryByRole("radio", { name: "Indefinite" })).not.toBeInTheDocument();
    expect(screen.getByText(/Default: 1h0m0s/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Confirm Reservation" }));
    expect(onConfirm).toHaveBeenCalledWith({});
    await waitFor(() => expect(onClose).toHaveBeenCalledOnce());
  });

  it("keeps the form and values on failure and offers a safe retry", async () => {
    const onConfirm = vi.fn().mockRejectedValueOnce({ status: "FETCH_ERROR" }).mockResolvedValueOnce(undefined);
    const { user, onClose } = setup(onConfirm);
    await user.click(screen.getByRole("radio", { name: "4 hours" }));
    await user.click(screen.getByRole("button", { name: "Confirm Reservation" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Retry with the same selection");
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByRole("radio", { name: "4 hours" })).toBeChecked();
    await user.click(screen.getByRole("button", { name: "Confirm Reservation" }));
    expect(onConfirm.mock.calls).toEqual([[{ duration: "4h" }], [{ duration: "4h" }]]);
    await waitFor(() => expect(onClose).toHaveBeenCalledOnce());
  });

  it("prevents duplicate submissions and dismissal while awaiting confirmation", async () => {
    let resolve!: () => void;
    const onConfirm = vi.fn(() => new Promise<void>((done) => { resolve = done; }));
    const { user, onClose } = setup(onConfirm);
    const button = screen.getByRole("button", { name: "Confirm Reservation" });
    await user.dblClick(button);
    expect(onConfirm).toHaveBeenCalledOnce();
    expect(screen.getByRole("button", { name: "Reserving…" })).toBeDisabled();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    await user.keyboard("{Escape}");
    expect(onClose).not.toHaveBeenCalled();
    await act(async () => resolve());
    await waitFor(() => expect(onClose).toHaveBeenCalledOnce());
  });

  it("validates positive whole custom durations and the server maximum", async () => {
    const { user, onConfirm } = setup();
    await user.click(screen.getByRole("radio", { name: "Custom" }));
    const submit = screen.getByRole("button", { name: "Confirm Reservation" });
    expect(submit).toBeDisabled();
    const days = screen.getByRole("spinbutton", { name: "Days" });
    fireEvent.change(days, { target: { value: "2" } });
    expect(submit).toBeDisabled();
    fireEvent.change(days, { target: { value: "0.5" } });
    expect(submit).toBeDisabled();
    fireEvent.change(days, { target: { value: "-1" } });
    expect(submit).toBeDisabled();
    fireEvent.change(days, { target: { value: "0" } });
    fireEvent.change(screen.getByRole("spinbutton", { name: "Minutes" }), { target: { value: "45" } });
    await user.click(submit);
    expect(onConfirm).toHaveBeenCalledWith({ duration: "2700s" });
  });

  it("hides overmaximum presets and only sends indefinite explicitly when allowed", async () => {
    mockPolicy({ ...policy, maxDuration: "2h0m0s", allowIndefinite: true });
    const { user, onConfirm } = setup();
    expect(screen.queryByRole("radio", { name: "4 hours" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("radio", { name: "Indefinite" }));
    await user.click(screen.getByRole("button", { name: "Confirm Reservation" }));
    expect(onConfirm).toHaveBeenCalledWith({ indefinite: true });
  });

  it("blocks submission when policy cannot be loaded and exposes a retry", async () => {
    mockPolicy(undefined, { status: 500 });
    const { user, onConfirm } = setup();
    expect(screen.getByRole("alert")).toHaveTextContent("Could not load reservation policy");
    expect(screen.getByRole("button", { name: "Confirm Reservation" })).toBeDisabled();
    await user.click(screen.getByRole("button", { name: "Refresh" }));
    expect(refetch).toHaveBeenCalledOnce();
    expect(onConfirm).not.toHaveBeenCalled();
  });

  it.each([
    [409, "availability or your reservation quota changed"],
    [403, "Contact your administrator"],
    [400, "Choose a shorter duration"],
  ])("provides actionable feedback for HTTP %s", async (status, message) => {
    const { user, onClose } = setup(vi.fn().mockRejectedValue({ status }));
    await user.click(screen.getByRole("button", { name: "Confirm Reservation" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(message);
    expect(onClose).not.toHaveBeenCalled();
  });
});
