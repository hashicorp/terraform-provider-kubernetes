# Add CentOS ol7_addons repo with VirtualBox
cat << EOF | sudo tee /etc/yum.repos.d/ol7_addons.repo
[ol7_addons]
name=Oracle Linux $releasever Add ons (\$basearch)
baseurl=http://public-yum.oracle.com/repo/OracleLinux/OL7/addons/\$basearch/
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-oracle
gpgcheck=1
enabled=1
EOF
rpm --import http://public-yum.oracle.com/RPM-GPG-KEY-oracle-ol7
yum makecache

# Install dependencies
yum install -y gcc make perl kernel-devel kernel-devel-3.10.0-693.17.1.el7.x86_64

yum install -y VirtualBox-5.2
