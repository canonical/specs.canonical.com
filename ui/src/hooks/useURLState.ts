import qs from "qs";
import { useEffect, useState } from "react";

export type UserOptions = {
  filter: any;
  searchQuery: string;
};

const useURLState = () => {
  const decodeQueryParam = (): UserOptions => {
    const searchState = qs.parse(window.location.search.substring(1)) as {
      filter?: any;
      searchQuery?: string;
      offset?: string;
      limit?: string;
    };

    return {
      filter: {
        team: searchState?.filter?.team || null,
        status: searchState?.filter?.status || null,
        type: searchState?.filter?.type || null,
        author: searchState?.filter?.author || null,
        orderBy: searchState?.filter?.orderBy || null,
      },
      searchQuery: searchState?.searchQuery || "",
    };
  };

  const [userOptions, setUserOptions] = useState<UserOptions>(
    decodeQueryParam()
  );

  useEffect(() => {
    const queryParams = qs.stringify(userOptions, {
      arrayFormat: "repeat",
      skipNulls: true,
      allowEmptyArrays: false,
    });
    const newURL = window.location.pathname + "?" + queryParams;
    history.pushState(null, "", newURL);
  }, [userOptions]);

  return { userOptions, setUserOptions };
};

export default useURLState;
