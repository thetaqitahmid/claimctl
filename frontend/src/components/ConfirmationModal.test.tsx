import { beforeEach, describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ConfirmationModal from "./ConfirmationModal";

const props = {
  isOpen: true,
  onClose: vi.fn(),
  onConfirm: vi.fn(),
  title: "Confirm action",
  message: "Are you sure?",
};

beforeEach(() => vi.clearAllMocks());

describe("ConfirmationModal", () => {
  it("renders nothing when closed", () => {
    render(<ConfirmationModal {...props} isOpen={false} />);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("has an accessible title and description", () => {
    render(<ConfirmationModal {...props} />);
    expect(
      screen.getByRole("dialog", { name: "Confirm action" }),
    ).toHaveAccessibleDescription("Are you sure?");
  });

  it("cancels without confirming", async () => {
    const user = userEvent.setup();
    render(<ConfirmationModal {...props} />);
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(props.onClose).toHaveBeenCalledOnce();
    expect(props.onConfirm).not.toHaveBeenCalled();
  });

  it("closes with the labelled close button", async () => {
    const user = userEvent.setup();
    render(<ConfirmationModal {...props} />);
    await user.click(
      screen.getByRole("button", { name: "Close confirmation" }),
    );
    expect(props.onClose).toHaveBeenCalledOnce();
  });

  it("closes after a successful confirmation", async () => {
    const user = userEvent.setup();
    render(<ConfirmationModal {...props} />);
    await user.click(screen.getByRole("button", { name: "Confirm" }));
    await waitFor(() => expect(props.onClose).toHaveBeenCalledOnce());
    expect(props.onConfirm).toHaveBeenCalledOnce();
  });

  it("requires the exact confirmation word", async () => {
    const user = userEvent.setup();
    render(
      <ConfirmationModal
        {...props}
        confirmationWord="RESTORE"
        confirmText="Restore backup"
      />,
    );
    const confirm = screen.getByRole("button", { name: "Restore backup" });
    expect(confirm).toBeDisabled();
    await user.type(screen.getByRole("textbox"), "restore");
    expect(confirm).toBeDisabled();
    await user.clear(screen.getByRole("textbox"));
    await user.type(screen.getByRole("textbox"), "RESTORE");
    expect(confirm).toBeEnabled();
  });

  it("keeps failures open and permits retry", async () => {
    const user = userEvent.setup();
    const confirm = vi
      .fn()
      .mockRejectedValueOnce({ data: { error: "Service unavailable" } })
      .mockResolvedValue(undefined);
    render(<ConfirmationModal {...props} onConfirm={confirm} />);
    await user.click(screen.getByRole("button", { name: "Confirm" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Service unavailable",
    );
    expect(props.onClose).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Confirm" }));
    await waitFor(() => expect(props.onClose).toHaveBeenCalledOnce());
  });

  it("disables dismissal and duplicate submissions while pending", async () => {
    const user = userEvent.setup();
    let resolve!: () => void;
    const confirm = vi.fn(
      () =>
        new Promise<void>((done) => {
          resolve = done;
        }),
    );
    render(<ConfirmationModal {...props} onConfirm={confirm} />);
    await user.click(screen.getByRole("button", { name: "Confirm" }));
    expect(screen.getByRole("button", { name: "Working…" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();
    await user.keyboard("{Escape}");
    expect(props.onClose).not.toHaveBeenCalled();
    resolve();
    await waitFor(() => expect(props.onClose).toHaveBeenCalledOnce());
    expect(confirm).toHaveBeenCalledOnce();
  });

  it("can let the caller handle navigation without a second close", async () => {
    const user = userEvent.setup();
    render(<ConfirmationModal {...props} closeOnConfirm={false} />);
    await user.click(screen.getByRole("button", { name: "Confirm" }));
    expect(props.onConfirm).toHaveBeenCalledOnce();
    expect(props.onClose).not.toHaveBeenCalled();
  });
});
