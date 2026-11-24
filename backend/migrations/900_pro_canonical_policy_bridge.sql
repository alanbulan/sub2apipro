-- Preserve the permission of legacy single-group keys when returning to the
-- canonical single-group model. Never guess which group a multi-group key uses.
DO $pro$
BEGIN
    IF to_regclass('public.api_key_groups') IS NOT NULL THEN
        IF EXISTS (
            SELECT 1 FROM api_key_groups edge
            JOIN api_keys key ON key.id = edge.api_key_id
            WHERE key.deleted_at IS NULL
            GROUP BY edge.api_key_id HAVING COUNT(DISTINCT edge.group_id) > 1
        ) THEN
            RAISE EXCEPTION 'Pro canonical migration requires review: active multi-group API keys';
        END IF;
        IF EXISTS (
            SELECT 1 FROM api_key_groups edge
            JOIN api_keys key ON key.id = edge.api_key_id
            WHERE key.deleted_at IS NULL AND key.group_id IS NOT NULL
              AND key.group_id <> edge.group_id
        ) THEN
            RAISE EXCEPTION 'Pro canonical migration requires review: conflicting API key group bindings';
        END IF;
        UPDATE api_keys key SET group_id = edge.group_id
        FROM api_key_groups edge
        WHERE key.id = edge.api_key_id AND key.group_id IS NULL;
    END IF;
    IF to_regclass('public.auto_credit_config') IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM auto_credit_config WHERE enabled) THEN
            RAISE EXCEPTION 'Pro canonical migration requires review: enabled retired automatic-credit policies';
        END IF;
    END IF;
END;
$pro$;
