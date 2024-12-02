import { Button } from "@canonical/react-components";
import clsx from "clsx";
import FocusTrap from "focus-trap-react";
import React, { useEffect, useState } from "react";
import type { Spec } from "../generated/types";

type SpecCardProps = {
  spec: Spec;
};

const SpecCard = ({ spec }: SpecCardProps) => {
  const [viewSpecsDetails, setViewSpecsDetails] = useState(false);
  const lastEdited = `Last edit: ${new Date(
    spec.google_doc_updated_at
  ).toLocaleDateString(undefined, {
    day: "numeric",
    month: "short",
    year: "numeric",
  })}`;

  const handleKeyDown = (event: React.KeyboardEvent) => {
    if (event.key == "Enter") setViewSpecsDetails(true);
  };

  useEffect(() => {
    // disable global scroll when modal is open
    if (viewSpecsDetails) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "auto";
    }
  }, [viewSpecsDetails]);

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
                    setViewSpecsDetails(true);
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
      {viewSpecsDetails && (
        <FocusTrap
          active={viewSpecsDetails}
          focusTrapOptions={{ fallbackFocus: ".spec-aside-backdrop" }}
        >
          <div
            className="spec-aside-backdrop"
            onClick={() => setViewSpecsDetails(false)}
          >
            <aside
              className="spec-aside l-aside is-wide"
              role="dialog"
              aria-modal="true"
              aria-labelledby="spec-preview"
              aria-describedby="spec-preview"
              onClick={(e: React.SyntheticEvent) => e.stopPropagation()}
            >
              <div className="spec-container">
                <section className="p-strip is-bordered is-shallow">
                  <small className="spec-card__metadata-list">
                    <ul className="header p-inline-list--middot u-no-margin--bottom">
                      <li className="p-inline-list__item">{spec.id}</li>
                      <li className="p-inline-list__item">{spec.team}</li>
                      <li className="p-inline-list__item metadata-type">
                        {spec.spec_type}
                      </li>
                    </ul>
                    <Button
                      appearance="brand"
                      element="a"
                      href={spec.google_doc_url}
                      target="_blank"
                      className="u-no-margin--bottom"
                    >
                      Open in Google Docs
                    </Button>
                  </small>
                  <button
                    className="p-modal__close"
                    aria-label="Close spec preview"
                    onClick={() => setViewSpecsDetails(false)}
                  >
                    Close
                  </button>
                </section>
                <div className="spec-preview">
                  <iframe
                    title={spec.title}
                    src={
                      spec.google_doc_url.replace("edit", "preview") +
                      "&embedded=true"
                    }
                    height="100%"
                    width="100%"
                  ></iframe>
                </div>
              </div>
            </aside>
          </div>
        </FocusTrap>
      )}
    </>
  );
};

export default SpecCard;
