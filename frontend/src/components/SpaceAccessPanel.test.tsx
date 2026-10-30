import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SpaceAccessPanel } from "./SpaceAccessPanel";

const mocks = vi.hoisted(() => ({
  query: vi.fn(),
  add: vi.fn(),
  remove: vi.fn(),
  notify: vi.fn(),
  refetch: vi.fn(),
}));
vi.mock("../store/api/spaces", () => ({
  useGetSpacePermissionsQuery: (...args: unknown[]) => mocks.query(...args),
  useAddSpacePermissionMutation: () => [mocks.add, { isLoading: false }],
  useRemoveSpacePermissionMutation: () => [mocks.remove],
}));
vi.mock("../store/api/groups", () => ({
  useGetGroupsQuery: () => ({
    data: [{ id: "g1", name: "Engineering", description: "Engineering team" }],
  }),
}));
vi.mock("../store/api/users", () => ({
  useGetUsersQuery: () => ({
    data: [{ id: "u1", name: "Alex", email: "alex@example.com" }],
  }),
}));
vi.mock("../hooks/useNotification", () => ({
  useNotificationContext: () => ({ showNotification: mocks.notify }),
}));
const space = {
  id: "s1",
  name: "Lab",
  description: "Shared equipment",
  createdAt: 0,
  updatedAt: 0,
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.query.mockReturnValue({
    data: [],
    error: null,
    isLoading: false,
    refetch: mocks.refetch,
  });
  mocks.add.mockReturnValue({ unwrap: () => Promise.resolve() });
  mocks.remove.mockReturnValue({ unwrap: () => Promise.resolve() });
});

describe("SpaceAccessPanel", () => {
  it("explains public access without showing grant controls", () => {
    render(<SpaceAccessPanel space={{ ...space, name: "Default Space" }} />);
    expect(
      screen.getByRole("heading", { name: "Public access" }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Grant access/ }),
    ).not.toBeInTheDocument();
    expect(mocks.query).toHaveBeenCalledWith("s1", { skip: true });
  });

  it("grants groups and individual users with explicit controls", async () => {
    const user = userEvent.setup();
    render(<SpaceAccessPanel space={space} />);
    await user.click(
      screen.getByRole("button", { name: "Grant access to Engineering" }),
    );
    expect(mocks.add).toHaveBeenCalledWith({ spaceId: "s1", groupId: "g1" });
    await user.selectOptions(screen.getByLabelText("Grant access to"), "users");
    await user.type(
      screen.getByRole("searchbox", { name: "Search users" }),
      "alex@example.com",
    );
    await user.click(
      screen.getByRole("button", { name: "Grant access to Alex" }),
    );
    expect(mocks.add).toHaveBeenCalledWith({ spaceId: "s1", userId: "u1" });
  });

  it("distinguishes group inheritance from direct access and confirms removal", async () => {
    mocks.query.mockReturnValue({
      data: [
        { id: "p1", groupId: "g1", groupName: "Engineering" },
        { id: "p2", userId: "u1", userEmail: "alex@example.com" },
      ],
    });
    const user = userEvent.setup();
    render(<SpaceAccessPanel space={space} />);
    expect(screen.getByText("Through group membership")).toBeInTheDocument();
    expect(screen.getByText("Direct user access")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Grant access to Engineering" }),
    ).not.toBeInTheDocument();
    await user.click(
      screen.getByRole("button", { name: "Remove access for Engineering" }),
    );
    expect(mocks.remove).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Remove access" }));
    await waitFor(() =>
      expect(mocks.remove).toHaveBeenCalledWith({
        spaceId: "s1",
        groupId: "g1",
      }),
    );
  });

  it("shows grant errors instead of silently failing", async () => {
    mocks.add.mockReturnValue({
      unwrap: () => Promise.reject({ data: { error: "Grant failed" } }),
    });
    const user = userEvent.setup();
    render(<SpaceAccessPanel space={space} />);
    await user.click(
      screen.getByRole("button", { name: "Grant access to Engineering" }),
    );
    expect(await screen.findByRole("alert")).toHaveTextContent("Grant failed");
  });

  it("does not mistake a query failure for private access", async () => {
    mocks.query.mockReturnValue({
      data: [],
      error: { status: 500 },
      refetch: mocks.refetch,
    });
    const user = userEvent.setup();
    render(<SpaceAccessPanel space={space} />);
    expect(screen.getByRole("alert")).toHaveTextContent(
      "Could not load access",
    );
    expect(screen.queryByText(/No access granted/)).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Retry" }));
    expect(mocks.refetch).toHaveBeenCalledOnce();
  });
});
