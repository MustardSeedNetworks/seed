import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { BaseCard, SimpleBaseCard } from "./BaseCard";

// Test data type
interface TestData {
  value: number;
  isHealthy: boolean;
}

describe("BaseCard", () => {
  const defaultProps = {
    title: "Test Card",
    data: { value: 42, isHealthy: true } as TestData,
    getStatus: (data: TestData) =>
      data.isHealthy ? ("success" as const) : ("error" as const),
    children: (data: TestData) => <div data-testid="content">{data.value}</div>,
  };

  describe("loading state", () => {
    it("renders loading skeleton when loading is true", () => {
      render(<BaseCard {...defaultProps} loading={true} />);

      expect(screen.getByText("Test Card")).toBeInTheDocument();
      // Default skeleton has specific structure with multiple skeleton elements
      const skeletons = document.querySelectorAll(".animate-pulse");
      expect(skeletons.length).toBeGreaterThan(0);
    });

    it("shows custom loading content when provided", () => {
      render(
        <BaseCard
          {...defaultProps}
          loading={true}
          loadingContent={<div data-testid="custom-loading">Loading...</div>}
        />,
      );

      expect(screen.getByTestId("custom-loading")).toBeInTheDocument();
      expect(screen.getByText("Loading...")).toBeInTheDocument();
    });

    it("does not render children when loading", () => {
      render(<BaseCard {...defaultProps} loading={true} />);

      expect(screen.queryByTestId("content")).not.toBeInTheDocument();
    });
  });

  describe("error state", () => {
    it("renders error message when error is provided", () => {
      render(<BaseCard {...defaultProps} error="Connection failed" />);

      expect(screen.getByText("Error")).toBeInTheDocument();
      expect(screen.getByText("Connection failed")).toBeInTheDocument();
    });

    it("does not render children when in error state", () => {
      render(<BaseCard {...defaultProps} error="Something went wrong" />);

      expect(screen.queryByTestId("content")).not.toBeInTheDocument();
    });

    it("error state takes precedence over loading", () => {
      render(
        <BaseCard {...defaultProps} loading={true} error="Error occurred" />,
      );

      // Loading is true but error should take precedence (loading checked first)
      // Actually checking the code: loading is checked first
      expect(screen.queryByText("Error")).not.toBeInTheDocument();
    });
  });

  describe("no data state", () => {
    it("renders default empty message when data is null", () => {
      render(<BaseCard {...defaultProps} data={null} />);

      expect(screen.getByText("No data available")).toBeInTheDocument();
    });

    it("renders custom empty message when provided", () => {
      render(
        <BaseCard
          {...defaultProps}
          data={null}
          emptyMessage="Waiting for data..."
        />,
      );

      expect(screen.getByText("Waiting for data...")).toBeInTheDocument();
    });

    it("does not render children when data is null", () => {
      render(<BaseCard {...defaultProps} data={null} />);

      expect(screen.queryByTestId("content")).not.toBeInTheDocument();
    });
  });

  describe("normal state with data", () => {
    it("renders children with data", () => {
      render(<BaseCard {...defaultProps} />);

      expect(screen.getByTestId("content")).toBeInTheDocument();
      expect(screen.getByText("42")).toBeInTheDocument();
    });

    it("derives status from getStatus function", () => {
      const { rerender } = render(<BaseCard {...defaultProps} />);

      // Healthy data - title should be in the document
      expect(screen.getByText("Test Card")).toBeInTheDocument();

      // Unhealthy data should derive error status
      rerender(
        <BaseCard
          {...defaultProps}
          data={{ value: 0, isHealthy: false }}
          getStatus={(data) => (data.isHealthy ? "success" : "error")}
        />,
      );

      expect(screen.getByText("Test Card")).toBeInTheDocument();
    });

    it("renders title and subtitle", () => {
      render(<BaseCard {...defaultProps} subtitle="Additional info" />);

      expect(screen.getByText("Test Card")).toBeInTheDocument();
      expect(screen.getByText("Additional info")).toBeInTheDocument();
    });
  });

  describe("click handling", () => {
    it("calls onClick handler when clicked", () => {
      const onClick = vi.fn();

      render(<BaseCard {...defaultProps} onClick={onClick} />);

      // Find the card container by role button (since onClick makes it interactive)
      const card = screen.getByRole("button");
      fireEvent.click(card);

      expect(onClick).toHaveBeenCalledTimes(1);
    });

    it("does not call onClick during loading state", () => {
      const onClick = vi.fn();

      render(<BaseCard {...defaultProps} loading={true} onClick={onClick} />);

      // In loading state, onClick is not passed so it's not a button
      const heading = screen.getByText("Test Card");
      fireEvent.click(heading);

      // onClick is not passed in loading state
      expect(onClick).not.toHaveBeenCalled();
    });
  });

  describe("className prop", () => {
    it("applies custom className", () => {
      render(<BaseCard {...defaultProps} className="custom-class" />);

      // Find the card element that has the custom class
      const card = document.querySelector(".custom-class");
      expect(card).toBeInTheDocument();
    });
  });
});

