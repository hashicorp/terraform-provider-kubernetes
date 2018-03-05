#!/bin/bash
export NODE_SIZE=n1-standard-1
export MASTER_SIZE=n1-standard-2
export KOPS_STATE_STORE="gs://${BUCKET_NAME}"
export KOPS_FEATURE_FLAGS=AlphaAllowGCE

# Auth for kops
echo $GOOGLE_CREDENTIALS > $TMP_CREDS_PATH
export GOOGLE_APPLICATION_CREDENTIALS=$TMP_CREDS_PATH

# Auth for gsutil/gcloud
gcloud auth activate-service-account $(echo $GOOGLE_CREDENTIALS | jq -r .client_email) --key-file=$TMP_CREDS_PATH
gcloud config set pass_credentials_to_gsutil true

gsutil mb -l $GOOGLE_REGION -p $GOOGLE_PROJECT gs://${BUCKET_NAME} && \
kops create cluster --cloud=gce \
  --name=$CLUSTER_NAME \
  --state=$KOPS_STATE_STORE \
  --zones $ZONES \
  --master-zones $ZONES \
  --node-count=2 \
  --project $GOOGLE_PROJECT \
  --image "ubuntu-os-cloud/ubuntu-1604-xenial-v20170202" \
  --kubernetes-version=$KUBERNETES_VERSION \
  --ssh-public-key=${SSH_PUBKEY_PATH} \
  --ssh-access=${IP_ADDRESS}/32 \
  --admin-access=${IP_ADDRESS}/32 \
  --yes

EXIT_CODE=$?
if [ $EXIT_CODE == 0 ]; then
  ../kops-waiter.sh 120
else
  exit $EXIT_CODE
fi
