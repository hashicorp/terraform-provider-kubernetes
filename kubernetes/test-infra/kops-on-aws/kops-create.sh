#!/bin/bash
export NODE_SIZE=t2.small
export MASTER_SIZE=t2.medium
export KOPS_STATE_STORE="s3://${BUCKET_NAME}"

aws s3api create-bucket --acl=private --bucket $BUCKET_NAME && \
kops create cluster --cloud=aws \
  --name=$CLUSTER_NAME \
  --state=$KOPS_STATE_STORE \
  --zones=$ZONES \
  --node-count=2 \
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
