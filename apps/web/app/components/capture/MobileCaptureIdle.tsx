import { CaptureBottomNav } from "./CaptureBottomNav";
import { CaptureRecordButton } from "./CaptureRecordButton";

type MobileCaptureIdleProps = {
  disabled?: boolean;
  error: string | null;
  onStart: () => void;
};

export function MobileCaptureIdle({ disabled = false, error, onStart }: MobileCaptureIdleProps) {
  return (
    <section className="mobile-capture-screen mobile-capture-idle" aria-label="Capture">
      <div className="mobile-capture-heading">
        <p>A quiet surface</p>
        <h1>
          What's on your <em>mind?</em>
        </h1>
      </div>

      <div className="mobile-capture-center">
        <CaptureRecordButton disabled={disabled} label="Start recording" onClick={onStart} />
        {error ? <div className="mobile-capture-error">{error}</div> : null}
      </div>

      <CaptureBottomNav />
    </section>
  );
}
