#!/bin/sh

set -eux

# TODO:リポジトリ名を変更する
REPO="yoRyuuuuu/isucon14"
TITLE="$(date '+%Y-%m-%d %H:%M:%S')の計測結果"
ISSUE_URL=$(gh issue create --repo $REPO --title "$TITLE" --body "")

# TODO: alpのオプションに、適切なURI matching groupsを設定する
# https://github.com/tkuchiki/alp/blob/main/README.ja.md#uri-matching-groups
# 例: -m '/@\w+,/image/*,/posts/[0-9]+'
{
  echo "alp:"
  echo "\`\`\`"
  sudo cat /var/log/nginx/access.log | alp json \
    --sort sum -r \
    -m "/api/char/rides/\w+,/api/chair/rides/\w+,/api/app/rides/\w+/evaluation,/assets/\w+,/images/\w+" \
    -o count,method,1xx,2xx,3xx,4xx,5xx,uri,min,max,sum,avg
  echo "\`\`\`"
} > /tmp/alp
gh issue comment "$ISSUE_URL" --body-file /tmp/alp
