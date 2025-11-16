-- Drop tables in correct order (respecting foreign keys)
DROP TABLE IF EXISTS assignment_stats CASCADE;
DROP TABLE IF EXISTS pr_reviewers CASCADE;
DROP TABLE IF EXISTS pull_requests CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS teams CASCADE;
DROP TABLE IF EXISTS schema_migrations CASCADE;