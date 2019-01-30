#
# EKS Worker Nodes Resources
#  * IAM role allowing Kubernetes actions to access other AWS services
#  * EC2 Security Group to allow networking traffic
#  * Data source to fetch latest EKS worker AMI
#  * AutoScaling Launch Configuration to configure worker instances
#  * AutoScaling Group to launch worker instances
#

resource "aws_iam_role" "k8s-acc-node" {
  name = "terraform-eks-k8s-acc-node"

  assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
POLICY
}

resource "aws_iam_role_policy_attachment" "k8s-acc-node-AmazonEKSWorkerNodePolicy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
  role = "${aws_iam_role.k8s-acc-node.name}"
}

resource "aws_iam_role_policy_attachment" "k8s-acc-node-AmazonEKS_CNI_Policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
  role = "${aws_iam_role.k8s-acc-node.name}"
}

resource "aws_iam_role_policy_attachment" "k8s-acc-node-AmazonEC2ContainerRegistryReadOnly" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
  role = "${aws_iam_role.k8s-acc-node.name}"
}

resource "aws_iam_instance_profile" "k8s-acc-node" {
  name = "terraform-eks-k8s-acc"
  role = "${aws_iam_role.k8s-acc-node.name}"
}

resource "aws_security_group" "k8s-acc-node" {
  name = "terraform-eks-k8s-acc-node"
  description = "Security group for all nodes in the cluster"
  vpc_id = "${aws_vpc.k8s-acc.id}"

  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = "${
    map(
     "Name", "terraform-eks-k8s-acc-node",
     "kubernetes.io/cluster/${local.cluster-name}", "owned",
    )
  }"
}

resource "aws_security_group_rule" "k8s-acc-node-ingress-self" {
  description = "Allow node to communicate with each other"
  from_port = 0
  protocol = "-1"
  security_group_id = "${aws_security_group.k8s-acc-node.id}"
  source_security_group_id = "${aws_security_group.k8s-acc-node.id}"
  to_port = 65535
  type = "ingress"
}

resource "aws_security_group_rule" "k8s-acc-node-ingress-cluster" {
  description = "Allow worker Kubelets and pods to receive communication from the cluster control plane"
  from_port = 1025
  protocol = "tcp"
  security_group_id = "${aws_security_group.k8s-acc-node.id}"
  source_security_group_id = "${aws_security_group.k8s-acc-cluster.id}"
  to_port = 65535
  type = "ingress"
}
resource "aws_security_group_rule" "k8s-acc-node-ssh-public" {
  cidr_blocks = ["0.0.0.0/0"]
  description = "Allow worker nodes to receive ssh connections from everywhere"
  from_port = 22
  protocol = "tcp"
  security_group_id = "${aws_security_group.k8s-acc-node.id}"
  to_port = 22
  type = "ingress"
}


data "aws_ami" "eks-worker" {
  filter {
    name = "name"
    values = ["amazon-eks-node-${aws_eks_cluster.k8s-acc.version}-*"]
  }

  most_recent = true
  owners = ["602401143452"] # Amazon EKS AMI Account ID
}

# EKS currently documents this required userdata for EKS worker nodes to
# properly configure Kubernetes applications on the EC2 instance.
# We utilize a Terraform local here to simplify Base64 encoding this
# information into the AutoScaling Launch Configuration.
# More information: https://docs.aws.amazon.com/eks/latest/userguide/launch-workers.html
locals {
  k8s-acc-node-userdata = <<USERDATA
#!/bin/bash
set -o xtrace
/etc/eks/bootstrap.sh \
  --use-max-pods false \
  --apiserver-endpoint '${aws_eks_cluster.k8s-acc.endpoint}' \
  --b64-cluster-ca '${aws_eks_cluster.k8s-acc.certificate_authority.0.data}' \
  '${local.cluster-name}'
USERDATA
}

data "local_file" "my_ssh_key" {
  filename = "/Users/alex/.ssh/id_rsa.pub"
}

resource "aws_key_pair" "node_access" {
  key_name   = "node_access"
  public_key = "${data.local_file.my_ssh_key.content}"
}

resource "aws_launch_configuration" "k8s-acc" {

  associate_public_ip_address = true
  iam_instance_profile        = "${aws_iam_instance_profile.k8s-acc-node.name}"
  image_id                    = "${data.aws_ami.eks-worker.id}"
  instance_type               = "${var.workers_type}"
  key_name                    = "${aws_key_pair.node_access.key_name}"
  name_prefix                 = "terraform-eks-k8s-acc"
  security_groups             = ["${aws_security_group.k8s-acc-node.id}"]
  user_data_base64            = "${base64encode(local.k8s-acc-node-userdata)}"
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_autoscaling_group" "k8s-acc" {
  desired_capacity     = "${var.workers_count}"
  launch_configuration = "${aws_launch_configuration.k8s-acc.id}"
  max_size             = "10"
  min_size             = 0
  name                 = "terraform-eks-k8s-acc"
  vpc_zone_identifier  = ["${aws_subnet.k8s-acc.*.id}"]

  tag {
    key                 = "Name"
    value               = "terraform-eks-k8s-acc"
    propagate_at_launch = true
  }

  tag {
    key                 = "kubernetes.io/cluster/${local.cluster-name}"
    value               = "owned"
    propagate_at_launch = true
  }
}
