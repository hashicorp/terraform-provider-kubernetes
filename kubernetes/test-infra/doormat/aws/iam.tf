provider aws {}

resource "aws_iam_role" "tfprov_kubernetes_gha" {
  name = "tfprov-kubernetes-gha"
  tags = {
    hc-service-uri = "github.com/hashicorp/terraform-provider-kubernetes@event_name=workflow_displatch"
  }
  max_session_duration = 43200 
  assume_role_policy   = data.aws_iam_policy_document.tfprov_kubernetes_gha_assume.json
  inline_policy {
    name   = "AdminAccess"
    policy = data.aws_iam_policy_document.tfprov_kubernetes_gha.json
  }
}

data "aws_iam_policy_document" "tfprov_kubernetes_gha_assume" {
  statement {
    actions = [
      "sts:AssumeRole",
      "sts:SetSourceIdentity",
      "sts:TagSession"
    ]
    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::397512762488:user/doormatServiceUser"] # infrasec_prod
    }
  }
}

data "aws_iam_policy_document" "tfprov_kubernetes_gha" {
  statement {
    actions   = ["*"]
    resources = ["*"]
    effect    = "Allow"
  }
}
