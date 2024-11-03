import SpecCard from "./SpecCard";
import { Spec } from "./types";
import VirtualizedGrid, { VirtualizedGridItemProps } from "./VirtualGrid";

interface SpecCardsListProps {
  specs: Spec[];
}

const SpecCardsList = ({ specs }: SpecCardsListProps) => {
  if (!specs.length) {
    return (
      <div className="l-fluid-breakout__main">
        <h2 id="no-results" className="u-align-text--center">
          No specs found
        </h2>
      </div>
    );
  }

  return (
    <div
      className="l-fluid-breakout__main"
      // full width child as the grid is handled by the VirtualizedGrid component
      style={{ gridTemplateColumns: "1fr" }}
    >
      <VirtualizedGrid
        items={specs}
        itemHeight={230}
        itemMinWidth={400}
        gridSpace={16}
        renderItem={(props: VirtualizedGridItemProps<Spec>) =>
          specs[props.index] ? (
            <div>
              <SpecCard spec={specs[props.index]} style={props.style} />
            </div>
          ) : (
            <></>
          )
        }
      />
    </div>
  );
};

export default SpecCardsList;
