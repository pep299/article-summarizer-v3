.PHONY: fmt lint setup clean

fmt:
	terraform fmt

lint:
	terraform fmt -check
	terraform validate
	tflint
	checkov -d .

clean:
	rm -rf .tflint.d/
	rm -f .terraform.lock.hcl
	rm -rf .terraform/

all: fmt lint