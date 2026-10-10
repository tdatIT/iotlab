import type { SVGProps } from "react";

/** Heroicons has no snowflake; drawn in the same 24px outline style. */
export function SnowflakeIcon(props: SVGProps<SVGSVGElement>) {
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
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M12 2.25v19.5M3.56 7.13l16.88 9.74M3.56 16.87l16.88-9.74M9.75 3.75 12 6l2.25-2.25M9.75 20.25 12 18l2.25 2.25M4.4 10.2l3.07-.82-.82-3.08M19.6 13.8l-3.07.82.82 3.08M4.4 13.8l3.07.82-.82 3.08M19.6 10.2l-3.07-.82.82-3.08"
      />
    </svg>
  );
}
