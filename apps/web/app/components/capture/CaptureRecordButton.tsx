import { MicrophoneIcon } from "./MicrophoneIcon";

type CaptureRecordButtonProps = {
  active?: boolean;
  disabled?: boolean;
  label: string;
  onClick: () => void;
};

export function CaptureRecordButton({ active = false, disabled = false, label, onClick }: CaptureRecordButtonProps) {
  return (
    <button
      type="button"
      className={`mobile-record-orb ${active ? "active" : ""}`}
      disabled={disabled}
      aria-label={label}
      onClick={onClick}
    >
      <MicrophoneIcon className="mobile-mic-icon" />
    </button>
  );
}
