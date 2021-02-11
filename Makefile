update-vendor:
	go mod vendor
	cp -r vendor/* ../caddy2-tmpdocker-vendor
	cd ../caddy2-tmpdocker-vendor \
		&& git add -A \
		&& git commit -m 'update vendor' \
		&& git push
