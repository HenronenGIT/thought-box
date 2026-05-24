-- +goose Up
alter table users
    add column if not exists email text,
    add column if not exists google_sub text,
    add column if not exists display_name text;

create unique index if not exists users_email_idx on users (lower(email)) where email is not null;
create unique index if not exists users_google_sub_idx on users (google_sub) where google_sub is not null;

-- Backfill the seeded user with the admin email so existing thoughts stay
-- attached after first real sign-in. The seeded UUID is preserved.
update users
set email = 'henrimaronen@gmail.com'
where id = '00000000-0000-4000-8000-000000000001'
  and email is null;

create table if not exists sessions (
    token_hash bytea primary key,
    user_id uuid not null references users(id) on delete cascade,
    created_at timestamptz not null default now(),
    expires_at timestamptz not null,
    last_seen_at timestamptz not null default now()
);
create index if not exists sessions_user_idx on sessions (user_id);
create index if not exists sessions_expires_idx on sessions (expires_at);

create table if not exists allowed_emails (
    email text primary key,
    added_at timestamptz not null default now()
);

insert into allowed_emails (email)
values ('henrimaronen@gmail.com')
on conflict (email) do nothing;

-- +goose Down
drop table if exists allowed_emails;
drop table if exists sessions;
drop index if exists users_google_sub_idx;
drop index if exists users_email_idx;
alter table users drop column if exists display_name;
alter table users drop column if exists google_sub;
alter table users drop column if exists email;
