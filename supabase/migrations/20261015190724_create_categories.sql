-- Fixed report categories. Reference data every environment needs, so it is
-- seeded here and not in seed.sql. The app can only read it.
create table public.categories (
  slug text primary key,
  label text not null,
  position smallint not null unique
);

alter table public.categories enable row level security;

create policy "categories are readable by everyone"
  on public.categories
  for select
  to anon, authenticated
  using (true);

insert into public.categories (slug, label, position) values
  ('limpieza', 'Limpieza', 1),
  ('alumbrado', 'Alumbrado', 2),
  ('pistas-y-veredas', 'Pistas y veredas', 3),
  ('agua-y-desague', 'Agua y desagüe', 4),
  ('otro', 'Otro', 5);
