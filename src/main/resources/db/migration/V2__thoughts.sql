create table thoughts (
    id uuid primary key,
    user_id uuid not null references users(id),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    audio_s3_key text not null,
    mime_type text not null,
    duration_ms integer,
    size_bytes bigint not null,
    transcript text,
    status text not null default 'pending',
    attempts integer not null default 0,
    last_error text,
    last_attempt_at timestamptz,
    transcribed_at timestamptz,
    constraint thoughts_status_check check (
        status in ('pending', 'transcribing', 'enriching', 'done', 'failed_transcription', 'failed_enrichment')
    )
);

create index thoughts_user_created_idx on thoughts(user_id, created_at desc);
create index thoughts_status_idx on thoughts(status, updated_at);

