export interface CursorPosition {
  type: "cursor";
  line: number;
  col: number;
}

export interface RangePosition {
  type: "range";
  start_line: number;
  start_col: number;
  end_line: number;
  end_col: number;
}

export type Position = CursorPosition | RangePosition;

export interface Comment {
  id: number;
  project_root: string;
  timestamp: string;
  file: string;
  position: Position;
  text: string;
  archived: boolean;
}
