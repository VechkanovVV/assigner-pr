.RECIPEPREFIX := >

.PHONY: help build up down logs test lint clean

help:
>@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build:
>docker-compose build

up:
>docker-compose up -d

down:
>docker-compose down

logs:
>docker-compose logs -f app

test:
>./internal/integration/run_tests.sh

lint:
>golangci-lint run --fix

clean:
>docker-compose down -v

.DEFAULT_GOAL := help