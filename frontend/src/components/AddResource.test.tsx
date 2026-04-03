import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../test/utils";
import AddResourcePopup from "./AddResource";

describe("AddResourcePopup", () => {
    it("renders the popup", () => {
        renderWithProviders(<AddResourcePopup isOpen={true} onClose={() => {}} onSave={() => {}} />);
        expect(screen.getByText("Add New Resource")).toBeInTheDocument();
    });

    it("Resource name is visible", () => {
        renderWithProviders(<AddResourcePopup isOpen={true} onClose={() => {}} onSave={() => {}} />);
        expect(screen.getByText("Resource Name")).toBeInTheDocument();
    });

    it("Resource type is visible", () => {
        renderWithProviders(<AddResourcePopup isOpen={true} onClose={() => {}} onSave={() => {}} />);
        expect(screen.getByText("Resource Type")).toBeInTheDocument();
    });

    it("Resource labels is visible", () => {
        renderWithProviders(<AddResourcePopup isOpen={true} onClose={() => {}} onSave={() => {}} />);
        expect(screen.getByText("Labels (comma-separated)")).toBeInTheDocument();
    });

    it("Add property button is visible", () => {
        renderWithProviders(<AddResourcePopup isOpen={true} onClose={() => {}} onSave={() => {}} />);
        expect(screen.getByText("+ Add Property")).toBeInTheDocument();
    });

    it("Space selection is visible", () => {
        renderWithProviders(<AddResourcePopup isOpen={true} onClose={() => {}} onSave={() => {}} />);
        expect(screen.getByText("Space (Optional, defaults to Default Space)")).toBeInTheDocument();
    });
});