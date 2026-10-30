import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import UserTable from "./UserTable";

const data = Array.from({ length: 13 }, (_, index) => ({
  id: String(index),
  name: `Person ${String(index).padStart(2, "0")}`,
  email: `person${index}@example.com`,
  role: index === 0 ? "admin" : "user",
  status: index === 1 ? "inactive" : "active",
}));
const callbacks = {
  onEdit: vi.fn(),
  onToggleStatus: vi.fn(),
  onDelete: vi.fn(),
};

describe("UserTable", () => {
  it("searches email and clears filters", async () => {
    const user = userEvent.setup();
    render(<UserTable data={data} {...callbacks} />);
    await user.type(
      screen.getByRole("searchbox", { name: "Search users" }),
      "person12@example.com",
    );
    expect(screen.getByText("Person 12")).toBeInTheDocument();
    expect(screen.queryByText("Person 00")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(screen.getByText("Person 00")).toBeInTheDocument();
  });

  it("filters by role and status with a useful no results state", async () => {
    const user = userEvent.setup();
    render(<UserTable data={data} {...callbacks} />);
    await user.selectOptions(screen.getByLabelText("Role"), "admin");
    expect(screen.getByText("Person 00")).toBeInTheDocument();
    await user.selectOptions(screen.getByLabelText("Status"), "inactive");
    expect(screen.getByText(/No users match your filters/)).toBeInTheDocument();
  });

  it("paginates and clamps the page after data shrinks", async () => {
    const user = userEvent.setup();
    const { rerender } = render(<UserTable data={data} {...callbacks} />);
    expect(screen.queryByText("Person 12")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Next" }));
    expect(screen.getByText("Person 12")).toBeInTheDocument();
    rerender(<UserTable data={data.slice(0, 1)} {...callbacks} />);
    expect(screen.getByText("Person 00")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
  });

  it("keeps editing discoverable and destructive operations separate", async () => {
    const user = userEvent.setup();
    render(<UserTable data={data.slice(0, 1)} {...callbacks} />);
    await user.click(screen.getByRole("button", { name: "Edit Person 00" }));
    expect(callbacks.onEdit).toHaveBeenCalledWith(data[0]);
    await user.click(screen.getByText("More"));
    await user.click(screen.getByRole("menuitem", { name: "Delete" }));
    expect(callbacks.onDelete).toHaveBeenCalledWith(data[0]);
  });
});
