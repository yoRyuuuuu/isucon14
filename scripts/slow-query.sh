#!/bin/sh

set -eux

# TODO:リポジトリ名を変更する
REPO="yoRyuuuuu/isucon14"
TITLE="$(date '+%Y-%m-%d %H:%M:%S')のスロークエリログ"
ISSUE_URL=$(gh issue create --repo $REPO --title "$TITLE" --body "")

{
  echo "pt-query-digest:"
  echo "\`\`\`"
  sudo pt-query-digest /var/log/mysql/mysql-slow.log | head -n 300
  echo "\`\`\`"
} > /tmp/pt-query-digest
gh issue comment "$ISSUE_URL" --body-file /tmp/pt-query-digest
