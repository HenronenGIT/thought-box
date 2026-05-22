-- +goose Up
create table if not exists thought_enrichments (
    thought_id uuid primary key references thoughts(id) on delete cascade,
    category text not null,
    tags text[] not null default '{}',
    title text not null,
    summary text not null,
    model text not null,
    prompt_version text not null,
    created_at timestamptz not null default now()
);

alter table thought_enrichments drop constraint if exists thought_enrichments_category_check;
alter table thought_enrichments add constraint thought_enrichments_category_check check (
    category in ('idea', 'observation', 'feeling', 'learning')
);

create index if not exists thought_enrichments_category_idx on thought_enrichments(category);
create index if not exists thought_enrichments_tags_idx on thought_enrichments using gin(tags);

-- +goose Down
drop table if exists thought_enrichments;
