import { MobileCaptureIdle } from "./MobileCaptureIdle";
import { MobileCaptureRecording } from "./MobileCaptureRecording";
import { MobileCaptureReview } from "./MobileCaptureReview";

type MobileCaptureViewProps = {
  category: string | null;
  disabled?: boolean;
  elapsedMs: number;
  error: string | null;
  recording: boolean;
  transcript: string | null;
  onStart: () => void;
  onStop: () => void;
};

export function MobileCaptureView({
  category,
  disabled = false,
  elapsedMs,
  error,
  recording,
  transcript,
  onStart,
  onStop,
}: MobileCaptureViewProps) {
  if (recording) {
    return <MobileCaptureRecording elapsedSeconds={Math.floor(elapsedMs / 1000)} onStop={onStop} />;
  }

  if (transcript) {
    return <MobileCaptureReview category={category} transcript={transcript} />;
  }

  return <MobileCaptureIdle disabled={disabled} error={error} onStart={onStart} />;
}
