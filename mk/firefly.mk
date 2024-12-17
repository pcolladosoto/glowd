.PHONY: firefly-start
firefly-start:
	cat ./plugins/fireflyp/testdata/start_firefly.json | nc -u -w 1 127.0.0.1 10514

.PHONY: firefly-end
firefly-end:
	cat ./plugins/fireflyp/testdata/end_firefly.json | nc -u -w 1 127.0.0.1 10514

.PHONY: firefly-listen
firefly-listen:
	nc -u -k -l 10515
