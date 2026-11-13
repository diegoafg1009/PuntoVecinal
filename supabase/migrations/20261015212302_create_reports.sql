-- Reports published by neighbors and their photos. Only the API (its own
-- database role) reads and writes these tables, so RLS is enabled with no
-- policies: anon and authenticated get nothing through Supabase's own API.
create extension if not exists postgis with schema extensions;

create type public.report_status as enum (
  'publicado',
  'derivado',
  'recibido',
  'en_atencion',
  'atendido',
  'cerrado'
);

create table public.reports (
  id uuid primary key default gen_random_uuid(),
  -- Session that published the report (anonymous or registered). Never shown.
  author_id uuid not null references auth.users (id),
  category_slug text not null references public.categories (slug),
  title text not null check (length(title) between 1 and 100),
  details text check (length(details) <= 1000),
  reference text check (length(reference) <= 150),
  location extensions.geography(Point, 4326) not null,
  -- Public author name; null means "Vecino anónimo".
  author_name text check (length(author_name) between 1 and 40),
  status public.report_status not null default 'publicado',
  created_at timestamptz not null default now()
);

create table public.report_photos (
  report_id uuid not null references public.reports (id) on delete cascade,
  position smallint not null check (position between 1 and 3),
  storage_key text not null,
  primary key (report_id, position)
);

-- Publication rate limit and "my reports".
create index reports_author_id_created_at_idx
  on public.reports (author_id, created_at desc);

-- Radius queries (new-report alerts).
create index reports_location_idx
  on public.reports using gist (location);

alter table public.reports enable row level security;
alter table public.report_photos enable row level security;
