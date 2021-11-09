docker pull rabbitmq
docker pull redis

cd gateway
docker build --pull --rm -f "Dockerfile" -t gateway:latest "."
cd ..
cd worker
docker build --pull --rm -f "Dockerfile" -t worker:latest "."
cd ..
cd statistics
docker build --pull --rm -f "Dockerfile" -t stats:latest "."
cd ..

docker run -d -p 5672:5672 --name messagebrocker rabbitmq:latest

docker run -d --name redis-main --restart=always -p 6379:6379 redis:latest

docker run -d -e "FLAGS= --redis 172.17.0.1:6379" --restart=always --name worker1 worker:latest
docker run -d -e "FLAGS= --redis 172.17.0.1:6379" --restart=always --name worker2 worker:latest

docker run -d --restart=always --name stats stats:latest

docker run -d -p 8080:8080 --restart=always --name gateway gateway:latest


