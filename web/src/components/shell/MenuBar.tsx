import { useEffect, useRef, useState } from "react";
import type { ReportActions } from "./actions";
import type { ReportTab } from "../../App";
import { setLang, tx, useLang } from "../../i18n";

interface MenuItem {
  label: string;
  hint?: string;
  run?: () => void;
  disabled?: boolean;
  checked?: boolean;
  separator?: boolean;
  title?: string;
  ariaLabel?: string;
}

interface MenuSpec {
  label: string;
  items: MenuItem[];
}

const TABS: { id: ReportTab; key: string }[] = [
  { id: "start", key: "menu.tabStart" },
  { id: "design", key: "menu.tabDesign" },
  { id: "preview", key: "menu.tabPreview" },
  { id: "source", key: "menu.tabSource" },
];

export function MenuBar({
  actions,
  tab,
  hasReport,
  canUndo,
  canRedo,
  consoleOpen,
}: {
  actions: ReportActions;
  tab: ReportTab;
  hasReport: boolean;
  canUndo: boolean;
  canRedo: boolean;
  consoleOpen: boolean;
}) {
  const [open, setOpen] = useState<string | null>(null);
  const rootRef = useRef<HTMLDivElement>(null);
  const lang = useLang();
  const t = (key: string): string => tx(lang, key);

  useEffect(() => {
    if (!open) return;
    const onPointer = (event: MouseEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(null);
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(null);
    };
    window.addEventListener("mousedown", onPointer);
    window.addEventListener("keydown", onKey);
    return () => {
      window.removeEventListener("mousedown", onPointer);
      window.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const menus: MenuSpec[] = [
    {
      label: t("menu.file"),
      items: [
        { label: t("menu.newReport"), run: actions.newReport },
        { label: t("menu.loadSample"), run: actions.loadSample },
        { label: t("menu.importFrx"), run: () => document.getElementById("rz-import")?.click() },
        { separator: true, label: "" },
        {
          label: t("menu.downloadJson"),
          hint: "Ctrl+S",
          run: actions.downloadJson,
          disabled: !hasReport,
        },
        { separator: true, label: "" },
        { label: t("menu.exportPdf"), run: () => actions.exportDoc("pdf"), disabled: !hasReport },
        { label: t("menu.exportDocx"), run: () => actions.exportDoc("docx"), disabled: !hasReport },
        { separator: true, label: "" },
        { label: t("menu.about"), run: actions.about },
      ],
    },
    {
      label: t("menu.edit"),
      items: [
        { label: t("menu.undo"), hint: "Ctrl+Z", run: actions.undo, disabled: !canUndo },
        { label: t("menu.redo"), hint: "Ctrl+Y", run: actions.redo, disabled: !canRedo },
      ],
    },
    {
      label: t("menu.view"),
      items: [
        ...TABS.map((tb) => ({
          label: t(tb.key),
          run: () => actions.showTab(tb.id),
          disabled: tb.id !== "start" && !hasReport,
          checked: tab === tb.id,
        })),
        { separator: true, label: "" },
        { label: t("menu.outputConsole"), run: actions.toggleConsole, checked: consoleOpen },
      ],
    },
    {
      label: t("menu.help"),
      items: [
        { label: t("menu.about"), run: actions.about },
        { label: t("menu.shortcuts"), hint: "F1", run: actions.about },
        {
          label: lang === "th" ? t("menu.useEnglish") : t("menu.useThai"),
          run: () => setLang(lang === "th" ? "en" : "th"),
          title: t("menu.langTitle"),
          ariaLabel: lang === "th" ? t("menu.useEnglish") : t("menu.useThai"),
        },
      ],
    },
  ];

  return (
    <div className="menubar" ref={rootRef}>
      {menus.map((menu) => (
        <div key={menu.label} className={open === menu.label ? "menu-root open" : "menu-root"}>
          <button
            type="button"
            className="menu-label"
            onMouseDown={(event) => {
              event.preventDefault();
              setOpen(open === menu.label ? null : menu.label);
            }}
            onMouseEnter={() => {
              if (open) setOpen(menu.label);
            }}
          >
            {menu.label}
          </button>
          {open === menu.label ? (
            <div className="menu-drop" role="menu">
              {menu.items.map((item, index) =>
                item.separator ? (
                  <div key={`sep-${index}`} className="menu-sep" />
                ) : (
                  <button
                    key={item.label}
                    type="button"
                    role="menuitem"
                    className="menu-item"
                    disabled={item.disabled}
                    title={item.title}
                    aria-label={item.ariaLabel}
                    onClick={() => {
                      setOpen(null);
                      item.run?.();
                    }}
                  >
                    <span style={{ width: 14, textAlign: "center" }}>{item.checked ? "✓" : ""}</span>
                    <span>{item.label}</span>
                    {item.hint ? <span className="menu-key">{item.hint}</span> : null}
                  </button>
                ),
              )}
            </div>
          ) : null}
        </div>
      ))}
    </div>
  );
}
