import { useState, useEffect, useRef } from "react";
import { useGame } from "@/context/GameContext";

const TOOLTIPS: string[] = [
  "Tap pulse to stay alive.",
  "Miss pulse → eliminated.",
  "Top 3 get paid.",
];
const TOOLTIP_DURATION_MS = 4500;

export function TutorialTooltips() {
  const { state } = useGame();
  const room = state.currentRoom;
  const [index, setIndex] = useState(0);
  const [hidden, setHidden] = useState(false);
  const survivalStartRef = useRef<number | null>(null);

  const isTutorial = room?.tier === 0;
  const isSurvival = room?.state === "survival";
  const show = isTutorial && isSurvival && !hidden && index < TOOLTIPS.length;

  useEffect(() => {
    if (!isTutorial || !isSurvival) {
      survivalStartRef.current = null;
      setIndex(0);
      setHidden(false);
      return;
    }
    if (survivalStartRef.current === null)
      survivalStartRef.current = Date.now();

    const t = window.setTimeout(() => {
      if (index + 1 >= TOOLTIPS.length) {
        setHidden(true);
      } else {
        setIndex((i) => i + 1);
      }
    }, TOOLTIP_DURATION_MS);
    return () => clearTimeout(t);
  }, [isTutorial, isSurvival, index]);

  if (!show) return null;

  return (
    <div className="fixed top-24 left-1/2 -translate-x-1/2 z-55 pointer-events-none animate-in fade-in duration-300">
      <div className="rounded-lg bg-primary/95 text-primary-foreground px-4 py-2.5 text-sm font-medium shadow-lg border border-primary/50 max-w-[90vw]">
        {TOOLTIPS[index]}
      </div>
    </div>
  );
}
