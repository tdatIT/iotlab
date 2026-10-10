import type { SVGProps } from "react";

// Heroicons has no thermometer; drawn on the same 24px grid as its outline/solid sets.
const BULB_TUBE =
  "M9.75 14.06V4.5a2.25 2.25 0 0 1 4.5 0v9.56a4.5 4.5 0 1 1-4.5 0Z";

export function ThermometerIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth={1.5}
      stroke="currentColor"
      aria-hidden="true"
      {...props}
    >
      <path strokeLinecap="round" strokeLinejoin="round" d={BULB_TUBE} />
      <path strokeLinecap="round" d="M12 9v8.25" />
      <circle cx={12} cy={17.25} r={1.5} fill="currentColor" stroke="none" />
    </svg>
  );
}

export function ThermometerIconSolid(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="currentColor"
      aria-hidden="true"
      {...props}
    >
      <path d={BULB_TUBE} stroke="currentColor" strokeWidth={1.5} strokeLinejoin="round" />
    </svg>
  );
}
