export const breakpoints = {
  mobileMaxPx: 760,
} as const;

export const mediaQueries = {
  mobile: `(max-width: ${breakpoints.mobileMaxPx}px)`,
} as const;
