import { useEffect, useState } from "react";

type WindowSize = [number, number];

export const useWindowSize = (): WindowSize => {
  const initSize: WindowSize = [window.innerWidth, window.innerHeight];
  const [windowSize, setWindowSize] = useState<WindowSize>(initSize);

  useEffect(() => {
    const handleResize = (): void => {
      setWindowSize([window.innerWidth, window.innerHeight]);
    };

    window.addEventListener("resize", handleResize);
    return () => {
      window.removeEventListener("resize", handleResize);
    };
  }, []);

  return windowSize;
};

export default useWindowSize;
