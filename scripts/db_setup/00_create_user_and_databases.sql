-- Consolidated Initialization Script for DanteGPU Platform PostgreSQL Databases
--
-- Runs once via the postgres docker-entrypoint (/docker-entrypoint-initdb.d) on a
-- fresh data volume, as the POSTGRES_USER superuser against POSTGRES_DB.
--
-- IMPORTANT: CREATE DATABASE cannot run inside a transaction block or a PL/pgSQL
-- function. The previous version wrapped it in a function and errored the moment
-- it tried to create the first missing database (dante_billing), leaving the
-- billing/registry/scheduler/storage databases uncreated and their services
-- unable to connect ("failed to connect user=dante_user"). We instead use the
-- psql \gexec pattern, which executes each generated statement outside any
-- transaction and is idempotent via the NOT EXISTS guard.

-- 1. Ensure the primary login role exists with the expected password.
DO
$do$
BEGIN
   IF NOT EXISTS (
      SELECT FROM pg_catalog.pg_roles
      WHERE  rolname = 'dante_user') THEN
      CREATE ROLE dante_user LOGIN PASSWORD 'dante_password';
   ELSE
      ALTER ROLE dante_user WITH LOGIN PASSWORD 'dante_password';
   END IF;
END
$do$;

-- 2. Create each service database (owned by dante_user) only if it is absent.
--    \gexec runs the produced CREATE DATABASE statements one per row, outside a
--    transaction, so this succeeds where the old function-based approach failed.
SELECT format('CREATE DATABASE %I OWNER %I', d, 'dante_user')
FROM unnest(ARRAY[
   'dante_auth',
   'dante_billing',
   'dante_registry',
   'dante_scheduler',
   'dante_storage'
]) AS d
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = d)
\gexec

-- 3. Ensure ownership is correct for any pre-existing database (e.g. dante_auth,
--    created by POSTGRES_DB). ALTER DATABASE is transaction-safe, unlike CREATE.
DO
$do$
DECLARE
   d text;
BEGIN
   FOREACH d IN ARRAY ARRAY['dante_auth','dante_billing','dante_registry','dante_scheduler','dante_storage'] LOOP
      IF EXISTS (SELECT 1 FROM pg_database WHERE datname = d) THEN
         EXECUTE format('ALTER DATABASE %I OWNER TO %I', d, 'dante_user');
      END IF;
   END LOOP;
END
$do$;

SELECT 'DanteGPU databases and user initialization complete.' AS status;
