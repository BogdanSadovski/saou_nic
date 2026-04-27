#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
  CREATE DATABASE user_service;
  CREATE DATABASE resume_service;
  CREATE DATABASE github_service;
  CREATE DATABASE interview_service;
  CREATE DATABASE scoring_service;
  CREATE DATABASE report_service;
  CREATE DATABASE notification_service;
  CREATE DATABASE analytics_service;
  CREATE DATABASE admin_service;
EOSQL
