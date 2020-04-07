# tool marcros
CC := go
CCFLAG := 

# path marcros
BIN_PATH := dist
OBJ_PATH := obj
SRC_PATH := src

# compile marcros
TARGET_NAME := tgmail
ifeq ($(OS),Windows_NT)
	TARGET_NAME := $(addsuffix .exe,$(TARGET_NAME))
endif
TARGET := $(BIN_PATH)/$(TARGET_NAME)
MAIN_SRC := main.go

# src files & obj files
SRC := $(foreach x, $(SRC_PATH), $(wildcard $(addprefix $(x)/*,.c*)))
OBJ := $(addprefix $(OBJ_PATH)/, $(addsuffix .o, $(notdir $(basename $(SRC)))))

# clean files list
DISTCLEAN_LIST := $(OBJ)
CLEAN_LIST := $(TARGET) \
			  $(DISTCLEAN_LIST)

# default rule
default: all

# non-phony targets
$(TARGET): $(OBJ)
	$(CC) build $(CCFLAG) -o $@ $?

# phony rules
.PHONY: all
all: $(TARGET)

.PHONY: clean
clean:
	@echo CLEAN $(CLEAN_LIST)
	@rm -f $(CLEAN_LIST)

.PHONY: distclean
distclean:
	@echo CLEAN $(CLEAN_LIST)
	@rm -f $(DISTCLEAN_LIST)

