#!/bin/sh
if ! [ $(git config --global --get safe.directory) ]; then
    echo "Setting safe.directory config to /build"
    git config --global --add safe.directory /build
fi
IKONO_NAME='github.com/iKonoTelecomunicaciones/go'
IKONO_VERSION=$(cat go.mod | grep $IKONO_NAME | awk '{ print $5 }' | head -n1)
GO_LDFLAGS="
    -s -w \
    -X main.Tag=$(git describe --exact-match --tags 2>/dev/null) \
    -X main.Commit=$(git rev-parse HEAD) \
    -X 'main.BuildTime=`date -Iseconds`' \
    -X '$IKONO_NAME.GoModVersion=$IKONO_VERSION' \
"
go clean -modcache
go build -ldflags="$GO_LDFLAGS" -o whatsapp-cloud-bin ./whatsapp-cloud
