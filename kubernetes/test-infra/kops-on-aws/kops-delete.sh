#!/bin/bash
kops delete cluster \
  --name=${CLUSTER_NAME} \
  --state=s3://${BUCKET_NAME} \
  --yes
