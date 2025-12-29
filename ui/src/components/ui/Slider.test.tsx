/**
 * Slider.test.tsx - Slider Component Tests
 *
 * Purpose: Test suite for Slider UI component.
 * Tests rendering, value changes, keyboard navigation, and formatting.
 *
 * Key Test Areas:
 * - Rendering: proper display of labels, values, and range input
 * - Value changes: onChange callback triggered correctly
 * - Keyboard navigation: arrow keys, Page Up/Down, Home/End
 * - Value formatting: custom formatters applied correctly
 * - Disabled state: interaction blocked when disabled
 * - Accessibility: ARIA attributes present
 *
 * Test Framework: Vitest with React Testing Library
 *
 * Usage:
 * ```bash
 * npm test -- Slider.test.tsx
 * ```
 *
 * Dependencies: vitest, @testing-library/react
 */

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { Slider } from "./Slider";

describe("Slider", () => {
  it("renders with basic props", () => {
    const onChange = vi.fn();
    render(<Slider value={50} onChange={onChange} min={0} max={100} step={10} />);

    const slider = screen.getByRole("slider");
    expect(slider).toBeInTheDocument();
    expect(slider).toHaveAttribute("min", "0");
    expect(slider).toHaveAttribute("max", "100");
    expect(slider).toHaveAttribute("step", "10");
    expect(slider).toHaveValue("50");
  });

  it("displays label and formatted value", () => {
    const onChange = vi.fn();
    render(
      <Slider
        value={500}
        onChange={onChange}
        min={100}
        max={1000}
        step={100}
        label="Timeout"
        formatValue={(v) => `${v}ms`}
      />,
    );

    expect(screen.getByText("Timeout")).toBeInTheDocument();
    expect(screen.getByText("500ms")).toBeInTheDocument();
  });

  it("displays end labels", () => {
    const onChange = vi.fn();
    render(
      <Slider
        value={50}
        onChange={onChange}
        min={0}
        max={100}
        step={10}
        leftLabel="Slower"
        rightLabel="Faster"
      />,
    );

    expect(screen.getByText("Slower")).toBeInTheDocument();
    expect(screen.getByText("Faster")).toBeInTheDocument();
  });

  it("calls onChange when slider value changes", () => {
    const onChange = vi.fn();
    render(<Slider value={50} onChange={onChange} min={0} max={100} step={10} />);

    const slider = screen.getByRole("slider");
    fireEvent.change(slider, { target: { value: "70" } });

    expect(onChange).toHaveBeenCalledWith(70);
  });

  it("handles keyboard navigation - Home key", () => {
    const onChange = vi.fn();
    render(<Slider value={50} onChange={onChange} min={0} max={100} step={10} />);

    const slider = screen.getByRole("slider");
    fireEvent.keyDown(slider, { key: "Home" });

    expect(onChange).toHaveBeenCalledWith(0);
  });

  it("handles keyboard navigation - End key", () => {
    const onChange = vi.fn();
    render(<Slider value={50} onChange={onChange} min={0} max={100} step={10} />);

    const slider = screen.getByRole("slider");
    fireEvent.keyDown(slider, { key: "End" });

    expect(onChange).toHaveBeenCalledWith(100);
  });

  it("handles keyboard navigation - PageUp key", () => {
    const onChange = vi.fn();
    render(<Slider value={50} onChange={onChange} min={0} max={200} step={10} />);

    const slider = screen.getByRole("slider");
    fireEvent.keyDown(slider, { key: "PageUp" });

    // PageUp increments by 10 steps (10 * 10 = 100)
    expect(onChange).toHaveBeenCalledWith(150);
  });

  it("handles keyboard navigation - PageDown key", () => {
    const onChange = vi.fn();
    render(<Slider value={150} onChange={onChange} min={0} max={200} step={10} />);

    const slider = screen.getByRole("slider");
    fireEvent.keyDown(slider, { key: "PageDown" });

    // PageDown decrements by 10 steps (10 * 10 = 100)
    expect(onChange).toHaveBeenCalledWith(50);
  });

  it("respects min boundary with PageDown", () => {
    const onChange = vi.fn();
    render(<Slider value={20} onChange={onChange} min={0} max={100} step={10} />);

    const slider = screen.getByRole("slider");
    fireEvent.keyDown(slider, { key: "PageDown" });

    // Should clamp to min (0) instead of going negative
    expect(onChange).toHaveBeenCalledWith(0);
  });

  it("respects max boundary with PageUp", () => {
    const onChange = vi.fn();
    render(<Slider value={90} onChange={onChange} min={0} max={100} step={10} />);

    const slider = screen.getByRole("slider");
    fireEvent.keyDown(slider, { key: "PageUp" });

    // Should clamp to max (100) instead of exceeding
    expect(onChange).toHaveBeenCalledWith(100);
  });

  it("is disabled when disabled prop is true", () => {
    const onChange = vi.fn();
    render(<Slider value={50} onChange={onChange} min={0} max={100} step={10} disabled />);

    const slider = screen.getByRole("slider");
    expect(slider).toBeDisabled();
  });

  it("has correct ARIA attributes", () => {
    const onChange = vi.fn();
    render(
      <Slider
        value={500}
        onChange={onChange}
        min={100}
        max={1000}
        step={100}
        label="Timeout"
        formatValue={(v) => `${v}ms`}
      />,
    );

    const slider = screen.getByRole("slider");
    expect(slider).toHaveAttribute("aria-valuemin", "100");
    expect(slider).toHaveAttribute("aria-valuemax", "1000");
    expect(slider).toHaveAttribute("aria-valuenow", "500");
    expect(slider).toHaveAttribute("aria-valuetext", "500ms");
    expect(slider).toHaveAttribute("aria-label", "Timeout");
  });

  it("formats values with custom formatter", () => {
    const onChange = vi.fn();
    const { rerender } = render(
      <Slider
        value={2000}
        onChange={onChange}
        min={500}
        max={10000}
        step={500}
        label="Duration"
        formatValue={(v) => (v >= 1000 ? `${v / 1000}s` : `${v}ms`)}
      />,
    );

    // Should show seconds
    expect(screen.getByText("2s")).toBeInTheDocument();

    // Update to value less than 1000
    rerender(
      <Slider
        value={500}
        onChange={onChange}
        min={500}
        max={10000}
        step={500}
        label="Duration"
        formatValue={(v) => (v >= 1000 ? `${v / 1000}s` : `${v}ms`)}
      />,
    );

    // Should show milliseconds
    expect(screen.getByText("500ms")).toBeInTheDocument();
  });

  it("applies custom className", () => {
    const onChange = vi.fn();
    const { container } = render(
      <Slider
        value={50}
        onChange={onChange}
        min={0}
        max={100}
        step={10}
        className="custom-class"
      />,
    );

    const wrapper = container.querySelector(".custom-class");
    expect(wrapper).toBeInTheDocument();
  });
});
