PROTO_SOURCES := $(wildcard **/*.proto)

all:

sync:
	@./sync.sh

fmt:
	@-clang-format -i $(PROTO_SOURCES)

clean:
	@rm $(filter-out $(wildcard **/reduced_*.proto), $(PROTO_SOURCES))
