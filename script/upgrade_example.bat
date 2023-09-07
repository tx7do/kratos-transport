cd ..\_example\broker\kafka\
go get all
go mod tidy

cd ..\..\..\_example\broker\mqtt\
go get all
go mod tidy

cd ..\..\..\_example\broker\rabbitmq\
go get all
go mod tidy

cd ..\..\..\_example\broker\redis\
go get all
go mod tidy

cd ..\..\..\_example\server\kafka\
go get all
go mod tidy

cd ..\..\..\_example\server\mqtt\
go get all
go mod tidy

cd ..\..\..\_example\server\rabbitmq\
go get all
go mod tidy

cd ..\..\..\_example\server\websocket\
go get all
go mod tidy

pause