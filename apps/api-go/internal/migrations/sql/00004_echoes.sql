-- +goose Up
create table if not exists echoes (
    id uuid primary key,
    thought_id uuid not null references thoughts(id) on delete cascade,
    mode text not null,
    content text,
    status text not null,
    is_default boolean not null default false,
    attempts int not null default 0,
    last_error text,
    last_attempt_at timestamptz,
    model text,
    prompt_version text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

alter table echoes drop constraint if exists echoes_mode_check;
alter table echoes add constraint echoes_mode_check check (
    mode in ('mirror', 'challenger', 'reframer', 'extender')
);

alter table echoes drop constraint if exists echoes_status_check;
alter table echoes add constraint echoes_status_check check (
    status in ('pending', 'generating', 'ready', 'failed')
);

create unique index if not exists echoes_thought_mode_active_idx
    on echoes (thought_id, mode)
    where status <> 'failed';

create index if not exists echoes_thought_idx on echoes (thought_id);
create index if not exists echoes_status_idx on echoes (status);

-- +goose Down
drop table if exists echoes;
