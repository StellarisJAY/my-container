
TARGET = my-container

build:
	@go build -o $(TARGET)
clear:
	@rm -r /var/lib/my-container
	@rm -r /var/run/my-container
	@cgdelete -r cpu:my_container pids:my_container memory:my_container
