cd .\broker\kafka\
go get all
go mod tidy

cd ..\..\broker\mqtt\
go get all
go mod tidy

cd ..\..\broker\nats\
go get all
go mod tidy

cd ..\..\broker\nsq\
go get all
go mod tidy

cd ..\..\broker\pulsar\
go get all
go mod tidy

cd ..\..\broker\rabbitmq\
go get all
go mod tidy

cd ..\..\broker\redis\
go get all
go mod tidy

cd ..\..\broker\rocketmq\
go get all
go mod tidy

cd ..\..\broker\stomp\
go get all
go mod tidy

pause