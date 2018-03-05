MAX_RETRIES=$1

RETRIES=1
while [ 1 ]; do
  kops validate cluster --state=s3://${BUCKET_NAME}
  if [ $? == 0 ]; then
    break
  fi

  sleep 5
  echo "Retrying kube validation... ($RETRIES)"
  ((RETRIES++))
  if [ $RETRIES -gt $MAX_RETRIES ]; then
    echo "Bailing out after $MAX_RETRIES retries"
    break
  fi
done

RETRIES=1
KUBE_HOST=$(kubectl config view -o jsonpath="{.clusters[?(@.name == \"${CLUSTER_NAME}\")].cluster.server}")
while [ 1 ]; do
  echo "Trying to resolve $KUBE_HOST"
  host $KUBE_HOST
  if [ $? == 0 ]; then
    break
  fi

  sleep 5
  echo "Retrying DNS query... ($RETRIES)"
  ((RETRIES++))
  if [ $RETRIES -gt $MAX_RETRIES ]; then
    echo "Bailing out after $MAX_RETRIES retries"
    break
  fi
done
