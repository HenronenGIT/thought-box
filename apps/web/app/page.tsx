"use client";

import { useEffect, useMemo, useRef, useState } from "react";

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

const categories = ["idea", "observation", "feeling", "learning"];
const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

function pickMimeType() {
  const candidates = ["audio/webm;codecs=opus", "audio/mp4", "audio/webm"];
  return candidates.find((candidate) => MediaRecorder.isTypeSupported(candidate)) ?? "";
}

export default function Home() {
  const [health, setHealth] = useState("checking");
  const [config, setConfig] = useState<ApiConfig | null>(null);
  const [thoughts, setThoughts] = useState<Thought[]>([]);
  const [recording, setRecording] = useState(false);
  const [elapsedMs, setElapsedMs] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [category, setCategory] = useState<string | null>(null);
  const [tag, setTag] = useState("");
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const recorderRef = useRef<MediaRecorder | null>(null);
  const chunksRef = useRef<BlobPart[]>([]);
  const startedRef = useRef<number>(0);
  const timerRef = useRef<number | null>(null);

  const activeTag = tag.trim();
  const filteredUrl = useMemo(() => {
    const url = new URL("/thoughts", apiBase);
    if (category) url.searchParams.set("category", category);
    if (activeTag) url.searchParams.set("tag", activeTag);
    return url;
  }, [category, activeTag]);

  async function fetchJson<T>(path: string | URL): Promise<T> {
    const response = await fetch(path instanceof URL ? path : `${apiBase}${path}`);
    if (!response.ok) throw new Error(await response.text());
    return response.json();
  }

  async function refreshList(cursor?: string) {
    const url = new URL(filteredUrl);
    if (cursor) url.searchParams.set("before", cursor);
    const page = await fetchJson<{ items: Thought[]; next_cursor: string | null }>(url);
    setThoughts((current) => (cursor ? [...current, ...page.items] : page.items));
    setNextCursor(page.next_cursor);
  }

  useEffect(() => {
    fetchJson<{ ok: boolean; env: string }>("/health")
      .then((body) => setHealth(`${body.ok ? "ok" : "bad"} / ${body.env}`))
      .catch(() => setHealth("offline"));
    fetchJson<ApiConfig>("/config").then(setConfig).catch((err) => setError(err.message));
  }, []);

  useEffect(() => {
    refreshList().catch((err) => setError(err.message));
  }, [filteredUrl]);

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
    if (!response.ok) throw new Error(await response.text());
    await refreshList();
    setElapsedMs(0);
  }

  return (
    <main className="shell">
      <div className="topbar">
        <h1>Thought Box</h1>
        <div className="health">{health}</div>
      </div>

      <section className="recorder">
        <div className="controls">
          <button className={`record-button ${recording ? "recording" : ""}`} onClick={recording ? stopRecording : startRecording}>
            {recording ? "Stop" : "Rec"}
          </button>
          <div>
            <strong>{Math.floor(elapsedMs / 1000)}s</strong>
            {config ? <div className="meta">max {Math.floor(config.max_duration_ms / 1000)}s</div> : null}
          </div>
        </div>
        {error ? <div className="error">{error}</div> : null}
      </section>

      <div className="chips">
        {categories.map((item) => (
          <button key={item} className={`chip ${category === item ? "active" : ""}`} onClick={() => setCategory(category === item ? null : item)}>
            {item}
          </button>
        ))}
        <input className="tag-input" value={tag} onChange={(event) => setTag(event.target.value)} placeholder="tag" />
      </div>

      <section className="thoughts">
        {thoughts.map((thought) => (
          <article className="card" key={thought.id}>
            <div className="meta">{new Date(thought.created_at).toLocaleString()} / {thought.status}</div>
            <h2>{thought.enrichment?.title ?? "Untitled thought"}</h2>
            {thought.enrichment ? <p>{thought.enrichment.summary}</p> : null}
            {thought.transcript ? <p>{thought.transcript}</p> : null}
            <audio controls src={`${apiBase}/thoughts/${thought.id}/audio`} />
            {thought.enrichment ? <div className="tags">{thought.enrichment.category} / {thought.enrichment.tags.join(", ")}</div> : null}
            {thought.last_error ? <div className="error">{thought.last_error}</div> : null}
          </article>
        ))}
      </section>

      {nextCursor ? <button className="chip" onClick={() => refreshList(nextCursor)}>Load more</button> : null}
    </main>
  );
}
