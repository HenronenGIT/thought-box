create table thought_enrichments (
    thought_id uuid primary key references thoughts(id) on delete cascade,
    category text not null,
    tags text[] not null default '{}',
    title text not null,
    summary text not null,
    model text not null,
    prompt_version text not null,
    created_at timestamptz not null default now(),
    constraint thought_enrichments_category_check check (
        category in ('idea', 'todo', 'feeling', 'question', 'observation', 'reminder')
    )
);

create index thought_enrichments_category_idx on thought_enrichments(category);
create index thought_enrichments_tags_idx on thought_enrichments using gin(tags);

