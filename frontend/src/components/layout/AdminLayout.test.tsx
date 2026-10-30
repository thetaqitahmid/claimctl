import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import AdminLayout from "./AdminLayout";

function setup() {
  render(
    <MemoryRouter initialEntries={["/admin/users"]}>
      <Routes>
        <Route path="/admin" element={<AdminLayout />}>
          <Route path="users" element={<h1>User accounts</h1>} />
          <Route path="groups" element={<h1>Group accounts</h1>} />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

describe("AdminLayout", () => {
  it("shows focused content, persistent destinations, and the current location", () => {
    setup();
    expect(
      screen.getByRole("heading", { name: "User accounts" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Users" })).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(
      screen.getByRole("link", { name: "Authentication" }),
    ).toHaveAttribute("href", "/admin/settings/authentication");
    expect(screen.getByRole("link", { name: "Spaces" })).toHaveAttribute(
      "href",
      "/admin/spaces",
    );
  });

  it("closes the mobile navigation when selecting a destination", async () => {
    const user = userEvent.setup();
    setup();
    const toggle = screen.getByRole("button", {
      name: "Administration navigation",
    });
    await user.click(toggle);
    expect(toggle).toHaveAttribute("aria-expanded", "true");
    await user.click(screen.getByRole("link", { name: "Groups" }));
    expect(
      screen.getByRole("heading", { name: "Group accounts" }),
    ).toBeInTheDocument();
    expect(toggle).toHaveAttribute("aria-expanded", "false");
  });
});
