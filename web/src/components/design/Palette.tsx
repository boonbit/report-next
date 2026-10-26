import type { NewObjectKind } from "../../ir/mutate";
import { Icon } from "../shell/Icon";

// U1 object palette: click-then-place entry points for new objects.
// Placement itself is owned by the parent via onAdd; this component
// only calls the callback and never mutates the report.
export interface PaletteProps {
  onAdd: (kind: NewObjectKind) => void;
  disabled?: boolean;
}

export function Palette({ onAdd, disabled = false }: PaletteProps) {
  return (
    <div className="palette-rail" role="toolbar" aria-label="Object palette">
      <button
        type="button"
        className="btn small palette-btn"
        title="Add text object"
        aria-label="Add text object"
        disabled={disabled}
        onClick={() => onAdd("TextObject")}
      >
        <Icon name="doc" />
        <span>Text</span>
      </button>
      <button
        type="button"
        className="btn small palette-btn"
        title="Add barcode object"
        aria-label="Add barcode object"
        disabled={disabled}
        onClick={() => onAdd("BarcodeObject")}
      >
        <Icon name="plus" />
        <span>Barcode</span>
      </button>
    </div>
  );
}
