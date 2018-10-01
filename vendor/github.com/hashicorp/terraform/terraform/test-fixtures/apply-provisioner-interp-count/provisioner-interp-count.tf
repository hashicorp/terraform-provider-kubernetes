variable "count" {
  default = 3
}

resource "aws_instance" "a" {
  count = "${var.count}"
}

resource "aws_instance" "b" {
  provisioner "local-exec" {
    # Since we're in a provisioner block here, this interpolation is
    # resolved during the apply walk and so the resource count must
    # be interpolated during that walk, even though apply walk doesn't
    # do DynamicExpand.
    command = "echo ${aws_instance.a.count}"
  }
}
