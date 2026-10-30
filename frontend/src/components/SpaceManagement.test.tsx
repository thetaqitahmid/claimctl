import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import SpaceManagement from "./SpaceManagement";

const mocks = vi.hoisted(() => ({
  query: vi.fn(),
  create: vi.fn(),
  remove: vi.fn(),
  notify: vi.fn(),
  refetch: vi.fn(),
}));
vi.mock("../store/api/spaces", () => ({
  useGetSpacesQuery: () => mocks.query(),
  useCreateSpaceMutation: () => [mocks.create, { isLoading: false }],
  useDeleteSpaceMutation: () => [mocks.remove],
}));
vi.mock("../hooks/useNotification", () => ({
  useNotificationContext: () => ({ showNotification: mocks.notify }),
}));
const spaces = [
  { id: "s1", name: "Default Space", description: "Public" },
  { id: "s2", name: "Lab", description: "Equipment" },
];
function setup() {
  render(
    <MemoryRouter>
      <SpaceManagement />
    </MemoryRouter>,
  );
  return userEvent.setup();
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.query.mockReturnValue({ data: spaces, refetch: mocks.refetch });
  mocks.create.mockReturnValue({ unwrap: () => Promise.resolve() });
  mocks.remove.mockReturnValue({ unwrap: () => Promise.resolve() });
});

describe("SpaceManagement", () => {
  it("opens creation on demand and keeps failed entries", async () => {
    mocks.create.mockReturnValue({
      unwrap: () => Promise.reject({ data: { error: "Name already exists" } }),
    });
    const user = setup();
    expect(screen.queryByLabelText("Space name")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Create space" }));
    await user.type(screen.getByLabelText("Space name"), "Engineering");
    await user.type(
      screen.getByLabelText("Description (optional)"),
      "Team resources",
    );
    await user.click(screen.getByRole("button", { name: "Create space" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Name already exists",
    );
    expect(screen.getByLabelText("Space name")).toHaveValue("Engineering");
  });

  it("links to access management and prevents deleting the default space", () => {
    setup();
    expect(
      screen.getByRole("link", { name: "Manage access to Lab" }),
    ).toHaveAttribute("href", "/admin/spaces/s2");
    expect(
      screen.queryByRole("button", { name: "Delete Default Space" }),
    ).not.toBeInTheDocument();
  });

  it("names the space and warns about resource deletion before submitting", async () => {
    const user = setup();
    await user.click(screen.getByRole("button", { name: "Delete Lab" }));
    expect(screen.getByRole("dialog")).toHaveTextContent("Lab");
    expect(screen.getByRole("dialog")).toHaveTextContent(
      "All resources in this space will also be deleted",
    );
    expect(mocks.remove).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Delete space" }));
    await waitFor(() => expect(mocks.remove).toHaveBeenCalledWith("s2"));
  });

  it("offers retry instead of an empty state on failure", async () => {
    mocks.query.mockReturnValue({
      data: [],
      error: { status: 500 },
      refetch: mocks.refetch,
    });
    const user = setup();
    expect(screen.queryByText("No spaces yet")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Retry" }));
    expect(mocks.refetch).toHaveBeenCalledOnce();
  });
});
