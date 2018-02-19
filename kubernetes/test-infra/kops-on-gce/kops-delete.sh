#!/bin/bash
kops delete cluster \
  --name=${CLUSTER_NAME} \
  --state=gs://${BUCKET_NAME} \
  --yes && \
gsutil rm -r gs://${BUCKET_NAME}
