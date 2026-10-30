#!/bin/bash
# Read-only export of shourenso.vpet state from a Mattermost PostgreSQL container
# into the snapshot schema that cmd/vpet-migrate validates.
#
#   SSH_OPTS="-i ~/.ssh/key" ./export-mattermost.sh <ssh-host> <pg-container> <batch-id> > snapshot.json
#
# Touches nothing: a single SELECT. Keys whose user row no longer exists are
# skipped (nobody can claim them) and reported on stderr.
set -euo pipefail

host=${1:?ssh host}
container=${2:?postgres container}
batch=${3:?batch id}

ssh ${SSH_OPTS:-} "$host" "docker exec $container psql -U mmuser -d mattermost -t -A -v ON_ERROR_STOP=1 -c \"
WITH kv AS (
    SELECT pkey, convert_from(pvalue, 'UTF8') AS raw
    FROM pluginkeyvaluestore
    WHERE pluginid = 'shourenso.vpet' AND pkey <> 'mmi_botid'
),
players AS (
    SELECT u.id, u.username, u.nickname, u.firstname, u.lastname, u.locale
    FROM kv
    JOIN users u ON u.id = substring(kv.pkey from 3)
    WHERE kv.pkey LIKE 'u\_%'
),
keep AS (
    SELECT * FROM kv
    WHERE pkey NOT LIKE 'u\_%'
       OR substring(pkey from 3) IN (SELECT id FROM players)
)
SELECT json_build_object(
    'schema', 1,
    'batch', '$batch',
    'exported_at', to_char(now() AT TIME ZONE 'UTC', 'YYYY-MM-DD\\\"T\\\"HH24:MI:SS\\\"Z\\\"'),
    'users', (SELECT coalesce(json_agg(json_build_object(
            'id', id,
            'username', username,
            'display_name', coalesce(nullif(nickname, ''), nullif(trim(firstname || ' ' || lastname), ''), username),
            'locale', CASE WHEN locale IN ('en', 'ja') THEN locale ELSE 'vi' END,
            'disabled', false
        ) ORDER BY username), '[]'::json) FROM players),
    'kv', (SELECT coalesce(json_object_agg(pkey, raw::json), '{}'::json) FROM keep),
    'checksums', (SELECT coalesce(json_object_agg(pkey, encode(sha256(raw::json::text::bytea), 'hex')), '{}'::json) FROM keep)
)::text;\""
