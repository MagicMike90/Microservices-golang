docker-compose -f docker-compose.yml -f docker-compose.test.yml up --build –d
go run ${PWD}/TestRobot/main.go