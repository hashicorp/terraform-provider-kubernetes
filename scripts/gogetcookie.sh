#!/bin/bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

eval 'set +o history' 2>/dev/null || setopt HIST_IGNORE_SPACE 2>/dev/null
touch ~/.gitcookies
chmod 0600 ~/.gitcookies

git config --global http.cookiefile ~/.gitcookies

tr , \\t <<\__END__ >>~/.gitcookies
go.googlesource.com,FALSE,/,TRUE,2147483647,o,git-alex.somesan.gmail.com=1/pqqFwLWAqtOsvsLsNE1FMgdyffwTGwNGvvax4yuG50k924qulFkcNacHhpDR-Kq-
go-review.googlesource.com,FALSE,/,TRUE,2147483647,o,git-alex.somesan.gmail.com=1/pqqFwLWAqtOsvsLsNE1FMgdyffwTGwNGvvax4yuG50k924qulFkcNacHhpDR-Kq-
__END__
eval 'set -o history' 2>/dev/null || unsetopt HIST_IGNORE_SPACE 2>/dev/null
