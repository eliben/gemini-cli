#!/bin/bash

set -eux
set -o pipefail

rm -rf input-db.db
cat create-input-db.sql | sqlite3 input-db.db
