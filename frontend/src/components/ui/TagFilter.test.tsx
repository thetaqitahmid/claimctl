import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import TagFilter from "./TagFilter";

describe("TagFilter", () => {
  const allTags = ["tag1", "tag2", "tag3"];
  const selectedTags: string[] = [];
  const onChange = vi.fn();

  it("renders with default message when no tags selected", () => {
    render(<TagFilter allTags={allTags} selectedTags={selectedTags} onChange={onChange} />);
    expect(screen.getByText("Filter by tags...")).toBeInTheDocument();
  });

  it("opens dropdown on click", () => {
    render(<TagFilter allTags={allTags} selectedTags={selectedTags} onChange={onChange} />);
    const button = screen.getByRole("button", { name: /Filter by tags.../i });
    fireEvent.click(button);
    
    expect(screen.getByText("tag1")).toBeInTheDocument();
    expect(screen.getByText("tag2")).toBeInTheDocument();
    expect(screen.getByText("tag3")).toBeInTheDocument();
  });

  it("calls onChange when a tag is selected", () => {
    render(<TagFilter allTags={allTags} selectedTags={selectedTags} onChange={onChange} />);
    const button = screen.getByRole("button", { name: /Filter by tags.../i });
    fireEvent.click(button);
    
    const tag1Button = screen.getByText("tag1");
    fireEvent.click(tag1Button);
    
    expect(onChange).toHaveBeenCalledWith(["tag1"]);
  });

  it("displays selected tags", () => {
    render(<TagFilter allTags={allTags} selectedTags={["tag1", "tag2"]} onChange={onChange} />);
    expect(screen.getByText("tag1")).toBeInTheDocument();
    expect(screen.getByText("tag2")).toBeInTheDocument();
    expect(screen.getByText("2 tags selected")).toBeInTheDocument();
  });

  it("calls onChange to remove a tag when 'X' is clicked", () => {
    render(<TagFilter allTags={allTags} selectedTags={["tag1"]} onChange={onChange} />);
    const removeButton = screen.getByTitle("Remove tag1");
    fireEvent.click(removeButton);
    
    expect(onChange).toHaveBeenCalledWith([]);
  });

  it("calls onChange to clear all tags", () => {
    render(<TagFilter allTags={allTags} selectedTags={["tag1", "tag2"]} onChange={onChange} />);
    const clearAllButton = screen.getByTitle("Clear all tags");
    fireEvent.click(clearAllButton);
    
    expect(onChange).toHaveBeenCalledWith([]);
  });

  it("closes dropdown when clicking outside", () => {
    render(
      <div>
        <div data-testid="outside">Outside</div>
        <TagFilter allTags={allTags} selectedTags={selectedTags} onChange={onChange} />
      </div>
    );
    
    const button = screen.getByRole("button", { name: /Filter by tags.../i });
    fireEvent.click(button);
    expect(screen.getByText("tag1")).toBeInTheDocument();
    
    fireEvent.mouseDown(screen.getByTestId("outside"));
    expect(screen.queryByText("tag1")).not.toBeInTheDocument();
  });

  it("shows 'No tags available' when allTags is empty", () => {
    render(<TagFilter allTags={[]} selectedTags={[]} onChange={onChange} />);
    const button = screen.getByRole("button", { name: /Filter by tags.../i });
    fireEvent.click(button);
    
    expect(screen.getByText("No tags available")).toBeInTheDocument();
  });
});
