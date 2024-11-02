import { Navigation, Theme } from "@canonical/react-components";
import Filters from "./Filters";
import Search from "./Search";
import SpecCard from "./SpecCard";
import { Spec, Team } from "./types";
import { useFilteredAndSortedSpecs } from "./UserFilterOptions";
import { sortSet } from "./utils";
import SpecCardsList from "./SpecCardsList";

export const specTypes = new Set([
  "Implementation",
  "Product Requirement",
  "Standard",
  "Informational",
  "Process",
]);
export const specStatuses = new Set([
  "active",
  "approved",
  "braindump",
  "completed",
  "drafting",
  "obsolete",
  "pending review",
  "rejected",
]);
function App({ specs, teams }: { specs: Spec[]; teams: Team[] }) {
  specs = specs.map((spec) => ({
    ...spec,
    title: spec.title || "Unknown title",
    index: spec.index?.length >= 5 ? spec.index : "Unknown",
    status: specStatuses.has(spec.status.toLowerCase())
      ? spec.status
      : "Unknown",
    folderName: spec.folderName || "Unknown",
    type: specTypes.has(spec.type) ? spec.type : "Unknown",
    created: new Date(spec.created),
    lastUpdated: new Date(spec.lastUpdated),
  }));

  const { filteredSpecs, filter, searchQuery, setFilter, setSearchQuery } =
    useFilteredAndSortedSpecs(specs);
  const authors = new Set<string>();
  specs.forEach((spec) =>
    spec.authors.forEach((author) => authors.add(author))
  );

  return (
    <>
      <a href="#cards" className="p-link--skip">
        Jump to main content
      </a>
      <Navigation
        logo={{
          src: "https://assets.ubuntu.com/v1/82818827-CoF_white.svg",
          title: "Specifications",
          url: "#",
        }}
        items={[
          {
            label: "All Docs",
            url: "/",
          },
          {
            label: "My Docs",
            url: "/my-specs",
          },
        ]}
        itemsRight={[
          {
            url: "https://docs.google.com/document/d/1lStJjBGW7lyojgBhxGLUNnliUocYWjAZ1VEbbVduX54/edit#heading=h.31hys4te5m58",
            label: "How to add a new spec",
          },
        ]}
        theme={Theme.DARK}
      />
      <main className="l-fluid-breakout" id="main">
        <h1 className="u-off-screen">Canonical specifications</h1>
        <div className="l-fluid-breakout__toolbar u-no-margin--bottom">
          <div className="l-fluid-breakout__toolbar-items">
            <div className="l-fluid-breakout__toolbar-item">
              <span className="filtered-count"></span>
              {filteredSpecs.length}&nbsp;specs
            </div>
            <div className="l-fluid-breakout__toolbar-item">
              <Search onChange={setSearchQuery} defaultValue={searchQuery} />
            </div>
          </div>
        </div>

        <div className="l-fluid-breakout__aside">
          <Filters
            authors={sortSet(authors)}
            teams={sortSet(new Set(teams))}
            onChange={setFilter}
            defaultOptions={filter}
          />
        </div>
        <SpecCardsList specs={filteredSpecs} />
      </main>
      <footer className="p-strip is-shallow">
        <div className="row">
          <div className="col-12">
            © {new Date().getFullYear()} Canonical Ltd. Ubuntu and Canonical are
            registered trademarks of Canonical Ltd.
          </div>
        </div>
      </footer>
    </>
  );
}

export default App;
