#!/bin/bash

set -e

# Function to generate random string
generate_random() {
  openssl rand -hex 32
}

# 1. Create infracon.db if it doesn't exist
if [ ! -f "infracon.db" ]; then
  touch infracon.db
  echo "Created infracon.db"
else
  echo "infracon.db already exists"
fi

# 2. Create setup-key.txt with random string if it doesn't exist
if [ ! -f "setup-key.txt" ]; then
  RANDOM_KEY=$(generate_random)
  echo "$RANDOM_KEY" > setup-key.txt
  echo "Created setup-key.txt"
else
  echo "setup-key.txt already exists"
fi

# 3. Create .env with JWT_SECRET if it doesn't exist
if [ ! -f ".env" ]; then
  JWT_SECRET=$(generate_random)
  echo "JWT_SECRET=$JWT_SECRET" > .env
  echo "Created .env"
else
  echo ".env already exists"
fi