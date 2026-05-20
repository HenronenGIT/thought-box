create table users (
    id uuid primary key,
    created_at timestamptz not null default now()
);

insert into users (id)
values ('00000000-0000-4000-8000-000000000001');

