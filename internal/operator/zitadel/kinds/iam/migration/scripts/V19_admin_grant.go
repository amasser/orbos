package scripts

const V19AdminGrant = `BEGIN;

GRANT SELECT, INSERT, UPDATE, DELETE ON DATABASE admin_api TO admin_api;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE admin_api.* TO admin_api;

COMMIT;
`
