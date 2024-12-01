import qs from "qs";
import { useEffect, useState } from "react";

export type UserOptions = {
  filter: any;
  searchQuery: string;
  offset?: number;
  limit?: number;
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
        team: searchState?.filter?.team,
        status: searchState?.filter?.status || [],
        type: searchState?.filter?.type || [],
        author: searchState?.filter?.author,
        sortBy: searchState?.filter?.sortBy || "date",
      },
      searchQuery: searchState?.searchQuery || "",
      offset: searchState?.offset ? parseInt(searchState.offset) : 0,
      limit: searchState?.limit ? parseInt(searchState.limit) : 10,
    };
  };

  const [userOptions, setUserOptions] = useState<UserOptions>(
    decodeQueryParam()
  );

  useEffect(() => {
    const queryParams = qs.stringify(userOptions);
    const newURL = window.location.pathname + "?" + queryParams;
    history.pushState(null, "", newURL);
  }, [userOptions]);

  return { userOptions, setUserOptions };
};

export default useURLState;
