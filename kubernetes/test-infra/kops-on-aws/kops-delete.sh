#!/bin/bash
kops delete cluster \
  --name=${CLUSTER_NAME} \
  --state=s3://${BUCKET_NAME} \
  --yes && \
aws s3 rm s3://${BUCKET_NAME} --recursive && \
aws s3api delete-bucket --bucket $BUCKET_NAME
