import {
  CheckboxInput,
  CustomSelect,
  MultiSelect,
  Select,
} from "@canonical/react-components";
import { useFormik } from "formik";
import { useEffect } from "react";
import type { UserOptions } from "../hooks/useURLState";
import { SPEC_STATUSES, SPEC_TYPES } from "../pages/Specs";

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
      <CustomSelect
        value={formik.values.team}
        label="Team"
        name="team"
        id="team"
        searchable="always"
        options={[
          { value: "", label: "All teams" },
          ...teams.map((team) => ({ label: team, value: team })),
        ]}
        onChange={(value) => formik.setFieldValue("team", value)}
      />
      <p className="u-no-margin--bottom">Status</p>
      {/* @ts-ignore: for some reason ReactNode type is not working */}
      <MultiSelect
        placeholder="Select status"
        variant="condensed"
        items={[...SPEC_STATUSES].map((status) => ({
          label: status,
          value: status,
        }))}
        selectedItems={
          typeof formik.values.status === "string"
            ? [
                {
                  label: formik.values.status,
                  value: formik.values.status,
                },
              ]
            : formik.values.status
            ? formik.values.status.map((status: string) => ({
                label: status,
                value: status,
              }))
            : []
        }
        onItemsUpdate={(items) => {
          formik.setFieldValue("status", [
            ...new Set([...items.map((item) => item.value)]),
          ]);
        }}
      />
      <p className="u-no-margin--bottom">Type</p>
      {[...SPEC_TYPES].map((typeName) => (
        <CheckboxInput
          key={typeName}
          label={typeName}
          value={typeName}
          name="type"
          onChange={formik.handleChange}
          checked={formik.values.type?.includes(typeName)}
        />
      ))}
      <CustomSelect
        value={formik.values.author}
        label="Author"
        name="author"
        id="author"
        searchable="always"
        options={[
          { value: "", label: "All authors" },
          ...authors.map((author) => ({ label: author, value: author })),
        ]}
        onChange={(value) => formik.setFieldValue("author", value)}
      />
      <Select
        value={formik.values.orderBy}
        label="Sort by"
        name="orderBy"
        id="orderBy"
        options={[
          { value: "updated_at", label: "Last modified" },
          { value: "created_at", label: "Create date" },
          { value: "title", label: "Name" },
          { value: "id", label: "Spec index" },
        ]}
        onChange={formik.handleChange}
      />
    </form>
  );
};

export default Filters;
