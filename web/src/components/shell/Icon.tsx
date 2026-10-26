// Shell component shared with designer-next (same look and behavior).
// Source: designer-next/web/src/components/shell/Icon.tsx.
import type { SVGProps } from "react";

export type IconName =
  | "app"
  | "save"
  | "open"
  | "newpage"
  | "table"
  | "build"
  | "play"
  | "stop"
  | "refresh"
  | "preview"
  | "undo"
  | "redo"
  | "close"
  | "folder"
  | "page"
  | "database"
  | "column"
  | "key"
  | "link"
  | "doc"
  | "search"
  | "external"
  | "trash"
  | "plus"
  | "warn"
  | "check"
  | "settings"
  | "code"
  | "layout"
  | "image";

const PATHS: Record<IconName, string> = {
  app: "M4 5.5h7l1.5 2H20v11H4z M8 10h8 M8 13h8 M8 16h5",
  save: "M4 4h11l5 5v11H4z M8 4v6h7V4 M8 20v-7h8v7",
  open: "M3 7h6l2 2h10v10H3z M3 7v-2h6l2 2",
  newpage: "M6 3h8l4 4v14H6z M14 3v5h5 M12 11v6 M9 14h6",
  table: "M4 5h16v14H4z M4 10h16 M4 15h16 M9 5v14 M15 5v14",
  build: "M4 6h16 M4 12h10 M4 18h13 M18 11l3 3-3 3 M21 14h-4",
  play: "M7 4l12 8-12 8z",
  stop: "M6 6h12v12H6z",
  refresh: "M19 8a7 7 0 1 0 1 6 M19 4v4h-4",
  preview: "M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6S2 12 2 12z M12 9a3 3 0 1 0 0 6 3 3 0 0 0 0-6z",
  undo: "M9 7H4v5 M4 12a8 8 0 0 1 14-5 M4 12a8 8 0 0 0 14 5",
  redo: "M15 7h5v5 M20 12a8 8 0 0 0-14-5 M20 12a8 8 0 0 1-14 5",
  close: "M6 6l12 12 M18 6L6 18",
  folder: "M3 6h7l2 2h9v11H3z",
  page: "M6 3h8l4 4v14H6z M14 3v5h5 M9 12h6 M9 15h6 M9 18h4",
  database: "M4 6c0-1.7 3.6-3 8-3s8 1.3 8 3-3.6 3-8 3-8-1.3-8-3z M4 6v6c0 1.7 3.6 3 8 3s8-1.3 8-3V6 M4 12v6c0 1.7 3.6 3 8 3s8-1.3 8-3v-6",
  column: "M4 4h16v16H4z M10 4v16 M14 9h3 M14 12h3 M14 15h3",
  key: "M8 15a4 4 0 1 1 3-6.6L20 8v3h-2v3h-3v-2h-2.2A4 4 0 0 1 8 15z M7 11h.01",
  link: "M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1.5 1.5 M14 10a4 4 0 0 0-5.7 0l-3 3A4 4 0 0 0 11 18.7l1.5-1.5",
  doc: "M5 3h9l5 5v13H5z M14 3v5h5 M8 12h8 M8 15h8 M8 18h5",
  search: "M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14z M16 16l4 4",
  external: "M14 4h6v6 M20 4l-9 9 M18 14v6H4V6h6",
  trash: "M4 7h16 M9 7V4h6v3 M6 7l1 14h10l1-14 M10 11v6 M14 11v6",
  plus: "M12 5v14 M5 12h14",
  warn: "M12 3l10 18H2z M12 9v6 M12 18h.01",
  check: "M4 12l5 5L20 6",
  settings: "M12 9a3 3 0 1 0 0 6 3 3 0 0 0 0-6z M19 12a7 7 0 0 0-.1-1l2-1.5-2-3.4-2.3 1a7 7 0 0 0-1.7-1L14.6 3h-4l-.4 2.6a7 7 0 0 0-1.7 1l-2.3-1-2 3.4L6.2 11a7 7 0 0 0 0 2l-2 1.5 2 3.4 2.3-1a7 7 0 0 0 1.7 1l.4 2.6h4l.4-2.6a7 7 0 0 0 1.7-1l2.3 1 2-3.4-2-1.5c.06-.33.1-.66.1-1z",
  code: "M8 7l-5 5 5 5 M16 7l5 5-5 5 M13 4l-2 16",
  layout: "M4 4h16v16H4z M4 9h16 M10 9v11 M14 12h4 M14 15h4",
  image: "M4 5h16v14H4z M4 15l4-4 3 3 3-4 6 6 M8 9h.01",
};

interface IconProps extends Omit<SVGProps<SVGSVGElement>, "name"> {
  name: IconName;
  size?: number;
}

export function Icon({ name, size = 14, ...rest }: IconProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.55"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
      {...rest}
    >
      <path d={PATHS[name]} />
    </svg>
  );
}
