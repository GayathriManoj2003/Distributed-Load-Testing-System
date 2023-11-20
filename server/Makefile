.PHONY: all clean test

# Define binaries
DRIVER_BIN = bin/driver
ORCHESTRATOR_BIN = bin/orchestrator

# Build targets
all: $(DRIVER_BIN) $(ORCHESTRATOR_BIN)

$(DRIVER_BIN): driver/*.go
	go build -o $(DRIVER_BIN) ./driver

$(ORCHESTRATOR_BIN): orchestrator/*.go
	go build -o $(ORCHESTRATOR_BIN) ./orchestrator

# Run targets
run-driver: $(DRIVER_BIN)
	$(DRIVER_BIN)

run-orchestrator: $(ORCHESTRATOR_BIN)
	$(ORCHESTRATOR_BIN)

# Clean target
clean:
	rm -rf bin/
