#!/usr/bin/env sh
set -e

git_tag=`git describe --exact-match 2> /dev/null || echo ""`
if [ "$git_tag" != "" ]; then
    echo "Releasing $git_tag!"
    mv ./bin/spsrv-darwin-amd64.tar.gz ./bin/spsrv-darwin-amd64-$git_tag.tar.gz
    mv ./bin/spsrv-darwin-arm64.tar.gz ./bin/spsrv-darwin-arm64-$git_tag.tar.gz
    mv ./bin/spsrv-linux-amd64.tar.gz ./bin/spsrv-linux-amd64-$git_tag.tar.gz
    mv ./bin/spsrv-linux-arm64.tar.gz ./bin/spsrv-linux-arm64-$git_tag.tar.gz

    curl -H"Authorization: token $SRHT_TOKEN" https://git.sr.ht/api/~hedy/repos/spsrv/artifacts/$git_tag -F "file=@./bin/spsrv-darwin-amd64-$git_tag.tar.gz"
    curl -H"Authorization: token $SRHT_TOKEN" https://git.sr.ht/api/~hedy/repos/spsrv/artifacts/$git_tag -F "file=@./bin/spsrv-darwin-arm64-$git_tag.tar.gz"
    curl -H"Authorization: token $SRHT_TOKEN" https://git.sr.ht/api/~hedy/repos/spsrv/artifacts/$git_tag -F "file=@./bin/spsrv-linux-amd64-$git_tag.tar.gz"
    curl -H"Authorization: token $SRHT_TOKEN" https://git.sr.ht/api/~hedy/repos/spsrv/artifacts/$git_tag -F "file=@./bin/spsrv-linux-arm64-$git_tag.tar.gz"
    echo ""
    echo "DONE!"
else
    echo "Non-tagged commit, not releasing!"
fi
