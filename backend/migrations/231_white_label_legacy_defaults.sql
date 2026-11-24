-- Replace only the historical built-in branding defaults.
-- Custom site names and subtitles are intentionally left unchanged.
UPDATE settings
SET value = 'API Gateway',
    updated_at = NOW()
WHERE key = 'site_name'
  AND value IN ('Sub2API', 'Sub2api');

UPDATE settings
SET value = 'API Gateway Platform',
    updated_at = NOW()
WHERE key = 'site_subtitle'
  AND value IN ('Subscription to API Conversion Platform', 'AI API Gateway Platform');
