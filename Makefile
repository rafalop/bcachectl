default:
	go mod init bcachectl
	go mod tidy
	go build bcachectl.go
clean:
	rm -f bcachectl
install: bcachectl scripts/ceph-bcache.sh
	install -d ${DESTDIR}/usr/bin
	install bcachectl ${DESTDIR}/usr/bin/
	install scripts/ceph-bcache.sh ${DESTDIR}/usr/bin/ceph-bcache
uninstall:
	rm -f ${DESTDIR}/usr/bin/bcachectl
	rm -f ${DESTDIR}/usr/bin/ceph-bcache
.PHONY:
	clean install
