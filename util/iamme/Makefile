CFLAGS=-Wall -Wextra -Wno-missing-braces -g -O2

iamme: iamme.c
	$(CC) $(CFLAGS) $^ -o $@

.PHONY: install
install: iamme
	cp /usr/local/bin/iamme

.PHONY: clean
clean:
	rm -f iamme

