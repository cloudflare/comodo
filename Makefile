PWD := $(shell pwd)
export GOPATH := $(PWD)

NAME := comodo

.PHONY: all
all: $(NAME)

.PHONY: $(NAME)
$(NAME): bin/$(NAME)

.PHONY: bin/$(NAME)
bin/$(NAME): ; @GOPATH="${PWD}" go install $(NAME)
