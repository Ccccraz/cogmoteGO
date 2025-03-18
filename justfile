build:
    #!/usr/bin/env sh
    echo 'Building...'
    mkdir -p bin
    go get
    go build -o bin/cogmoteGO cogmoteGO.go

test:
    #!/usr/bin/env sh
    cd test
    uv sync --all-extras --dev
    uv run pytest -v