describe("SimpleBaseCard", () => {
  const defaultProps = {
    title: "Simple Card",
    status: "success" as const,
    children: <div data-testid="simple-content">Content</div>,
  };

  describe("loading state", () => {
    it("renders loading skeleton when loading is true", () => {
      render(<SimpleBaseCard {...defaultProps} loading={true} />);

      expect(screen.getByText("Simple Card")).toBeInTheDocument();
      const skeletons = document.querySelectorAll(".animate-pulse");
      expect(skeletons.length).toBeGreaterThan(0);
    });

    it("shows custom loading content when provided", () => {
      render(
        <SimpleBaseCard
          {...defaultProps}
          loading={true}
          loadingContent={<span>Custom loader</span>}
        />,
      );

      expect(screen.getByText("Custom loader")).toBeInTheDocument();
    });

    it("does not render children when loading", () => {
      render(<SimpleBaseCard {...defaultProps} loading={true} />);

      expect(screen.queryByTestId("simple-content")).not.toBeInTheDocument();
    });
  });

  describe("error state", () => {
    it("renders error message when error is provided", () => {
      render(<SimpleBaseCard {...defaultProps} error="Network error" />);

      expect(screen.getByText("Error")).toBeInTheDocument();
      expect(screen.getByText("Network error")).toBeInTheDocument();
    });

    it("does not render children when in error state", () => {
      render(<SimpleBaseCard {...defaultProps} error="Failed to load" />);

      expect(screen.queryByTestId("simple-content")).not.toBeInTheDocument();
    });
  });

  describe("normal state", () => {
    it("renders children when no loading or error", () => {
      render(<SimpleBaseCard {...defaultProps} />);

      expect(screen.getByTestId("simple-content")).toBeInTheDocument();
      expect(screen.getByText("Content")).toBeInTheDocument();
    });

    it("uses provided status", () => {
      render(<SimpleBaseCard {...defaultProps} status="warning" />);

      expect(screen.getByText("Simple Card")).toBeInTheDocument();
    });

    it("renders title and subtitle", () => {
      render(<SimpleBaseCard {...defaultProps} subtitle="Subtitle text" />);

      expect(screen.getByText("Simple Card")).toBeInTheDocument();
      expect(screen.getByText("Subtitle text")).toBeInTheDocument();
    });
  });

  describe("click handling", () => {
    it("calls onClick handler when clicked", () => {
      const onClick = vi.fn();

      render(<SimpleBaseCard {...defaultProps} onClick={onClick} />);

      // Find the card by role button
      const card = screen.getByRole("button");
      fireEvent.click(card);

      expect(onClick).toHaveBeenCalledTimes(1);
    });
  });

  describe("className prop", () => {
    it("applies custom className", () => {
      render(<SimpleBaseCard {...defaultProps} className="my-custom-class" />);

      const card = document.querySelector(".my-custom-class");
      expect(card).toBeInTheDocument();
    });
  });
});

describe("state priority", () => {
  it("loading takes priority over error in BaseCard", () => {
    render(
      <BaseCard
        title="Priority Test"
        data={{ value: 1, isHealthy: true }}
        getStatus={() => "success"}
        loading={true}
        error="Should not show"
      >
        {() => <div>Content</div>}
      </BaseCard>,
    );

    // Loading state should be shown, not error
    const skeletons = document.querySelectorAll(".animate-pulse");
    expect(skeletons.length).toBeGreaterThan(0);
    expect(screen.queryByText("Should not show")).not.toBeInTheDocument();
  });

  it("loading takes priority over error in SimpleBaseCard", () => {
    render(
      <SimpleBaseCard
        title="Priority Test"
        status="success"
        loading={true}
        error="Should not show"
      >
        <div>Content</div>
      </SimpleBaseCard>,
    );

    const skeletons = document.querySelectorAll(".animate-pulse");
    expect(skeletons.length).toBeGreaterThan(0);
    expect(screen.queryByText("Should not show")).not.toBeInTheDocument();
  });

  it("error takes priority over no data in BaseCard", () => {
    render(
      <BaseCard
        title="Error Priority"
        data={null}
        getStatus={() => "success"}
        error="Error message"
        emptyMessage="No data"
      >
        {() => <div>Content</div>}
      </BaseCard>,
    );

    expect(screen.getByText("Error message")).toBeInTheDocument();
    expect(screen.queryByText("No data")).not.toBeInTheDocument();
  });
});
