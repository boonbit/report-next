// Shell component shared with designer-next (same look and behavior).
// Source: designer-next/web/src/components/shell/Splitter.tsx.
import { useRef, useState, type PointerEvent as ReactPointerEvent } from "react";

interface SplitterProps {
  orientation?: "vertical" | "horizontal";
  onDelta: (delta: number) => void;
  label: string;
}

/** A classic drag bar. Deltas are reported per pointer move. */
export function Splitter({ orientation = "vertical", onDelta, label }: SplitterProps) {
  const last = useRef<number | null>(null);
  const [dragging, setDragging] = useState(false);

  const start = (event: ReactPointerEvent<HTMLDivElement>) => {
    event.preventDefault();
    last.current = orientation === "vertical" ? event.clientX : event.clientY;
    setDragging(true);
    const move = (moveEvent: PointerEvent) => {
      if (last.current === null) return;
      const position = orientation === "vertical" ? moveEvent.clientX : moveEvent.clientY;
      const delta = position - last.current;
      if (delta !== 0) {
        last.current = position;
        onDelta(delta);
      }
    };
    const up = () => {
      last.current = null;
      setDragging(false);
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
      document.body.style.cursor = "";
    };
    document.body.style.cursor = orientation === "vertical" ? "col-resize" : "row-resize";
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  };

  return (
    <div
      className={`splitter${orientation === "horizontal" ? " horizontal" : ""}${dragging ? " dragging" : ""}`}
      role="separator"
      aria-label={label}
      aria-orientation={orientation === "vertical" ? "vertical" : "horizontal"}
      onPointerDown={start}
    />
  );
}
