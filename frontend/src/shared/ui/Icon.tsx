import type { ReactElement, SVGProps } from "react";

/**
 * Lightweight inline-SVG icon set in a single file. Stroke-based,
 * outline style — matches the rest of the Liquid Glass redesign.
 *
 * Conventions:
 *   - 24x24 viewBox, currentColor stroke, stroke-width 1.7.
 *   - Components inherit text color so they paint via CSS without props.
 *   - Add new glyphs to the `paths` map below; React picks them up.
 */

export type IconName =
  | "home"
  | "dashboard"
  | "career"
  | "mic"
  | "report"
  | "user"
  | "shield"
  | "resume"
  | "settings"
  | "logout"
  | "sun"
  | "moon"
  | "system"
  | "github"
  | "bell"
  | "credit"
  | "sparkles"
  | "chevron-right"
  | "chart"
  | "users"
  | "audit";

const paths: Record<IconName, ReactElement> = {
  home: (
    <>
      <path d="M3 11l9-8 9 8v9a2 2 0 0 1-2 2h-4v-7h-6v7H5a2 2 0 0 1-2-2v-9z" />
    </>
  ),
  dashboard: (
    <>
      <rect x="3" y="3" width="8" height="10" rx="2" />
      <rect x="13" y="3" width="8" height="6" rx="2" />
      <rect x="13" y="11" width="8" height="10" rx="2" />
      <rect x="3" y="15" width="8" height="6" rx="2" />
    </>
  ),
  career: (
    <>
      <path d="M3 7h18M3 12h18M3 17h12" />
      <circle cx="18" cy="17" r="3" />
    </>
  ),
  mic: (
    <>
      <rect x="9" y="3" width="6" height="11" rx="3" />
      <path d="M5 11a7 7 0 0 0 14 0M12 18v3M8 21h8" />
    </>
  ),
  report: (
    <>
      <path d="M5 3h11l4 4v14a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1z" />
      <path d="M16 3v4h4M8 13h8M8 17h6M8 9h3" />
    </>
  ),
  user: (
    <>
      <circle cx="12" cy="8" r="4" />
      <path d="M4 21c0-4 4-7 8-7s8 3 8 7" />
    </>
  ),
  shield: (
    <>
      <path d="M12 3l8 3v6c0 5-3.5 8-8 9-4.5-1-8-4-8-9V6l8-3z" />
    </>
  ),
  resume: (
    <>
      <rect x="5" y="3" width="14" height="18" rx="2" />
      <circle cx="12" cy="9" r="2.4" />
      <path d="M8 16c1-2 3-3 4-3s3 1 4 3" />
    </>
  ),
  settings: (
    <>
      <circle cx="12" cy="12" r="3" />
      <path d="M19.4 14a1.6 1.6 0 0 0 .3 1.7l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.6 1.6 0 0 0-1.7-.3 1.6 1.6 0 0 0-1 1.5V20a2 2 0 1 1-4 0v-.1a1.6 1.6 0 0 0-1-1.5 1.6 1.6 0 0 0-1.7.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.6 1.6 0 0 0 .3-1.7 1.6 1.6 0 0 0-1.5-1H4a2 2 0 1 1 0-4h.1a1.6 1.6 0 0 0 1.5-1 1.6 1.6 0 0 0-.3-1.7l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.6 1.6 0 0 0 1.7.3H10a1.6 1.6 0 0 0 1-1.5V4a2 2 0 1 1 4 0v.1a1.6 1.6 0 0 0 1 1.5 1.6 1.6 0 0 0 1.7-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.6 1.6 0 0 0-.3 1.7V10a1.6 1.6 0 0 0 1.5 1H20a2 2 0 1 1 0 4h-.1a1.6 1.6 0 0 0-1.5 1z" />
    </>
  ),
  logout: (
    <>
      <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9" />
    </>
  ),
  sun: (
    <>
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M4 12H2M22 12h-2M5.6 5.6l1.4 1.4M17 17l1.4 1.4M5.6 18.4l1.4-1.4M17 7l1.4-1.4" />
    </>
  ),
  moon: (
    <>
      <path d="M21 13A9 9 0 0 1 11 3a7 7 0 1 0 10 10z" />
    </>
  ),
  system: (
    <>
      <rect x="3" y="4" width="18" height="12" rx="2" />
      <path d="M8 20h8M12 16v4" />
    </>
  ),
  github: (
    <>
      <path d="M9 19c-4 1.5-4-2-6-2.5M15 22v-3.5a3 3 0 0 0-.9-2.3c2.7-.3 5.5-1.3 5.5-6 0-1.2-.5-2.4-1.3-3.3.4-1.2.4-2.5-.1-3.7 0 0-1-.3-3.4 1.3a11.6 11.6 0 0 0-6 0C6.4 2 5.4 2.3 5.4 2.3 4.9 3.5 4.9 4.8 5.3 6 4.5 6.9 4 8.1 4 9.3c0 4.7 2.8 5.7 5.5 6a3 3 0 0 0-.9 2.3V22" />
    </>
  ),
  bell: (
    <>
      <path d="M6 8a6 6 0 1 1 12 0c0 7 3 9 3 9H3s3-2 3-9M10.3 21a2 2 0 0 0 3.4 0" />
    </>
  ),
  credit: (
    <>
      <rect x="2.5" y="5" width="19" height="14" rx="2" />
      <path d="M2.5 10h19M6.5 15h3" />
    </>
  ),
  sparkles: (
    <>
      <path d="M12 3l1.6 4.4L18 9l-4.4 1.6L12 15l-1.6-4.4L6 9l4.4-1.6L12 3zM18 14l.9 2.5L21.4 17l-2.5.9L18 20l-.9-2.5-2.5-.5L17.1 16l.9-2z" />
    </>
  ),
  "chevron-right": (
    <>
      <path d="M9 6l6 6-6 6" />
    </>
  ),
  chart: (
    <>
      <path d="M3 3v18h18M7 14l3-3 4 3 5-6" />
    </>
  ),
  users: (
    <>
      <circle cx="9" cy="8" r="3.5" />
      <path d="M2.5 21c0-3 3-5.5 6.5-5.5s6.5 2.5 6.5 5.5" />
      <path d="M16 11a3 3 0 1 0 0-6M21.5 21c0-2.4-1.7-4.4-4-5.1" />
    </>
  ),
  audit: (
    <>
      <path d="M5 4h11l4 4v12a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1z" />
      <path d="M9 12l2 2 4-4M16 4v4h4" />
    </>
  ),
};

type Props = SVGProps<SVGSVGElement> & {
  name: IconName;
  size?: number;
};

export function Icon({ name, size = 18, ...rest }: Props) {
  return (
    <svg
      aria-hidden="true"
      fill="none"
      height={size}
      stroke="currentColor"
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={1.7}
      viewBox="0 0 24 24"
      width={size}
      {...rest}
    >
      {paths[name]}
    </svg>
  );
}
