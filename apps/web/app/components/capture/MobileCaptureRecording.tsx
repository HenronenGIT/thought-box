import { CaptureParticles } from "./CaptureParticles";
import { CaptureRecordButton } from "./CaptureRecordButton";
import { CaptureWaveform } from "./CaptureWaveform";

type MobileCaptureRecordingProps = {
  elapsedSeconds: number;
  onStop: () => void;
};

export function MobileCaptureRecording({ elapsedSeconds, onStop }: MobileCaptureRecordingProps) {
  return (
    <section className="mobile-capture-screen mobile-capture-recording" aria-label="Recording">
      <CaptureParticles />

      <div className="mobile-capture-listening">
        <CaptureRecordButton active label={`Stop recording, ${elapsedSeconds} seconds elapsed`} onClick={onStop} />
        <CaptureWaveform />
        <p>Take as long, or as little, as you need.</p>
      </div>
    </section>
  );
}
