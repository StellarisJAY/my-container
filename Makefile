
TARGET = mycontainer

build:
	@go build -o $(TARGET)
clear:
	@cgdelete -r cpu:my_container pids:my_container memory:my_container
