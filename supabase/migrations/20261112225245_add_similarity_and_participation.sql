-- Similar reports and neighbor participation: a text embedding per report
-- (pgvector), the "También me afecta" support and the aportes (contributions).
-- Only the API (its own database role) reads and writes these tables, so RLS is
-- enabled with no policies, as for reports.
create extension if not exists vector with schema extensions;

-- Meaning of title + details (Voyage, 1024 dims). Null until embedded.
alter table public.reports add column embedding extensions.vector(1024);

-- Reports published while the embedding service was unavailable, for the
-- backfill job.
create index reports_embedding_missing_idx
  on public.reports (created_at)
  where embedding is null;

-- "También me afecta". The author is never stored here: the affected count is
-- the author plus the rows of this table.
create table public.report_supports (
  report_id uuid not null references public.reports (id) on delete cascade,
  user_id uuid not null references auth.users (id),
  created_at timestamptz not null default now(),
  primary key (report_id, user_id)
);

-- Aportes: a text with an optional photo added to a report.
create table public.report_contributions (
  id uuid primary key default gen_random_uuid(),
  report_id uuid not null references public.reports (id) on delete cascade,
  -- Session that added the aporte. Never shown.
  author_id uuid not null references auth.users (id),
  -- Public author name; null means "Vecino anónimo".
  author_name text check (length(author_name) between 1 and 40),
  text text not null check (length(text) between 1 and 1200),
  photo_key text,
  created_at timestamptz not null default now()
);

-- Listing the aportes of a report, oldest first.
create index report_contributions_report_idx
  on public.report_contributions (report_id, created_at, id);

-- Aporte rate limit.
create index report_contributions_author_idx
  on public.report_contributions (author_id, created_at desc);

alter table public.report_supports enable row level security;
alter table public.report_contributions enable row level security;
