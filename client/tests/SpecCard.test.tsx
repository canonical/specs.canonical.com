import SpecCard from "../SpecCard";
import { testSpec } from "./__mocks__/utils";
import { fireEvent, render, screen } from "@testing-library/react";

describe("renders spec card component", () => {
  let SpecCardComponent: any;

  beforeEach(() => {
    SpecCardComponent = render(<SpecCard spec={testSpec} />);
  });

  it("displays spec details on the card", () => {
    const specTitle = screen.getByText("test_title");
    expect(specTitle).toBeInTheDocument();
    const specIndex = screen.getByText("AB123");
    expect(specIndex).toBeInTheDocument();
    const specStatus = screen.getByText("active");
    expect(specStatus).toBeInTheDocument();
  });

  it("opens the spec preview on click of the spec title", async () => {
    const specTitle = screen.getByText("test_title");
    fireEvent.click(specTitle);
    const specPreview: any =
      SpecCardComponent.container.querySelector(".spec-aside");
    expect(specPreview).toBeInTheDocument();
  });

  it("opens the spec preview when Enter key is pressed", async () => {
    const specTitle = screen.getByText("test_title");
    fireEvent.keyDown(specTitle, { key: "Enter" });
    const specPreview: any =
      SpecCardComponent.container.querySelector(".spec-aside");
    expect(specPreview).toBeInTheDocument();
  });
});

describe("renders spec card with edgecases", () => {
  let SpecCardComponent: any;
  let testSpecClone: any;

  beforeEach(() => {
    testSpecClone = { ...testSpec };
  });

  it("shows a long index", async () => {
    testSpecClone.index = "ABC12345";
    SpecCardComponent = render(<SpecCard spec={testSpecClone} />);
    const specIndex = screen.getByText("ABC12345");
    expect(specIndex).toBeInTheDocument();
  });

  it("shows lowercase statuses", async () => {
    testSpecClone.status = "active";
    SpecCardComponent = render(<SpecCard spec={testSpecClone} />);
    const specStatus = screen.getByText("active");
    expect(specStatus).toHaveClass("p-status-label--positive");
  });
});
