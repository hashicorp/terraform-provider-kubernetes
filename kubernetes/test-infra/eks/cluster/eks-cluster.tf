resource "random_id" "cluster_name" {
  byte_length = 10
}

#
# EKS Cluster Resources
#  * IAM Role to allow EKS service to manage other AWS services
#  * EC2 Security Group to allow networking traffic with EKS cluster
#  * EKS Cluster
#

resource "aws_iam_role" "k8s-acc-cluster" {
  name = "terraform-eks-k8s-acc-cluster"

  assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "eks.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
POLICY
}

resource "aws_iam_role_policy_attachment" "k8s-acc-cluster-AmazonEKSClusterPolicy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
  role = "${aws_iam_role.k8s-acc-cluster.name}"
}

resource "aws_iam_role_policy_attachment" "k8s-acc-cluster-AmazonEKSServicePolicy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSServicePolicy"
  role = "${aws_iam_role.k8s-acc-cluster.name}"
}

resource "aws_security_group" "k8s-acc-cluster" {
  name = "terraform-eks-k8s-acc-cluster"
  description = "Cluster communication with worker nodes"
  vpc_id = "${aws_vpc.k8s-acc.id}"

  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    Name = "terraform-eks-k8s-acc"
  }
}

resource "aws_security_group_rule" "k8s-acc-cluster-ingress-node-https" {
  description = "Allow pods to communicate with the cluster API Server"
  from_port = 443
  protocol = "tcp"
  security_group_id = "${aws_security_group.k8s-acc-cluster.id}"
  source_security_group_id = "${aws_security_group.k8s-acc-node.id}"
  to_port = 443
  type = "ingress"
}

resource "aws_security_group_rule" "k8s-acc-cluster-ingress-workstation-https" {
  cidr_blocks = ["${local.workstation-external-cidr}"]
  description = "Allow workstation to communicate with the cluster API Server"
  from_port = 443
  protocol = "tcp"
  security_group_id = "${aws_security_group.k8s-acc-cluster.id}"
  to_port = 443
  type = "ingress"
}

resource "aws_eks_cluster" "k8s-acc" {
  name = "${local.cluster-name}"
  role_arn = "${aws_iam_role.k8s-acc-cluster.arn}"
  version = "${var.kubernetes_version}"

  vpc_config {
    security_group_ids = ["${aws_security_group.k8s-acc-cluster.id}"]
    subnet_ids = ["${aws_subnet.k8s-acc.*.id}"]
  }

  depends_on = [
    "aws_iam_role_policy_attachment.k8s-acc-cluster-AmazonEKSClusterPolicy",
    "aws_iam_role_policy_attachment.k8s-acc-cluster-AmazonEKSServicePolicy",
  ]

  provisioner "local-exec" {
    command = <<CMDEOF
cat <<EOF > ${path.module}/cluster_ca.pem
${base64decode(aws_eks_cluster.k8s-acc.certificate_authority.0.data)}
EOF
while ! curl -s --cacert ${path.module}/cluster_ca.pem ${aws_eks_cluster.k8s-acc.endpoint}/version > /dev/null; do 
  echo "Waiting for the cluster API to come online..."
  sleep 3
done
CMDEOF
  working_dir = "${path.module}"
}
}

resource "local_file" "kubeconfig" {
  content  = "${local.kubeconfig}"
  filename = "${path.module}/../kubeconfig"
}
