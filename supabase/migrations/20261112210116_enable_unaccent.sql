-- Word search ignores accents: «agüa» matches «agua».
create extension if not exists unaccent with schema extensions;
