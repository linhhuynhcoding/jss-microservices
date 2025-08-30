run-app:
	docker compose \
		-f ./market-service/docker-compose.yaml \
		-f ./product-customer-service/docker-compose.yaml \
		--project-directory . \
		up --build -d
# 		-f docker-compose.yml \

PHOYE: run-app