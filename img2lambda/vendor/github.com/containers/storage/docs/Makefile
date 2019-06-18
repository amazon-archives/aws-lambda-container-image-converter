GOMD2MAN = go-md2man

docs: $(patsubst %.md,%.1,$(filter-out %.5.md,$(wildcard *.md))) containers-storage.conf.5

%.1: %.md
	$(GOMD2MAN) -in $^ -out $@

containers-storage.conf.5: containers-storage.conf.5.md
	$(GOMD2MAN) -in $^ -out $@
