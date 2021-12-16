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


docker run -d --name redis-main --restart=unless-stopped -p 6379:6379 redis:latest
docker run -d -p 5672:5672 --name messagebrocker rabbitmq:latest

#sleep 10

docker run -d -p 8080:8080 --restart=unless-stopped --name gateway gateway:latest
docker run -d -e "FLAGS= --redis 172.17.0.1:6379" --restart=unless-stopped --name worker1 worker:latest
docker run -d -e "FLAGS= --redis 172.17.0.1:6379" --restart=unless-stopped --name worker2 worker:latest
docker run -d --restart=unless-stopped --name stats stats:latest

# wait until all running

until  [ "`docker inspect -f {{.State.Running}}{{.State.Restarting}} redis-main`"=="truefalse" ] \
    && [ "`docker inspect -f {{.State.Running}}{{.State.Restarting}} worker1`"=="truefalse" ] \
    && [ "`docker inspect -f {{.State.Running}}{{.State.Restarting}} worker2`"=="truefalse" ] \
    && [ "`docker inspect -f {{.State.Running}}{{.State.Restarting}} gateway`"=="truefalse" ] \
    && [ "`docker inspect -f {{.State.Running}}{{.State.Restarting}} stats`"=="truefalse" ] 
do
    echo wait
    sleep 5
done

sleep 5

for i in 0 1 2 3 4 5
do
    until  [ "`docker inspect -f {{.State.Running}} redis-main`"=="true" ] \
        && [ "`docker inspect -f {{.State.Running}} worker1`"=="true" ] \
        && [ "`docker inspect -f {{.State.Running}} worker2`"=="true" ] \
        && [ "`docker inspect -f {{.State.Running}} gateway`"=="true" ] \
        && [ "`docker inspect -f {{.State.Running}} stats`"=="true" ] \
        && [ "`docker inspect -f {{.State.Restarting}} redis-main`"=="false" ] \
        && [ "`docker inspect -f {{.State.Restarting}} worker1`"=="false" ] \
        && [ "`docker inspect -f {{.State.Restarting}} worker2`"=="false" ] \
        && [ "`docker inspect -f {{.State.Restarting}} gateway`"=="false" ] \
        && [ "`docker inspect -f {{.State.Restarting}} stats`"=="false" ] 
    do
        echo wait
        sleep 5
    done

sleep 1

done

echo dockerReady
#sleep 30
