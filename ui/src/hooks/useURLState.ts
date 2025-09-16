import qs from "qs";
import { useEffect, useRef, useState } from "react";

export type UserOptions = {
  filter: any;
  searchQuery: string | null;
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
        reviewer: searchState?.filter?.reviewer || null,
        orderBy: searchState?.filter?.orderBy || null,
      },
      searchQuery: searchState?.searchQuery || null,
    };
  };

  const [userOptions, setUserOptions] = useState<UserOptions>(
    decodeQueryParam()
  );

  const previousQueryParams = useRef<string | null>(null);

  useEffect(() => {
    const queryParams = qs.stringify(userOptions, {
      arrayFormat: "repeat",
      skipNulls: true,
      allowEmptyArrays: false,
    });
    if (
      previousQueryParams.current &&
      previousQueryParams.current === queryParams
    ) {
      return;
    }
    previousQueryParams.current = queryParams;

    if (queryParams) {
      const newURL = window.location.pathname + "?" + queryParams;
      history.pushState(null, "", newURL);
    } else {
      history.pushState(null, "", window.location.pathname);
    }
  }, [userOptions]);

  return { userOptions, setUserOptions };
};

export default useURLState;
