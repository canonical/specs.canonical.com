import { CheckboxInput, Select } from "@canonical/react-components";
import { useFormik } from "formik";
import React, { useEffect } from "react";
import { SPEC_STATUSES, SPEC_TYPES } from "../pages/Specs";
import type { UserOptions } from "../hooks/useURLState";

type FiltersProps = {
  authors: string[];
  teams: string[];
  userOptions: UserOptions;
  setUserOptions: (options: UserOptions) => void;
};

const Filters = ({
  authors,
  teams,
  userOptions,
  setUserOptions,
}: FiltersProps) => {
  const formik = useFormik({
    initialValues: userOptions.filter,
    onSubmit: (values) => {
      setUserOptions({ ...userOptions, filter: values });
    },
    enableReinitialize: true,
  });

  useEffect(() => {
    setUserOptions({ ...userOptions, filter: formik.values });
  }, [formik.values]);

  return (
    <form onSubmit={formik.handleSubmit}>
      <Select
        value={formik.values.team}
        label="Team"
        name="team"
        id="team"
        options={[
          { value: "", label: "All teams" },
          ...teams.map((team) => ({ label: team, value: team })),
        ]}
        onChange={formik.handleChange}
      />
      <p className="u-no-margin--bottom">Status</p>
      {[...SPEC_STATUSES].map((status) => (
        <CheckboxInput
          key={status}
          label={status}
          name="status"
          value={status}
          onChange={formik.handleChange}
          checked={formik.values.status.includes(status)}
        />
      ))}
      <p className="u-no-margin--bottom">Type</p>
      {[...SPEC_TYPES].map((typeName) => (
        <CheckboxInput
          key={typeName}
          label={typeName}
          value={typeName}
          name="type"
          onChange={formik.handleChange}
          checked={formik.values.type.includes(typeName)}
        />
      ))}
      <Select
        value={formik.values.author}
        label="Author"
        name="author"
        id="author"
        options={[
          { value: "", label: "All authors" },
          ...authors.map((author) => ({ label: author, value: author })),
        ]}
        onChange={formik.handleChange}
      />
      <Select
        value={formik.values.sortBy}
        label="Sort by"
        name="sortBy"
        id="sortBy"
        options={[
          { value: "date", label: "Last modified" },
          { value: "created", label: "Create date" },
          { value: "name", label: "Name" },
          { value: "index", label: "Spec index" },
        ]}
        onChange={formik.handleChange}
      />
    </form>
  );
};

export default Filters;
