package aws

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func dataSourceAwsIAMServerCertificate() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsIAMServerCertificateRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validation.StringLenBetween(0, 128),
			},

			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.StringLenBetween(0, 128-resource.UniqueIDSuffixLength),
			},

			"path_prefix": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"latest": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"path": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"expiration_date": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"upload_date": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"certificate_body": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"certificate_chain": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

type certificateByExpiration []*iam.ServerCertificateMetadata

func (m certificateByExpiration) Len() int {
	return len(m)
}

func (m certificateByExpiration) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m certificateByExpiration) Less(i, j int) bool {
	return m[i].Expiration.After(*m[j].Expiration)
}

func dataSourceAwsIAMServerCertificateRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	var matcher = func(cert *iam.ServerCertificateMetadata) bool {
		return strings.HasPrefix(aws.StringValue(cert.ServerCertificateName), d.Get("name_prefix").(string))
	}
	if v, ok := d.GetOk("name"); ok {
		matcher = func(cert *iam.ServerCertificateMetadata) bool {
			return aws.StringValue(cert.ServerCertificateName) == v.(string)
		}
	}

	var metadatas []*iam.ServerCertificateMetadata
	input := &iam.ListServerCertificatesInput{}
	if v, ok := d.GetOk("path_prefix"); ok {
		input.PathPrefix = aws.String(v.(string))
	}
	log.Printf("[DEBUG] Reading IAM Server Certificate")
	err := iamconn.ListServerCertificatesPages(input, func(p *iam.ListServerCertificatesOutput, lastPage bool) bool {
		for _, cert := range p.ServerCertificateMetadataList {
			if matcher(cert) {
				metadatas = append(metadatas, cert)
			}
		}
		return true
	})
	if err != nil {
		return fmt.Errorf("Error describing certificates: %s", err)
	}

	if len(metadatas) == 0 {
		return fmt.Errorf("Search for AWS IAM server certificate returned no results")
	}
	if len(metadatas) > 1 {
		if !d.Get("latest").(bool) {
			return fmt.Errorf("Search for AWS IAM server certificate returned too many results")
		}

		sort.Sort(certificateByExpiration(metadatas))
	}

	metadata := metadatas[0]
	d.SetId(*metadata.ServerCertificateId)
	d.Set("arn", *metadata.Arn)
	d.Set("path", *metadata.Path)
	d.Set("name", *metadata.ServerCertificateName)
	if metadata.Expiration != nil {
		d.Set("expiration_date", metadata.Expiration.Format(time.RFC3339))
	}

	log.Printf("[DEBUG] Get Public Key Certificate for %s", *metadata.ServerCertificateName)
	serverCertificateResp, err := iamconn.GetServerCertificate(&iam.GetServerCertificateInput{
		ServerCertificateName: metadata.ServerCertificateName,
	})
	if err != nil {
		return err
	}
	d.Set("upload_date", serverCertificateResp.ServerCertificate.ServerCertificateMetadata.UploadDate.Format(time.RFC3339))
	d.Set("certificate_body", aws.StringValue(serverCertificateResp.ServerCertificate.CertificateBody))
	d.Set("certificate_chain", aws.StringValue(serverCertificateResp.ServerCertificate.CertificateChain))

	return nil
}
