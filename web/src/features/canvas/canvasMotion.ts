import type { TargetAndTransition, Transition, Variants } from "framer-motion";
import type { AgentCanvasState } from "@/features/canvas/canvasModel";

export const CANVAS_MOTION_DURATION_MS = 300;
export const CANVAS_STREAMING_SILENCE_MS = 300;
export const CANVAS_MOTION_SECONDS = CANVAS_MOTION_DURATION_MS / 1000;
export const CANVAS_PANEL_OFFSET_X = 380;
export const CANVAS_MOTION_EASE = [0.16, 1, 0.3, 1] as const;

export function getCanvasTransition(reducedMotion: boolean): Transition {
  return reducedMotion
    ? { duration: 0.01 }
    : { duration: CANVAS_MOTION_SECONDS, ease: CANVAS_MOTION_EASE };
}

export const canvasNodeEnterVariants: Variants = {
  hidden: {
    opacity: 0,
    scale: 0.96,
  },
  visible: {
    opacity: 1,
    scale: 1,
  },
};

export const canvasPanelPresenceVariants: Variants = {
  hidden: {
    opacity: 0,
    x: CANVAS_PANEL_OFFSET_X,
  },
  visible: {
    opacity: 1,
    x: 0,
  },
  exit: {
    opacity: 0,
    x: CANVAS_PANEL_OFFSET_X,
  },
};

export function getNodeMotionTarget(options: {
  reducedMotion: boolean;
  selected: boolean;
  state: AgentCanvasState;
  streamingActive: boolean;
}): TargetAndTransition {
  const { reducedMotion, selected, state, streamingActive } = options;

  if (reducedMotion) {
    return {
      opacity: 1,
      scale: 1,
      transition: getCanvasTransition(true),
    };
  }

  let scale = selected ? 1.01 : 1;

  if (
    state === "thinking" ||
    state === "assigned" ||
    state === "tool_running" ||
    state === "approval_required"
  ) {
    scale = selected ? 1.015 : 1.01;
  }

  if (streamingActive) {
    scale = selected ? 1.018 : 1.012;
  }

  if (state === "completed") {
    scale = selected ? 1.008 : 1.002;
  }

  return {
    opacity: 1,
    scale,
    transition: getCanvasTransition(false),
  };
}