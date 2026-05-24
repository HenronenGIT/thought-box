"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { MobileCaptureView } from "./components/capture/MobileCaptureView";
import { useIsMobile } from "./design-system/useIsMobile";

type ApiConfig = {
  max_duration_ms: number;
  min_duration_ms: number;
  max_size_bytes: number;
};

type Thought = {
  id: string;
  status: string;
  created_at: string;
  transcript: string | null;
  audio: { mime_type: string; duration_ms: number | null; size_bytes: number };
  enrichment: null | {
    category: string;
    tags: string[];
    title: string;
    summary: string;
  };
  last_error: string | null;
};

type Echo = {
  id: string;
  thought_id: string;
  mode: string;
  content: string | null;
  status: string;
  is_default: boolean;
};

const categories = ["idea", "observation", "feeling", "learning"];
// All API calls go through Next.js's /api proxy (configured in next.config.ts)
// so web and API are same-origin and the session cookie travels naturally.
const apiBase = "/api";

const echoModeLabels: Record<string, string> = {
  mirror: "A reflection",
  challenger: "A challenge",
  reframer: "Another angle",
  extender: "Where to go next",
};

function pickMimeType() {
  const candidates = ["audio/webm;codecs=opus", "audio/mp4", "audio/webm"];
  return candidates.find((candidate) => MediaRecorder.isTypeSupported(candidate)) ?? "";
}

const echoModes = ["mirror", "challenger", "reframer", "extender"] as const;
const maxEchoesPerThought = 4;

function EchoesSection({ thoughtId, thoughtStatus }: { thoughtId: string; thoughtStatus: string }) {
  const [echoes, setEchoes] = useState<Echo[]>([]);
  const [pollKey, setPollKey] = useState(0);
  const [selecting, setSelecting] = useState(false);

  useEffect(() => {
    if (thoughtStatus !== "done") return;
    let cancelled = false;
    let timer: number | null = null;
    let attempts = 0;
    const maxAttempts = 30;

    const tick = async () => {
      attempts += 1;
      try {
        const res = await fetch(`${apiBase}/thoughts/${thoughtId}/echoes`);
        if (!res.ok) return;
        const body = (await res.json()) as { items: Echo[] };
        if (cancelled) return;
        setEchoes(body.items);
        const hasReady = body.items.some((e) => e.status === "ready");
        const stillWaiting = body.items.some((e) => e.status === "pending" || e.status === "generating");
        if (stillWaiting || (!hasReady && attempts < maxAttempts)) {
          timer = window.setTimeout(tick, 2000);
        }
      } catch {
        // swallow — silent failure
      }
    };
    void tick();
    return () => {
      cancelled = true;
      if (timer) window.clearTimeout(timer);
    };
  }, [thoughtId, thoughtStatus, pollKey]);

  async function requestEcho(mode: string) {
    setSelecting(false);
    try {
      const res = await fetch(`${apiBase}/thoughts/${thoughtId}/echoes`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ mode }),
      });
      if (res.ok) {
        const created = (await res.json()) as Echo;
        setEchoes((current) => [...current, created]);
      }
    } catch {
      // silent failure
    } finally {
      setPollKey((k) => k + 1);
    }
  }

  const ready = echoes.filter((e) => e.status === "ready" && e.content);
  const activeEchoes = echoes.filter((e) => e.status !== "failed");
  const presentModes = new Set(activeEchoes.map((e) => e.mode));
  const remainingModes = echoModes.filter((m) => !presentModes.has(m));
  const atCap = activeEchoes.length >= maxEchoesPerThought;
  const canRequestMore = thoughtStatus === "done" && !atCap && remainingModes.length > 0;

  if (ready.length === 0 && !canRequestMore) return null;

  return (
    <div className="echoes">
      {ready.map((echo) => (
        <div className="echo" key={echo.id}>
          <div className="echo-label">{echoModeLabels[echo.mode] ?? echo.mode}</div>
          <p className="echo-content">{echo.content}</p>
        </div>
      ))}
      {canRequestMore ? (
        <div className="echo-more">
          {selecting ? (
            <div className="echo-mode-picker">
              {remainingModes.map((mode) => (
                <button key={mode} className="chip" onClick={() => requestEcho(mode)}>
                  {echoModeLabels[mode] ?? mode}
                </button>
              ))}
              <button className="chip" onClick={() => setSelecting(false)}>Cancel</button>
            </div>
          ) : (
            <button className="button-secondary" onClick={() => setSelecting(true)}>
              More angles
            </button>
          )}
        </div>
      ) : null}
    </div>
  );
}

