run-app:
	docker compose \
		-f ./market-service/docker-compose.yaml \
		-f ./product-customer-service/docker-compose.yaml \
		-f ./loyalty-service/docker-compose.yaml \
		--project-directory . \
		up --build -d
# 		-f docker-compose.yml \

down-app:
	docker compose \
		-f ./market-service/docker-compose.yaml \
		-f ./product-customer-service/docker-compose.yaml \
		-f ./loyalty-service/docker-compose.yaml \
		--project-directory . \
		down -v
# 		-f docker-compose.yml \


rm-v:
	rm -rf ./data-product-postgres	
	rm -rf ./data-market-postgres

PHOYE: run-app