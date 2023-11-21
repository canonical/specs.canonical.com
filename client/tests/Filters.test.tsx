import crypto from "crypto";
import Filters from "../Filters";
import { testSpec, testTeams } from "./__mocks__/utils";
import { fireEvent, render, screen } from "@testing-library/react";

Object.defineProperty(global, "crypto", {
  value: {
    getRandomValues: (arr: any) => crypto.randomBytes(arr.length),
  },
});

describe("renders filter component", () => {
  const authors = testSpec.authors;
  const teams = testTeams;
  const onChange = jest.fn();
  const defaultOptions = {
    team: teams[0],
    status: [testSpec.status],
    type: [testSpec.type],
    author: testSpec.authors[0],
    sortBy: "date",
  };

  beforeEach(() => {
    render(
      <Filters
        authors={authors}
        teams={teams}
        onChange={onChange}
        defaultOptions={defaultOptions}
      />
    );
  });

  it("selects a spec team, author and sort by from respective options", () => {
    // Team
    const teamLabel = screen.getByLabelText("Team");
    fireEvent.change(teamLabel, { target: { value: "test_team_1" } });
    const teamOption1 = screen.getByText("All teams") as HTMLOptionElement;
    const teamOption2 = screen.getByText("test_team_1") as HTMLOptionElement;
    const teamOption3 = screen.getByText("test_team_2") as HTMLOptionElement;
    expect(teamOption1.selected).toBeFalsy();
    expect(teamOption2.selected).toBeTruthy();
    expect(teamOption3.selected).toBeFalsy();

    // Author
    const authorLabel = screen.getByLabelText("Author");
    fireEvent.change(authorLabel, { target: { value: "test_author" } });
    const authorOption1 = screen.getByText("All authors") as HTMLOptionElement;
    const authorOption2 = screen.getByText("test_author") as HTMLOptionElement;
    expect(authorOption1.selected).toBeFalsy();
    expect(authorOption2.selected).toBeTruthy();

    // Sort by
    const sortLabel = screen.getByLabelText("Sort by");
    fireEvent.change(sortLabel, { target: { value: "date" } });
    const sortOption1 = screen.getByText("Last modified") as HTMLOptionElement;
    const sortOption2 = screen.getByText("Name") as HTMLOptionElement;
    const sortOption3 = screen.getByText("Spec index") as HTMLOptionElement;
    const sortOption4 = screen.getByText("Create date") as HTMLOptionElement;
    expect(sortOption1.selected).toBeTruthy();
    expect(sortOption2.selected).toBeFalsy();
    expect(sortOption3.selected).toBeFalsy();
    expect(sortOption4.selected).toBeFalsy();
  });

  it("checks a spec status and type", () => {
    // Status
    const activeCheckbox = screen.getByDisplayValue(
      "Active"
    ) as HTMLInputElement;
    expect(activeCheckbox.checked).toEqual(false);
    fireEvent.click(activeCheckbox);
    expect(activeCheckbox.checked).toEqual(true);

    //Type
    const standardCheckbox = screen.getByDisplayValue(
      "Standard"
    ) as HTMLInputElement;
    expect(standardCheckbox.checked).toEqual(false);
    fireEvent.click(standardCheckbox);
    expect(standardCheckbox.checked).toEqual(true);
  });
});
