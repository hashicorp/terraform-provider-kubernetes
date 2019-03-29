package eks

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
)

type ClusterInfo struct {
	Arn                  string
	Endpoint             string
	CertificateAuthority string
}

func getAwsSession(region string) *session.Session {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))

	return sess
}

func GetEksInfo(cluster, region string) (*ClusterInfo, error) {
	session := getAwsSession(region)
	eksClient := eks.New(session)

	dci := &eks.DescribeClusterInput{Name: aws.String(cluster)}
	dco, err := eksClient.DescribeCluster(dci)
	if err != nil {
		return nil, err
	}

	info := &ClusterInfo{}
	info.Arn = *dco.Cluster.Arn
	info.Endpoint = *dco.Cluster.Endpoint
	info.CertificateAuthority = *dco.Cluster.CertificateAuthority.Data
	return info, nil
}
