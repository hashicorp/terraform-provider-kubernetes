TEST?="./..."
ACCTEST?="./test/acceptance"

test: 
	go test -v $(TESTARGS) -timeout=30s -parallel=4 $(TEST)

testacc: 
	go test -count=1 -tags acceptance $(ACCTEST) -v $(TESTARGS) -timeout 120m


.PHONY: test testacc 
