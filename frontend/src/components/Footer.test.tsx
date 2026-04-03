import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import Footer from "./Footer";

describe("Footer", () => {
  it("renders claimctl branding", () => {
    render(<Footer />);

    expect(screen.getByText("claimctl")).toBeInTheDocument();
  });

  it("displays current year in copyright", () => {
    render(<Footer />);

    const currentYear = new Date().getFullYear();
    expect(screen.getByText(new RegExp(`${currentYear}`))).toBeInTheDocument();
  });

  it("renders all navigation links", () => {
    render(<Footer />);

    expect(screen.getByText("About")).toBeInTheDocument();
    expect(screen.getByText("Contact")).toBeInTheDocument();
    expect(screen.getByText("Terms of Service")).toBeInTheDocument();
    expect(screen.getByText("Privacy Policy")).toBeInTheDocument();
  });

  it("links have correct href attributes", () => {
    render(<Footer />);

    expect(screen.getByText("About").closest("a")).toHaveAttribute(
      "href",
      "#about",
    );
    expect(screen.getByText("Contact").closest("a")).toHaveAttribute(
      "href",
      "#contact",
    );
    expect(screen.getByText("Terms of Service").closest("a")).toHaveAttribute(
      "href",
      "#terms",
    );
    expect(screen.getByText("Privacy Policy").closest("a")).toHaveAttribute(
      "href",
      "#privacy",
    );
  });

  it("renders as a footer element", () => {
    const { container } = render(<Footer />);

    expect(container.querySelector("footer")).toBeInTheDocument();
  });
});
