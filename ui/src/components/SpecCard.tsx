import clsx from "clsx";
import React from "react";
import type { Spec } from "../generated/types";

type SpecCardProps = {
  spec: Spec;
};

const SpecCard = ({ spec }: SpecCardProps) => {
  // const [viewSpecsDetails, setViewSpecsDetails] = useState<boolean>(false);
  const lastEdited = `Last edit: ${new Date(
    spec.google_doc_updated_at
  ).toLocaleDateString(undefined, {
    day: "numeric",
    month: "short",
    year: "numeric",
  })}`;

  const handleKeyDown = (event: React.KeyboardEvent) => {
    // if (event.key == "Enter") setViewSpecsDetails(true);
  };

  return (
    <>
      <div className="l-fluid-breakout__item" data-js="grid-item">
        <div
          className={`spec-card spec-card--${spec.status.toLowerCase()} p-card col-4 u-no-padding`}
        >
          <div className="spec-card__content p-card__inner">
            <div className="spec-card__header">
              <small className="spec-card__metadata-list">
                <ul className="p-inline-list--middot u-no-margin--bottom">
                  <li className="p-inline-list__item">{spec.id}</li>
                  <li className="p-inline-list__item">{spec.team}</li>
                  <li className="p-inline-list__item">{spec.spec_type}</li>
                </ul>
              </small>
              <div
                className={clsx("spec-card__status u-no-margin", {
                  "p-status-label--positive":
                    spec.status.toLowerCase() === "approved" ||
                    spec.status.toLowerCase() === "completed" ||
                    spec.status.toLowerCase() === "active",
                  "p-status-label--caution": spec.status
                    .toLowerCase()
                    .startsWith("pending"),
                  "p-status-label":
                    spec.status.toLowerCase() === "drafting" ||
                    spec.status.toLowerCase() === "braindump",
                  "p-status-label--negative":
                    spec.status.toLowerCase() === "rejected" ||
                    spec.status.toLowerCase() === "obsolete" ||
                    spec.status.toLowerCase() === "unknown",
                })}
              >
                {spec.status}
              </div>
            </div>
            <h3 className="p-heading--4 u-no-margin--bottom">
              <a
                href={spec.google_doc_url}
                onClick={(e) => {
                  if (e.metaKey || e.ctrlKey) {
                    return;
                  } else {
                    e.preventDefault();
                    // setViewSpecsDetails(true);
                  }
                }}
                onKeyDown={handleKeyDown}
              >
                {spec.title}
              </a>
            </h3>
            <small>
              <em>{spec.authors.join(", ")}</em>
            </small>
          </div>
          <div className="spec-card__footer p-card__inner">
            <em className="u-align--right">{lastEdited}</em>
          </div>
        </div>
      </div>
      {/* TODO */}
      {/* {viewSpecsDetails && (
        <SpecsDetails
          moreSpecDetails={{
            fileID: spec.fileID,
            folderName: spec.folderName,
            lastEdited,
          }}
          viewSpecsDetails={viewSpecsDetails}
          setViewSpecsDetails={setViewSpecsDetails}
        />
      )} */}
    </>
  );
};

export default SpecCard;
