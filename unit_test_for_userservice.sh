docker-compose -f docker-compose.yml up --build –d

docker exec -it $(shell docker ps -q --filter "name=orcherstrator_news_service_1") python tests.py