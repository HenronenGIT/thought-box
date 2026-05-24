"use client";

import { mediaQueries } from "./breakpoints";
import { useMediaQuery } from "./useMediaQuery";

export function useIsMobile() {
  return useMediaQuery(mediaQueries.mobile);
}
