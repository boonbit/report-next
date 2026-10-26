import type { ReportTab } from "../../App";

/** Commands owned by App (they touch fetch or open dialogs). */
export interface ReportActions {
  loadSample: () => void;
  importFile: (f: File) => void;
  renderPreview: () => void;
  exportDoc: (kind: "pdf" | "docx") => void;
  downloadJson: () => void;
  showTab: (tab: ReportTab) => void;
  about: () => void;
  undo: () => void;
  redo: () => void;
  newReport: () => void;
  toggleConsole: () => void;
}
