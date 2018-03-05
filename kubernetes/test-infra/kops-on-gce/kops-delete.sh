#!/bin/bash
export KOPS_STATE_STORE="gs://${BUCKET_NAME}"
export KOPS_FEATURE_FLAGS=AlphaAllowGCE

# Auth for kops
echo $GOOGLE_CREDENTIALS > $TMP_CREDS_PATH
export GOOGLE_APPLICATION_CREDENTIALS=$TMP_CREDS_PATH

# Auth for gsutil/gcloud
gcloud auth activate-service-account $(echo $GOOGLE_CREDENTIALS | jq -r .client_email) --key-file=$TMP_CREDS_PATH
gcloud config set pass_credentials_to_gsutil true

kops delete cluster \
  --name=${CLUSTER_NAME} \
  --state=${KOPS_STATE_STORE} \
  --yes && \
gsutil rm -r ${KOPS_STATE_STORE}
