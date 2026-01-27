#!/bin/bash

# MySQL initialization wrapper script
# This script ensures proper execution of init.sql

mysql -u root -p"$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE" < /docker-entrypoint-initdb.d/init.sql
