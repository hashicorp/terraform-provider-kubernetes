#!/bin/bash
echo "KOPS_STATE_STORE=$KOPS_STATE_STORE"
MAX_RETRIES=$1

RETRIES=1
while [ 1 ]; do
  kops validate cluster
  if [ $? == 0 ]; then
    break
  fi

  sleep 5
  echo "Retrying kube validation... ($RETRIES)"
  ((RETRIES++))
  if [ $RETRIES -gt $MAX_RETRIES ]; then
    echo "Bailing out after $MAX_RETRIES retries"
    exit 1
    break
  fi
done
