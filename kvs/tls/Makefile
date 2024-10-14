default:
.PHONY: default

gencert:
	@cfssl gencert \
		-initca ca-csr.json | cfssljson -bare ca
	@cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=ca-config.json \
		-profile=server \
		server-csr.json | cfssljson -bare server
	@cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=ca-config.json \
		-profile=client \
		client-csr.json | cfssljson -bare client
	@mv *.pem *.csr ../etc/certs/
.PHONY: gencert
