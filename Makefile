
archive_version = $(shell dpkg-parsechangelog --show-field Version | cut -f1 -d-)
archive_name = "bitmark-wallet_${archive_version}.orig.tar.gz"
transform_source = "--transform=s@^@${service}_${archive_version}/@"

release_version := $(shell dpkg-parsechangelog --show-field Version)

all: build_deb

build:
	go build -o bitmark-wallet -buildmode=exe -ldflags "-X main.version=${archive_version}" ./command/...

archive:
	if [ ! -f "../${archive_name}" ]; then \
		go mod download; \
		tar zcf go.tar.gz ${GOPATH}; \
		tar czf "../${archive_name}" --exclude-vcs --exclude='debian' --exclude='vendor' --exclude=".circleci" ${transform_source} .; \
	fi;

build_deb: archive
	dpkg-buildpackage --diff-ignore='.*'

release: archive
	DEBFULLNAME="Jim Yeh" DEBEMAIL=jim@bitmark.com dch -a "release bitmarkd ${archive_version} to launchpad" -u medium -D bionic
	DEBFULLNAME="Jim Yeh" DEBEMAIL=jim@bitmark.com dch -r "" -u medium -D bionic
	debuild -S -sa --diff-ignore='.*'
	dput ppa:bitmark/bitmark-wallet ../bitmark-wallet_${release_version}_source.changes

clean_build:
	rm -f go.tar.gz
	rm -f ../bitmark-wallet_*
	dh clean

