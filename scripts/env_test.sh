#!/usr/bin/env bash
set -e

# .env laden
set -o allexport
source .env
set +o allexport

# Beispiel: Variablen nutzen
echo "Backend: $BACKEND_HOST"
