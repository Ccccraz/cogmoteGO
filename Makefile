BIN_DIR := bin
CMD_DIR := cmd

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR) ./$(CMD_DIR)/main.go

clean:
	rm -rf $(BIN_DIR)/*

deps:
	go mod download