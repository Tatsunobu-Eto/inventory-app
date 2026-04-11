TAILWIND_VERSION := 4.1.4

.PHONY: tailwind build run clean npm-install

npm-install:
	npm install

tailwind: npm-install
	npm run tailwind

build: tailwind
	go build -o inventory.exe .

run:
	go run .

clean:
	rm -f inventory.exe static/style.css
