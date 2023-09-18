#!/bin/sh
/tern migrate --migrations /app/migrations --config \
        /app/migrations/tern.conf && /app/app