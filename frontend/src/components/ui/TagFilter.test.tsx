import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import "../../test/i18n";
import TagFilter from "./TagFilter";
import TypeFilter from "./TypeFilter";

vi.unmock("react-i18next");

describe.each(["labels", "types"] as const)("Resource %s filter", (kind) => {
  const onChange = vi.fn();
  const values = ["linux", "fast", "gpu"];
  const label = `Filter by ${kind}`;
  const setup = (selected: string[] = [], available = values) => render(
    <div>
      <button type="button">Outside</button>
      {kind === "labels"
        ? <TagFilter allTags={available} selectedTags={selected} onChange={onChange} />
        : <TypeFilter allTypes={available} selectedTypes={selected} onChange={onChange} />}
    </div>,
  );

  beforeEach(() => vi.clearAllMocks());

  it("names the trigger and exposes expanded state and selection to assistive technology", async () => {
    setup(["linux"]);
    const trigger = screen.getByRole("button", { name: label });
    expect(trigger).toHaveAttribute("aria-expanded", "false");
    await userEvent.click(trigger);
    expect(trigger).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByRole("group", { name: label })).toHaveAttribute("id", trigger.getAttribute("aria-controls"));
    expect(screen.getByRole("button", { name: "linux" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "fast" })).toHaveAttribute("aria-pressed", "false");
  });

  it("can select and deselect using the keyboard", async () => {
    setup(["linux"]);
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: label }));
    const option = screen.getByRole("button", { name: "fast" });
    option.focus();
    await user.keyboard("{Enter}");
    expect(onChange).toHaveBeenCalledWith(["linux", "fast"]);
    screen.getByRole("button", { name: "linux" }).focus();
    await user.keyboard(" ");
    expect(onChange).toHaveBeenLastCalledWith([]);
  });

  it("closes on Escape and restores focus to its trigger", async () => {
    setup();
    const user = userEvent.setup();
    const trigger = screen.getByRole("button", { name: label });
    await user.click(trigger);
    screen.getByRole("button", { name: "gpu" }).focus();
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("group")).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });

  it("closes when clicking outside", async () => {
    setup();
    await userEvent.click(screen.getByRole("button", { name: label }));
    fireEvent.mouseDown(screen.getByRole("button", { name: "Outside" }));
    expect(screen.queryByRole("group")).not.toBeInTheDocument();
  });

  it("localizes plural counts and removes selections", async () => {
    setup(["linux", "fast"]);
    expect(screen.getByText(`2 ${kind} selected`)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Remove linux" }));
    expect(onChange).toHaveBeenCalledWith(["fast"]);
    await userEvent.click(screen.getByRole("button", { name: `Clear all ${kind}` }));
    expect(onChange).toHaveBeenLastCalledWith([]);
  });

  it("shows an empty state", async () => {
    setup([], []);
    await userEvent.click(screen.getByRole("button", { name: label }));
    expect(screen.getByText(`No ${kind} available`)).toBeInTheDocument();
  });
});
