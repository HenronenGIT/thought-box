import { MicrophoneIcon } from "./MicrophoneIcon";

export function CaptureBottomNav() {
  return (
    <nav className="capture-bottom-nav" aria-label="Primary">
      <button type="button" aria-label="Thoughts">
        <span className="nav-lines" aria-hidden="true" />
        <span>Thoughts</span>
      </button>
      <button type="button" className="active" aria-label="Capture">
        <MicrophoneIcon className="nav-mic-icon" />
        <span>Capture</span>
      </button>
      <button type="button" aria-label="Echoes">
        <span className="nav-echo" aria-hidden="true" />
        <span>Echoes</span>
      </button>
    </nav>
  );
}