export default function Home() {
  const isMobile = useIsMobile();
  const [health, setHealth] = useState("checking");
  const [config, setConfig] = useState<ApiConfig | null>(null);
  const [thoughts, setThoughts] = useState<Thought[]>([]);
  const [recording, setRecording] = useState(false);
  const [elapsedMs, setElapsedMs] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [category, setCategory] = useState<string | null>(null);
  const [tag, setTag] = useState("");
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [me, setMe] = useState<{ email: string; displayName: string } | null>(null);
  const [reviewThoughtId, setReviewThoughtId] = useState<string | null>(null);
  const recorderRef = useRef<MediaRecorder | null>(null);
  const chunksRef = useRef<BlobPart[]>([]);
  const startedRef = useRef<number>(0);
  const timerRef = useRef<number | null>(null);

  const activeTag = tag.trim();
  const healthState = health.startsWith("ok") ? "ok" : health.startsWith("bad") ? "bad" : health;
  const filteredQuery = useMemo(() => {
    const params = new URLSearchParams();
    if (category) params.set("category", category);
    if (activeTag) params.set("tag", activeTag);
    return params;
  }, [category, activeTag]);
  const reviewThought = reviewThoughtId ? thoughts.find((thought) => thought.id === reviewThoughtId) ?? null : null;
  const reviewCategory = reviewThought?.enrichment?.category ?? null;
  const reviewTranscript = reviewThought?.transcript ?? null;

  async function fetchJson<T>(path: string): Promise<T> {
    const response = await fetch(`${apiBase}${path}`);
    if (response.status === 401) {
      window.location.assign("/login");
      throw new Error("unauthorized");
    }
    if (!response.ok) throw new Error(await response.text());
    return response.json();
  }

  async function refreshList(cursor?: string) {
    const params = new URLSearchParams(filteredQuery);
    if (cursor) params.set("before", cursor);
    const qs = params.toString();
    const page = await fetchJson<{ items: Thought[]; next_cursor: string | null }>(
      "/thoughts" + (qs ? "?" + qs : ""),
    );
    setThoughts((current) => (cursor ? [...current, ...page.items] : page.items));
    setNextCursor(page.next_cursor);
  }

  useEffect(() => {
    fetchJson<{ ok: boolean; env: string }>("/health")
      .then((body) => setHealth(`${body.ok ? "ok" : "bad"} / ${body.env}`))
      .catch(() => setHealth("offline"));
    fetchJson<ApiConfig>("/config").then(setConfig).catch((err) => setError(err.message));
    fetchJson<{ user_id: string; email?: string; display_name?: string }>("/me")
      .then((u) => setMe({ email: u.email ?? "", displayName: u.display_name ?? "" }))
      .catch(() => {/* fetchJson already redirected on 401 */});
  }, []);

  useEffect(() => {
    refreshList().catch((err) => setError(err.message));
  }, [filteredQuery]);

  useEffect(() => {
    const pending = thoughts.filter((thought) => ["pending", "transcribing", "enriching"].includes(thought.status));
    if (pending.length === 0) return;
    const interval = window.setInterval(async () => {
      const updates = await Promise.all(pending.map((thought) => fetchJson<Thought>(`/thoughts/${thought.id}`)));
      setThoughts((current) => current.map((thought) => updates.find((item) => item.id === thought.id) ?? thought));
    }, 2000);
    return () => window.clearInterval(interval);
  }, [thoughts]);

  async function startRecording() {
    if (!config) return;
    setError(null);
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    const mimeType = pickMimeType();
    const recorder = new MediaRecorder(stream, mimeType ? { mimeType } : undefined);
    recorderRef.current = recorder;
    chunksRef.current = [];
    startedRef.current = Date.now();
    recorder.ondataavailable = (event) => {
      if (event.data.size > 0) chunksRef.current.push(event.data);
    };
    recorder.onstop = () => {
      stream.getTracks().forEach((track) => track.stop());
      void uploadRecording(mimeType || recorder.mimeType);
    };
    recorder.start();
    setRecording(true);
    timerRef.current = window.setInterval(() => {
      const elapsed = Date.now() - startedRef.current;
      setElapsedMs(elapsed);
      if (elapsed >= config.max_duration_ms) stopRecording();
    }, 100);
  }

  function stopRecording() {
    if (timerRef.current) window.clearInterval(timerRef.current);
    timerRef.current = null;
    recorderRef.current?.stop();
    setRecording(false);
  }

  async function uploadRecording(mimeType: string) {
    if (!config) return;
    const durationMs = Date.now() - startedRef.current;
    const blob = new Blob(chunksRef.current, { type: mimeType });
    if (durationMs < config.min_duration_ms) {
      setError("Recording too short");
      return;
    }
    if (blob.size > config.max_size_bytes) {
      setError("Recording too large");
      return;
    }
    const form = new FormData();
    form.append("mime_type", mimeType);
    form.append("duration_ms", String(durationMs));
    form.append("audio", blob, "thought");
    const response = await fetch(`${apiBase}/thoughts`, { method: "POST", body: form });
    if (response.status === 401) {
      window.location.assign("/login");
      return;
    }
    if (!response.ok) throw new Error(await response.text());
    const created = (await response.json()) as { id: string };
    setReviewThoughtId(created.id);
    await refreshList();
    setElapsedMs(0);
  }

  return (
    <main className="app-shell">
      {isMobile ? (
        <div className="mobile-only">
          <MobileCaptureView
            category={reviewCategory}
            disabled={!config}
            elapsedMs={elapsedMs}
            error={error}
            recording={recording}
            transcript={reviewTranscript}
            onStart={startRecording}
            onStop={stopRecording}
          />
        </div>
      ) : null}

      {!isMobile ? (
      <div className="desktop-only">
        <header className="app-header">
          <div className="brand-stack">
            <h1 className="page-title">Thought Box</h1>
          </div>
          <div className="header-meta">
            {me ? (
              <>
                <div className="user-pill" title={me.email}>
                  {me.displayName || me.email || "Signed in"}
                </div>
                <button
                  type="button"
                  className="button-secondary"
                  onClick={async () => {
                    await fetch(`${apiBase}/auth/logout`, { method: "POST" });
                    window.location.assign("/login");
                  }}
                >
                  Sign out
                </button>
              </>
            ) : null}
            <div className="status-pill" data-state={healthState}>{health}</div>
          </div>
        </header>

        <div className="main-grid">
          <aside className="panel" aria-label="Capture and filters">
            <div className="panel-header">
              <h2 className="section-title">Capture</h2>
              <p className="section-copy">Record a thought and send it into the enrichment queue.</p>
            </div>

            <div className="recorder-control">
              <button className={`record-button ${recording ? "recording" : ""}`} onClick={recording ? stopRecording : startRecording}>
                {recording ? "Stop" : "Rec"}
              </button>
              <div>
                <span className="timer-value">{Math.floor(elapsedMs / 1000)}s</span>
                {config ? <div className="meta">max {Math.floor(config.max_duration_ms / 1000)}s</div> : null}
              </div>
            </div>

            {error ? <div className="error">{error}</div> : null}

            <div className="filter-stack">
              <div className="panel-header">
                <h2 className="section-title">Filters</h2>
                <p className="section-copy">Refine the feed without leaving capture mode.</p>
              </div>
              <div className="chip-row">
                {categories.map((item) => (
                  <button key={item} className={`chip ${category === item ? "active" : ""}`} onClick={() => setCategory(category === item ? null : item)}>
                    {item}
                  </button>
                ))}
              </div>
              <input className="tag-input" value={tag} onChange={(event) => setTag(event.target.value)} placeholder="Filter by tag" />
            </div>
          </aside>

          <section className="thoughts" aria-label="Thought feed">
            {thoughts.map((thought) => (
              <article className="thought-card" key={thought.id}>
                <div className="card-meta-row">
                  <div className="meta">{new Date(thought.created_at).toLocaleString()} / {thought.status}</div>
                  {thought.enrichment ? <div className="category-mark">{thought.enrichment.category}</div> : null}
                </div>
                <h2>{thought.enrichment?.title ?? "Untitled thought"}</h2>
                {thought.enrichment ? <p>{thought.enrichment.summary}</p> : null}
                {thought.transcript ? <p>{thought.transcript}</p> : null}
                <audio controls src={`${apiBase}/thoughts/${thought.id}/audio`} />
                {thought.enrichment ? <div className="tags">{thought.enrichment.tags.join(", ")}</div> : null}
                {thought.last_error ? <div className="error">{thought.last_error}</div> : null}
                <EchoesSection thoughtId={thought.id} thoughtStatus={thought.status} />
              </article>
            ))}
            {nextCursor ? (
              <div className="load-row">
                <button className="button-secondary" onClick={() => refreshList(nextCursor)}>Load more</button>
              </div>
            ) : null}
          </section>
        </div>
      </div>
      ) : null}
    </main>
  );
}
