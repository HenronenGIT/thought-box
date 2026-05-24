import { CaptureBottomNav } from "./CaptureBottomNav";
import { CaptureWaveform } from "./CaptureWaveform";

type MobileCaptureReviewProps = {
  category: string | null;
  transcript: string | null;
};

export function MobileCaptureReview({ category, transcript }: MobileCaptureReviewProps) {
  return (
    <section className="mobile-capture-screen mobile-capture-review" aria-label="Review capture">
      <div className="mobile-review-heading">
        <h1>Got it.</h1>
        <p>Keep this thought?</p>
      </div>

      <article className="mobile-review-card">
        <div className="mobile-review-label">What you said</div>
        <CaptureWaveform muted />
        <p className="mobile-review-transcript">
          "{transcript || "Your thought is being shaped..."}"
        </p>
        <div className="mobile-review-divider" />
        <div className="mobile-review-category">+ Feels like · {category || "thought"}</div>
      </article>

      <CaptureBottomNav />
    </section>
  );
}
