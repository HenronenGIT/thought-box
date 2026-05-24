type CaptureWaveformProps = {
  muted?: boolean;
};

const bars = [34, 50, 26, 44, 56, 36, 62, 40, 58, 68, 42, 52, 66, 46, 34, 60, 72, 38, 54, 64, 44, 74, 58, 36, 48, 62, 70, 50, 42, 64, 56, 36, 52, 68, 44, 58, 34, 62];

export function CaptureWaveform({ muted = false }: CaptureWaveformProps) {
  return (
    <div className={`capture-waveform ${muted ? "muted" : ""}`} aria-hidden="true">
      {bars.map((height, index) => (
        <span key={`${height}-${index}`} style={{ height: `${height}%` }} />
      ))}
    </div>
  );
}
