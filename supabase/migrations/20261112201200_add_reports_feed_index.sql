-- Keyset pagination of the report feed: newest first, ties broken by id.
create index reports_created_at_id_idx
  on public.reports (created_at desc, id desc);
