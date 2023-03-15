#!/bin/sh

package_name="gosma"
buildstamp=$(date -I)-$(git rev-parse --short HEAD)
platforms=("linux/amd64" "linux/arm64" "windows/amd64" "darwin/amd64" "darwin/arm64")

for platform in "${platforms[@]}"
do
	platform_split=(${platform//\// })
	GOOS=${platform_split[0]}
	GOARCH=${platform_split[1]}
	output_name="bin/$package_name-$GOOS-$GOARCH"
	if [ $GOOS = "windows" ]; then
		output_name+='.exe'
	fi	

	GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 \
    go build -o $output_name -ldflags "-X github.com/k3a/gosma/config.Buildstamp=$buildstamp" .
	if [ $? -ne 0 ]; then
   		echo 'An error has occurred!'
		exit 1
	fi
done
