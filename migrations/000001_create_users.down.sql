BEGIN;

DROP TABLE IF EXISTS auth_keys_permissions;
DROP TABLE IF EXISTS keys_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS keys;
DROP TABLE IF EXISTS tokens;
DROP TABLE IF EXISTS users;

COMMIT;