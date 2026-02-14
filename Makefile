# Запуск приложения
run:
	docker compose -f docker-compose.yml up -d --build

# Остановка приложения
stop:
	docker compose -f docker-compose.yml down

# Пересборка и перезапуск
rerun:
	docker compose -f docker-compose.yml down
	docker compose -f docker-compose.yml build
	docker compose -f docker-compose.yml up -d

# Генерация Swagger документации
swag:
	~/go/bin/swag init -g main.go
