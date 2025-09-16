import {
  Navigation,
  Notification,
  SearchBox,
  Spinner,
  Theme,
} from "@canonical/react-components";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import qs from "qs";
import InfiniteScroll from "react-infinite-scroll-component";
import Filters from "../components/Filters";
import { SpecCard } from "../components/SpecCard";
import type { ListSpecsResponse } from "../generated/types";
import useURLState from "../hooks/useURLState";
import { sortedSet } from "../utils";

const LIMIT = 50;

export const SPEC_TYPES = new Set([
  "Implementation",
  "Product Requirement",
  "Standard",
  "Informational",
  "Process",
]);
export const SPEC_STATUSES = new Set([
  "Active",
  "Approved",
  "Braindump",
  "Completed",
  "Drafting",
  "Obsolete",
  "Pending review",
  "Rejected",
]);

function Specs() {
  const { userOptions, setUserOptions } = useURLState();
  const { data, fetchNextPage, hasNextPage, error, isLoading } =
    useInfiniteQuery({
      queryKey: ["specs", userOptions.filter, userOptions.searchQuery],
      queryFn: async ({ pageParam = 0 }) => {
        const params = {
          ...userOptions.filter,
          orderBy: userOptions.filter.orderBy || "updated_at",
          searchQuery: userOptions.searchQuery || "",
          offset: pageParam,
          limit: LIMIT,
        };
        const queryString = qs.stringify(params, {
          arrayFormat: "repeat",
          skipNulls: true,
          allowEmptyArrays: false,
        });
        const res = await fetch(`/api/specs?${queryString}`);
        const data = (await res.json()) as Promise<ListSpecsResponse>;
        if (!res.ok) {
          throw new Error((data as unknown as { message: string }).message);
        }
        return data;
      },
      getNextPageParam: (lastPage, pages) =>
        lastPage.specs?.length === LIMIT ? pages.length * LIMIT : undefined,
      initialPageParam: 0,
    });

  const totalSpecs = data?.pages[0]?.total || 0;
  const allSpecs =
    data?.pages.flatMap((page) => page.specs).filter(Boolean) || [];
  const { data: authorsData } = useQuery({
    queryKey: ["authors"],
    queryFn: async () => {
      const res = await fetch("/api/specs/authors");
      return res.json() as Promise<string[]>;
    },
    refetchOnWindowFocus: false,
    refetchOnMount: false,
  });

  const { data: reviewersData } = useQuery({
    queryKey: ["reviewers"],
    queryFn: async () => {
      const res = await fetch("/api/specs/reviewers");
      return res.json() as Promise<string[]>;
    },
    refetchOnWindowFocus: false,
    refetchOnMount: false,
  });

  const { data: teamsData } = useQuery({
    queryKey: ["teams"],
    queryFn: async () => {
      const res = await fetch("/api/specs/teams");
      return res.json() as Promise<string[]>;
    },
    refetchOnWindowFocus: false,
    refetchOnMount: false,
  });
  const authors = authorsData || [];
  const reviewers = reviewersData || [];
  const teams = teamsData || [];

  return (
    <>
      <a href="#cards" className="p-link--skip">
        Jump to main content
      </a>
      <Navigation
        logo={{
          src: "https://assets.ubuntu.com/v1/82818827-CoF_white.svg",
          title: "Specifications",
          url: "/",
        }}
        items={
          [
            // TODO: maybe add these later
            // { label: "All Docs", url: "/" },
            // { label: "My Docs", url: "/my-specs" },
          ]
        }
        itemsRight={[
          {
            url: "https://docs.google.com/document/d/1lStJjBGW7lyojgBhxGLUNnliUocYWjAZ1VEbbVduX54/edit#heading=h.31hys4te5m58",
            label: "How to add a new spec",
          },
        ]}
        theme={Theme.DARK}
      />
      <main className="l-fluid-breakout">
        <div className="l-fluid-breakout__toolbar">
          <div className="l-fluid-breakout__toolbar-items">
            <h4 className="l-fluid-breakout__toolbar-item p-muted-heading">
              {totalSpecs} specs
            </h4>
            <div className="l-fluid-breakout__toolbar-item">
              <input
                type="search"
                value={userOptions.searchQuery || undefined}
                onChange={(e) =>
                  setUserOptions({
                    ...userOptions,
                    searchQuery: e.target.value,
                  })
                }
                placeholder="Search specs..."
                className="p-search-box__input"
              />
            </div>
          </div>
        </div>
        <div className="l-fluid-breakout__aside sticky-sidebar">
          <Filters
            authors={sortedSet(new Set(authors))}
            teams={sortedSet(new Set(teams))}
            reviewers={sortedSet(new Set(reviewers))}
            userOptions={userOptions}
            setUserOptions={setUserOptions}
          />
        </div>
        {isLoading && <Spinner text="Loading..." />}
        {error && (
          <Notification severity="negative" title="Error fetching specs">
            <pre>{error.message}</pre>
          </Notification>
        )}
        <div className="l-fluid-breakout__main" id="cards">
          <InfiniteScroll
            dataLength={allSpecs.length}
            next={fetchNextPage}
            hasMore={!!hasNextPage}
            loader={<p className="u-align--center">Loading more specs...</p>}
          >
            {allSpecs.map((spec, index) => (
              <SpecCard key={`${spec.id}-${index}`} spec={spec} />
            ))}
          </InfiniteScroll>
        </div>
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

export default Specs;
