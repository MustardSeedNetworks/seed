import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Card, CardValue, CardRow, CardDivider } from "./Card";

describe("Card", () => {
  it("renders with title and children", () => {
    render(
      <Card title="Test Card" subtitle="Sub" status="success">
        <span>Card content</span>
      </Card>,
    );

    expect(screen.getByText("Test Card")).toBeInTheDocument();
    expect(screen.getByText("Sub")).toBeInTheDocument();
    expect(screen.getByText("Card content")).toBeInTheDocument();
  });

  it("applies correct status styling for success", () => {
    render(
      <Card title="Success Card" status="success">
        <span>Content</span>
      </Card>,
    );

    const statusIcon = screen.getByLabelText("Status: success");
    expect(statusIcon).toHaveClass("text-status-success");
  });

  it("applies correct status styling for warning", () => {
    render(
      <Card title="Warning Card" status="warning">
        <span>Content</span>
      </Card>,
    );

    const statusIcon = screen.getByLabelText("Status: warning");
    expect(statusIcon).toHaveClass("text-status-warning");
  });

  it("applies correct status styling for error", () => {
    render(
      <Card title="Error Card" status="error">
        <span>Content</span>
      </Card>,
    );

    const statusIcon = screen.getByLabelText("Status: error");
    expect(statusIcon).toHaveClass("text-status-error");
  });

  it("shows unknown status correctly", () => {
    render(
      <Card title="Unknown Card" status="unknown">
        <span>Content</span>
      </Card>,
    );

    const statusIcon = screen.getByLabelText("Status: unknown");
    expect(statusIcon).toHaveClass("text-text-muted");
  });

  it("shows loading status with animation", () => {
    render(
      <Card title="Loading Card" status="loading">
        <span>Content</span>
      </Card>,
    );

    const statusIcon = screen.getByLabelText("Status: loading");
    const spinner = statusIcon.querySelector(".animate-spin");
    expect(spinner).toBeInTheDocument();
  });

  it("handles click events", () => {
    const handleClick = vi.fn();
    render(
      <Card title="Clickable Card" status="success" onClick={handleClick}>
        <span>Content</span>
      </Card>,
    );

    const card = screen.getByText("Clickable Card").closest("div");
    fireEvent.click(card!);
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it("applies custom className", () => {
    const { container } = render(
      <Card title="Custom Class Card" status="success" className="custom-class">
        <span>Content</span>
      </Card>,
    );

    // The outermost div is the Card container
    const card = container.firstChild;
    expect(card).toHaveClass("custom-class");
  });
});

describe("CardValue", () => {
  it("renders value without label", () => {
    render(<CardValue value="100" />);
    expect(screen.getByText("100")).toBeInTheDocument();
  });

  it("renders value with label", () => {
    render(<CardValue label="Speed" value="100" />);
    expect(screen.getByText("Speed")).toBeInTheDocument();
    expect(screen.getByText("100")).toBeInTheDocument();
  });

  it("renders value with unit", () => {
    render(<CardValue value="100" unit="Mbps" />);
    expect(screen.getByText("100")).toBeInTheDocument();
    expect(screen.getByText("Mbps")).toBeInTheDocument();
  });

  it("applies size classes correctly", () => {
    const { rerender } = render(<CardValue value="100" size="sm" />);
    expect(screen.getByTestId("card-value")).toHaveClass("text-sm");

    rerender(<CardValue value="100" size="md" />);
    expect(screen.getByTestId("card-value")).toHaveClass("text-base");

    rerender(<CardValue value="100" size="lg" />);
    expect(screen.getByTestId("card-value")).toHaveClass("text-lg");
  });

  it("applies status color", () => {
    render(<CardValue value="Error" status="error" />);
    expect(screen.getByTestId("card-value")).toHaveClass("text-status-error");
  });
});

describe("CardRow", () => {
  it("renders label and value", () => {
    render(<CardRow label="Latency" value="15ms" />);
    expect(screen.getByText("Latency")).toBeInTheDocument();
    expect(screen.getByText("15ms")).toBeInTheDocument();
  });

  it("renders numeric value", () => {
    render(<CardRow label="Count" value={42} />);
    expect(screen.getByText("Count")).toBeInTheDocument();
    expect(screen.getByText("42")).toBeInTheDocument();
  });

  it("applies status color to value", () => {
    render(<CardRow label="Status" value="Failed" status="error" />);
    expect(screen.getByTestId("card-row-value")).toHaveClass(
      "text-status-error",
    );
  });

  it("sets title attribute for truncation", () => {
    render(<CardRow label="Long Value" value="This is a very long value" />);
    const valueElement = screen.getByTestId("card-row-value");
    expect(valueElement).toHaveAttribute("title", "This is a very long value");
  });
});

describe("CardDivider", () => {
  it("renders a divider", () => {
    const { container } = render(<CardDivider />);
    const hr = container.querySelector("hr");
    expect(hr).toBeInTheDocument();
    expect(hr).toHaveClass("border-surface-border");
  });

  it("applies custom className", () => {
    const { container } = render(<CardDivider className="my-custom-class" />);
    const hr = container.querySelector("hr");
    expect(hr).toHaveClass("my-custom-class");
  });
});
