import { Button, Icon, Spinner } from "@canonical/react-components";
import clsx from "clsx";
import FocusTrap from "focus-trap-react";
import React from "react";
import type { Spec } from "../../generated/types";
import "./styles.scss";

type SpecPreviewSidePanelProps = {
  viewSpecsDetails: boolean;
  onClose: () => void;
  spec: Spec;
};

const SpecPreviewSidePanel = ({
  viewSpecsDetails,
  onClose,
  spec,
}: SpecPreviewSidePanelProps) => {
  if (!viewSpecsDetails) {
    return null;
  }
  return (
    <FocusTrap
      active={viewSpecsDetails}
      focusTrapOptions={{ fallbackFocus: ".spec-aside-backdrop" }}
    >
      <div className={clsx("spec-aside-backdrop")} onClick={onClose}>
        <aside
          className="spec-aside l-aside is-wide"
          role="dialog"
          aria-modal="true"
          aria-labelledby="spec-preview"
          aria-describedby="spec-preview"
          onClick={(e: React.SyntheticEvent) => e.stopPropagation()}
        >
          <div className="spec-container">
            <section className="spec-header">
              <small className="spec-card__metadata-list">
                <ul className="header p-inline-list--middot u-no-margin--bottom">
                  <li className="p-inline-list__item">{spec.id}</li>
                  <li className="p-inline-list__item u-truncate">
                    {spec.team}
                  </li>
                  <li className="p-inline-list__item metadata-type  u-truncate">
                    {spec.spec_type}
                  </li>
                </ul>
                <Button
                  appearance="brand"
                  element="a"
                  href={spec.google_doc_url}
                  target="_blank"
                  className="u-no-margin--bottom u-truncate"
                >
                  Open in Google Docs
                </Button>
              </small>
              <Button
                className="u-no-margin--bottom"
                hasIcon
                onClick={onClose}
                appearance="base"
              >
                <Icon name="close" />
              </Button>
            </section>
            {spec.google_doc_url && (
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
                <Spinner text="Loading..." className="spec-preview__spinner" />
              </div>
            )}
          </div>
        </aside>
      </div>
    </FocusTrap>
  );
};

export default SpecPreviewSidePanel;
